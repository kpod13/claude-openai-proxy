package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	// errUnexpectedRunnerCall marks a runner that must not be invoked in a test.
	errUnexpectedRunnerCall = errors.New("unexpected runner call")
)

// --- Content.UnmarshalJSON ---

func TestContentUnmarshal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		json      string
		wantParts []ContentPart
	}{
		{
			name:      "plain string",
			json:      `"hello"`,
			wantParts: []ContentPart{{Type: "text", Text: "hello"}},
		},
		{
			name:      "null",
			json:      `null`,
			wantParts: nil,
		},
		{
			name: "text array",
			json: `[{"type":"text","text":"a"},{"type":"text","text":"b"}]`,
			wantParts: []ContentPart{
				{Type: "text", Text: "a"},
				{Type: "text", Text: "b"},
			},
		},
		{
			name: "text and image",
			json: `[{"type":"text","text":"look"},{"type":"image_url","image_url":{"url":"data:image/png;base64,aGVsbG8="}}]`,
			wantParts: []ContentPart{
				{Type: "text", Text: "look"},
				{Type: "image_url", ImageURL: "data:image/png;base64,aGVsbG8="},
			},
		},
		{
			name:      "unsupported part captured",
			json:      `[{"type":"input_audio"}]`,
			wantParts: []ContentPart{{Type: "input_audio"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var c Content

			err := json.Unmarshal([]byte(tc.json), &c)
			require.NoError(t, err)
			require.Equal(t, tc.wantParts, c.Parts)
		})
	}
}

func TestContentUnmarshalError(t *testing.T) {
	t.Parallel()

	var c Content

	err := json.Unmarshal([]byte(`42`), &c)
	require.Error(t, err)
}

// --- inspectContent ---

func TestInspectContent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		messages  []Message
		wantImage bool
		wantErr   bool
	}{
		{
			name:      "text only",
			messages:  []Message{{Role: "user", Content: textContent("hi")}},
			wantImage: false,
		},
		{
			name: "has image",
			messages: []Message{{Role: "user", Content: Content{Parts: []ContentPart{
				{Type: "text", Text: "hi"},
				{Type: "image_url", ImageURL: "https://example.com/x.png"},
			}}}},
			wantImage: true,
		},
		{
			name: "unsupported part",
			messages: []Message{{Role: "user", Content: Content{Parts: []ContentPart{
				{Type: "input_audio"},
			}}}},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := inspectContent(tc.messages)
			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantImage, got)
		})
	}
}

// --- buildStreamJSONInput ---

func TestBuildStreamJSONInput(t *testing.T) {
	t.Parallel()

	messages := []Message{
		{Role: "user", Content: Content{Parts: []ContentPart{
			{Type: "text", Text: "what are these"},
			{Type: "image_url", ImageURL: "data:image/png;base64,aGVsbG8="},
			{Type: "image_url", ImageURL: "https://example.com/x.png"},
		}}},
	}

	payload, err := buildStreamJSONInput(messages)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(payload, "\n"), "payload must be newline-terminated")

	var env streamJSONEnvelope

	err = json.Unmarshal([]byte(strings.TrimSpace(payload)), &env)
	require.NoError(t, err)

	require.Equal(t, "user", env.Type)
	require.Equal(t, "user", env.Message.Role)

	// [User] marker, text, base64 image, url image
	require.Len(t, env.Message.Content, 4)
	require.Equal(t, "[User]:", env.Message.Content[0].Text)
	require.Equal(t, "what are these", env.Message.Content[1].Text)

	require.Equal(t, "image", env.Message.Content[2].Type)
	require.Equal(t, "base64", env.Message.Content[2].Source.Type)
	require.Equal(t, "image/png", env.Message.Content[2].Source.MediaType)
	require.Equal(t, "aGVsbG8=", env.Message.Content[2].Source.Data)

	require.Equal(t, "image", env.Message.Content[3].Type)
	require.Equal(t, "url", env.Message.Content[3].Source.Type)
	require.Equal(t, "https://example.com/x.png", env.Message.Content[3].Source.URL)
}

func TestBuildStreamJSONInputErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		messages []Message
	}{
		{
			name:     "bad role",
			messages: []Message{{Role: "tool", Content: textContent("x")}},
		},
		{
			name: "malformed data uri",
			messages: []Message{{Role: "user", Content: Content{Parts: []ContentPart{
				{Type: "image_url", ImageURL: "data:image/png;base64,@@@not-base64@@@"},
			}}}},
		},
		{
			name: "unsupported image url scheme",
			messages: []Message{{Role: "user", Content: Content{Parts: []ContentPart{
				{Type: "image_url", ImageURL: "ftp://example.com/x.png"},
			}}}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := buildStreamJSONInput(tc.messages)
			require.Error(t, err)
		})
	}
}

// --- parseDataURI ---

func TestParseDataURI(t *testing.T) {
	t.Parallel()

	mt, data, err := parseDataURI("data:image/jpeg;base64,aGVsbG8=")
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", mt)
	require.Equal(t, "aGVsbG8=", data)

	_, _, err = parseDataURI("data:image/png,aGVsbG8=") // no ;base64
	require.Error(t, err)

	_, _, err = parseDataURI("data:image/png;base64") // no comma
	require.Error(t, err)

	_, _, err = parseDataURI("data:image/png;base64,@@@") // bad base64
	require.Error(t, err)
}

// --- Handler: image path routing ---

func TestChatCompletions_ImagePath(t *testing.T) {
	t.Parallel()

	var capturedPayload string

	reg := NewRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{
		Registry: reg,
		RunBlockingImages: func(_ context.Context, _, payload string) (*CLIResult, error) {
			capturedPayload = payload

			return &CLIResult{Text: "a red square", InputTokens: 9, OutputTokens: 4}, nil
		},
	}

	body := `{"model":"sonnet","messages":[{"role":"user","content":[` +
		`{"type":"text","text":"what is this"},` +
		`{"type":"image_url","image_url":{"url":"data:image/png;base64,aGVsbG8="}}]}]}`

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.ChatCompletions(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp ChatResponse

	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, "a red square", resp.Choices[0].Message.Content)

	// The image runner received a stream-json payload carrying the image block.
	require.Contains(t, capturedPayload, `"type":"image"`)
	require.Contains(t, capturedPayload, `"data":"aGVsbG8="`)
}

func TestChatCompletions_ImageBadRequest(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{
		Registry: reg,
		RunBlockingImages: func(_ context.Context, _, _ string) (*CLIResult, error) {
			t.Fatal("runner must not be called on a bad request")

			return nil, errUnexpectedRunnerCall
		},
	}

	cases := []struct {
		name string
		body string
	}{
		{
			name: "unsupported part type",
			body: `{"model":"sonnet","messages":[{"role":"user","content":[{"type":"input_audio"}]}]}`,
		},
		{
			name: "malformed data uri",
			body: `{"model":"sonnet","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"data:image/png;base64,@@@"}}]}]}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			h.ChatCompletions(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}
