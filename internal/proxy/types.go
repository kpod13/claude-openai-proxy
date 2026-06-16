package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	// errContentFormat is returned when message content is neither a string nor an array.
	errContentFormat = errors.New("content: expected a string or an array of parts")
)

// OpenAI-compatible request/response types.

// ChatRequest is the body of POST /v1/chat/completions.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// Message is a single entry in the request messages array. Its content may be a
// plain string or an array of multimodal parts (text + image_url).
type Message struct {
	Role    string  `json:"role"`
	Content Content `json:"content"`
}

// Content holds a message's content normalized into ordered parts. A plain JSON
// string becomes a single text part.
type Content struct {
	Parts []ContentPart
}

// ContentPart is one element of a multimodal content array.
type ContentPart struct {
	Type     string // "text", "image_url", or an unsupported type
	Text     string // for Type == "text"
	ImageURL string // raw image_url.url for Type == "image_url"
}

// UnmarshalJSON accepts either a JSON string or an array of content parts.
func (c *Content) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)

	if len(b) == 0 || string(b) == "null" {
		c.Parts = nil

		return nil
	}

	switch b[0] {
	case '"':
		var s string

		err := json.Unmarshal(b, &s)
		if err != nil {
			return fmt.Errorf("content: %w", err)
		}

		c.Parts = []ContentPart{{Type: "text", Text: s}}

		return nil
	case '[':
		var raw []struct {
			Type     string `json:"type"`
			Text     string `json:"text"`
			ImageURL *struct {
				URL string `json:"url"`
			} `json:"image_url"`
		}

		err := json.Unmarshal(b, &raw)
		if err != nil {
			return fmt.Errorf("content: %w", err)
		}

		parts := make([]ContentPart, 0, len(raw))
		for _, p := range raw {
			part := ContentPart{Type: p.Type, Text: p.Text}
			if p.ImageURL != nil {
				part.ImageURL = p.ImageURL.URL
			}

			parts = append(parts, part)
		}

		c.Parts = parts

		return nil
	default:
		return errContentFormat
	}
}

// text returns the concatenation of all text parts.
func (c *Content) text() string {
	var sb strings.Builder

	for _, p := range c.Parts {
		if p.Type == "text" {
			sb.WriteString(p.Text)
		}
	}

	return sb.String()
}

// textContent builds a Content holding a single text part (used in tests and
// when constructing internal messages).
func textContent(s string) Content {
	return Content{Parts: []ContentPart{{Type: "text", Text: s}}}
}

// OutputMessage is the assistant message in a response (content is always text).
type OutputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse is the non-streaming response body.
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice is a single completion choice in a non-streaming response.
type Choice struct {
	Index        int           `json:"index"`
	Message      OutputMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// Usage holds token counts.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletionChunk is a single SSE chunk in a streaming response.
type ChatCompletionChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ChunkChoice `json:"choices"`
}

// ChunkChoice is a single choice delta in a streaming chunk.
type ChunkChoice struct {
	Index        int     `json:"index"`
	Delta        Delta   `json:"delta"`
	FinishReason *string `json:"finish_reason"`
}

// Delta carries the incremental content in a streaming chunk.
type Delta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// ModelList is the response body for GET /v1/models.
type ModelList struct {
	Object string        `json:"object"`
	Data   []ModelObject `json:"data"`
}

// ModelObject is a single model entry.
type ModelObject struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}
