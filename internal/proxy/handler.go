package proxy

import (
	"context"
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
)

// Handler holds shared state for the proxy HTTP handlers.
type Handler struct {
	Registry    *Registry
	RunBlocking func(ctx context.Context, model, prompt string) (*CLIResult, error)
	RunStreaming func(ctx context.Context, model, prompt string) (<-chan StreamChunk, error)
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

	prompt, err := serializeMessages(req.Messages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if req.Stream {
		h.handleStreaming(w, r, modelID, prompt)
	} else {
		h.handleBlocking(w, r, modelID, prompt)
	}
}

// handleBlocking runs a non-streaming completion and writes a ChatResponse.
func (h *Handler) handleBlocking(w http.ResponseWriter, r *http.Request, modelID, prompt string) {
	runBlocking := h.RunBlocking
	if runBlocking == nil {
		runBlocking = RunBlocking
	}

	result, err := runBlocking(r.Context(), modelID, prompt)
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
				Message:      Message{Role: "assistant", Content: result.Text},
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
func (h *Handler) handleStreaming(w http.ResponseWriter, r *http.Request, modelID, prompt string) {
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

	runStreaming := h.RunStreaming
	if runStreaming == nil {
		runStreaming = RunStreaming
	}

	ch, err := runStreaming(r.Context(), modelID, prompt)
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

// serializeMessages converts the OpenAI messages array into a single prompt string.
func serializeMessages(messages []Message) (string, error) {
	var sb strings.Builder

	for _, m := range messages {
		switch m.Role {
		case "system":
			fmt.Fprintf(&sb, "[System]: %s\n", m.Content)
		case "user":
			fmt.Fprintf(&sb, "[User]: %s\n", m.Content)
		case "assistant":
			fmt.Fprintf(&sb, "[Assistant]: %s\n", m.Content)
		default:
			return "", fmt.Errorf("%w: %q", errUnsupportedRole, m.Role)
		}
	}

	return sb.String(), nil
}
