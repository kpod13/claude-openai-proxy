package proxy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	headerContentType     = "Content-Type"
	headerCacheControl    = "Cache-Control"
	headerXAccelBuffering = "X-Accel-Buffering"

	mimeJSON = "application/json"
	mimeSSE  = "text/event-stream"
)

var (
	// errUnsupportedRole is returned when a message has an unrecognised role.
	errUnsupportedRole = errors.New("unsupported message role")

	// errUnsupportedContentPart is returned for a content part that is neither
	// text nor image_url (e.g. input_audio).
	errUnsupportedContentPart = errors.New("unsupported content part type")

	// errMalformedDataURI is returned when an image data: URI cannot be parsed.
	errMalformedDataURI = errors.New("malformed image data URI")

	// errUnsupportedImageURL is returned for an image_url that is neither a
	// data: URI nor an http(s) URL.
	errUnsupportedImageURL = errors.New("unsupported image_url")
)

// Handler holds shared state for the proxy HTTP handlers.
type Handler struct {
	Registry           *Registry
	RunBlocking        func(ctx context.Context, model, prompt string) (*CLIResult, error)
	RunStreaming       func(ctx context.Context, model, prompt string) (<-chan StreamChunk, error)
	RunBlockingImages  func(ctx context.Context, model, payload string) (*CLIResult, error)
	RunStreamingImages func(ctx context.Context, model, payload string) (<-chan StreamChunk, error)
}

// blockingRunner returns the non-streaming runner for the given input kind,
// preferring an injected function and falling back to the package default.
func (h *Handler) blockingRunner(images bool) func(context.Context, string, string) (*CLIResult, error) {
	if images {
		if h.RunBlockingImages != nil {
			return h.RunBlockingImages
		}

		return RunBlockingImages
	}

	if h.RunBlocking != nil {
		return h.RunBlocking
	}

	return RunBlocking
}

// streamingRunner returns the streaming runner for the given input kind.
func (h *Handler) streamingRunner(images bool) func(context.Context, string, string) (<-chan StreamChunk, error) {
	if images {
		if h.RunStreamingImages != nil {
			return h.RunStreamingImages
		}

		return RunStreamingImages
	}

	if h.RunStreaming != nil {
		return h.RunStreaming
	}

	return RunStreaming
}

// Models handles GET /v1/models.
func (h *Handler) Models(w http.ResponseWriter, _ *http.Request) {
	list := ModelList{
		Object: "list",
		Data:   h.Registry.List(),
	}

	w.Header().Set(headerContentType, mimeJSON)

	err := json.NewEncoder(w).Encode(list)
	if err != nil {
		http.Error(w, fmt.Sprintf("encode error: %v", err), http.StatusInternalServerError)

		return
	}
}

// ChatCompletions handles POST /v1/chat/completions.
func (h *Handler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)

		return
	}

	modelID, err := h.Registry.Resolve(req.Model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	hasImages, err := inspectContent(req.Messages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	var stdin string
	if hasImages {
		stdin, err = buildStreamJSONInput(req.Messages)
	} else {
		stdin, err = serializeMessages(req.Messages)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if req.Stream {
		h.handleStreaming(w, r, modelID, stdin, hasImages)
	} else {
		h.handleBlocking(w, r, modelID, stdin, hasImages)
	}
}

// handleBlocking runs a non-streaming completion and writes a ChatResponse.
func (h *Handler) handleBlocking(w http.ResponseWriter, r *http.Request, modelID, stdin string, images bool) {
	result, err := h.blockingRunner(images)(r.Context(), modelID, stdin)
	if err != nil {
		http.Error(w, fmt.Sprintf("claude error: %v", err), http.StatusInternalServerError)

		return
	}

	finishReason := "stop"
	resp := ChatResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   modelID,
		Choices: []Choice{
			{
				Index:        0,
				Message:      OutputMessage{Role: "assistant", Content: result.Text},
				FinishReason: finishReason,
			},
		},
		Usage: Usage{
			PromptTokens:     result.InputTokens,
			CompletionTokens: result.OutputTokens,
			TotalTokens:      result.InputTokens + result.OutputTokens,
		},
	}

	w.Header().Set(headerContentType, mimeJSON)

	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("encode error: %v", err), http.StatusInternalServerError)

		return
	}
}

// handleStreaming runs a streaming completion and writes SSE events.
func (h *Handler) handleStreaming(w http.ResponseWriter, r *http.Request, modelID, stdin string, images bool) {
	w.Header().Set(headerContentType, mimeSSE)
	w.Header().Set(headerCacheControl, "no-cache")
	w.Header().Set(headerXAccelBuffering, "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)

		return
	}

	id := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	created := time.Now().Unix()

	// Send the role delta first.
	sendChunk(w, flusher, &ChatCompletionChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   modelID,
		Choices: []ChunkChoice{
			{Index: 0, Delta: Delta{Role: "assistant"}},
		},
	})

	ch, err := h.streamingRunner(images)(r.Context(), modelID, stdin)
	if err != nil {
		_, _ = fmt.Fprintf(w, "data: {\"error\": %q}\n\n", err.Error())

		flusher.Flush()

		return
	}

	for chunk := range ch {
		if chunk.Err != nil {
			_, _ = fmt.Fprintf(w, "data: {\"error\": %q}\n\n", chunk.Err.Error())

			flusher.Flush()

			return
		}

		sendChunk(w, flusher, &ChatCompletionChunk{
			ID:      id,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   modelID,
			Choices: []ChunkChoice{
				{Index: 0, Delta: Delta{Content: chunk.Text}},
			},
		})
	}

	// Terminating chunk with finish_reason.
	finishReason := "stop"
	sendChunk(w, flusher, &ChatCompletionChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   modelID,
		Choices: []ChunkChoice{
			{Index: 0, Delta: Delta{}, FinishReason: &finishReason},
		},
	})

	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")

	flusher.Flush()
}

// sendChunk serializes a chunk and writes it as an SSE event.
func sendChunk(w http.ResponseWriter, f http.Flusher, chunk *ChatCompletionChunk) {
	data, err := json.Marshal(chunk)
	if err != nil {
		return
	}

	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)

	f.Flush()
}

// rolePrefix returns the conversation prefix for a message role.
func rolePrefix(role string) (string, error) {
	switch role {
	case "system":
		return "[System]: ", nil
	case "user":
		return "[User]: ", nil
	case "assistant":
		return "[Assistant]: ", nil
	default:
		return "", fmt.Errorf("%w: %q", errUnsupportedRole, role)
	}
}

// serializeMessages converts a text-only messages array into a single prompt string.
func serializeMessages(messages []Message) (string, error) {
	var sb strings.Builder

	for _, m := range messages {
		prefix, err := rolePrefix(m.Role)
		if err != nil {
			return "", err
		}

		fmt.Fprintf(&sb, "%s%s\n", prefix, m.Content.text())
	}

	return sb.String(), nil
}

// inspectContent validates content part types and reports whether any message
// carries an image part.
func inspectContent(messages []Message) (bool, error) {
	hasImages := false

	for _, m := range messages {
		for _, p := range m.Content.Parts {
			switch p.Type {
			case "text":
			case "image_url":
				hasImages = true
			default:
				return false, fmt.Errorf("%w: %q", errUnsupportedContentPart, p.Type)
			}
		}
	}

	return hasImages, nil
}

// streamJSONEnvelope is one line of `claude --input-format stream-json` stdin.
type streamJSONEnvelope struct {
	Type    string `json:"type"`
	Message struct {
		Role    string         `json:"role"`
		Content []contentBlock `json:"content"`
	} `json:"message"`
}

// contentBlock is an Anthropic content block (text or image).
type contentBlock struct {
	Type   string       `json:"type"`
	Text   string       `json:"text,omitempty"`
	Source *imageSource `json:"source,omitempty"`
}

// imageSource is the source of an image content block.
type imageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
}

// buildStreamJSONInput flattens the conversation into a single stream-json user
// message whose content blocks preserve text/image order, role-prefixed like the
// text path. The result is one newline-terminated JSON line for the CLI's stdin.
func buildStreamJSONInput(messages []Message) (string, error) {
	var blocks []contentBlock

	for _, m := range messages {
		prefix, err := rolePrefix(m.Role)
		if err != nil {
			return "", err
		}

		blocks = append(blocks, contentBlock{Type: "text", Text: strings.TrimSpace(prefix)})

		for _, p := range m.Content.Parts {
			switch p.Type {
			case "text":
				if p.Text != "" {
					blocks = append(blocks, contentBlock{Type: "text", Text: p.Text})
				}
			case "image_url":
				block, err := imageBlock(p.ImageURL)
				if err != nil {
					return "", err
				}

				blocks = append(blocks, block)
			default:
				return "", fmt.Errorf("%w: %q", errUnsupportedContentPart, p.Type)
			}
		}
	}

	var env streamJSONEnvelope

	env.Type = "user"
	env.Message.Role = "user"
	env.Message.Content = blocks

	b, err := json.Marshal(env)
	if err != nil {
		return "", fmt.Errorf("build stream-json input: %w", err)
	}

	return string(b) + "\n", nil
}

// imageBlock converts an OpenAI image_url value into an Anthropic image block:
// data: URIs become base64 blocks; http(s) URLs become url blocks (the model
// backend fetches them).
func imageBlock(url string) (contentBlock, error) {
	switch {
	case strings.HasPrefix(url, "data:"):
		mediaType, data, err := parseDataURI(url)
		if err != nil {
			return contentBlock{}, err
		}

		return contentBlock{
			Type:   "image",
			Source: &imageSource{Type: "base64", MediaType: mediaType, Data: data},
		}, nil
	case strings.HasPrefix(url, "http://"), strings.HasPrefix(url, "https://"):
		return contentBlock{
			Type:   "image",
			Source: &imageSource{Type: "url", URL: url},
		}, nil
	default:
		return contentBlock{}, fmt.Errorf("%w: %q", errUnsupportedImageURL, url)
	}
}

// parseDataURI parses a base64 image data URI of the form data:<media>;base64,<data>.
func parseDataURI(uri string) (mediaType, data string, err error) {
	header, payload, found := strings.Cut(strings.TrimPrefix(uri, "data:"), ",")
	if !found {
		return "", "", fmt.Errorf("%w: missing comma", errMalformedDataURI)
	}

	if !strings.Contains(header, ";base64") {
		return "", "", fmt.Errorf("%w: not base64-encoded", errMalformedDataURI)
	}

	mediaType = strings.TrimSuffix(header, ";base64")
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}

	_, err = base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", errMalformedDataURI, err)
	}

	return mediaType, payload, nil
}
