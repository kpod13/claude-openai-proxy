package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var (
	// errNoJSON is returned when the CLI output contains no JSON object.
	errNoJSON = errors.New("claude: no JSON in output")

	// errInvalidModelID is returned when a model ID contains unexpected characters.
	errInvalidModelID = errors.New("claude: invalid model ID")

	// modelIDRe matches only safe Claude model ID characters (letters, digits, hyphens).
	modelIDRe = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

	// newCommand is the exec.CommandContext factory; replaced in tests.
	newCommand = exec.CommandContext
)

// Version returns the version string of the claude CLI (e.g. "1.2.3").
// It runs `claude --version` and returns the trimmed first line of output.
func Version(ctx context.Context) (string, error) {
	out, err := newCommand(ctx, "claude", "--version").Output()
	if err != nil {
		return "", fmt.Errorf("claude --version: %w", err)
	}

	line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]

	return line, nil
}

// sanitizeModelID validates and returns the model ID, allowing only letters, digits, and hyphens.
// It returns the regex-extracted value so the result is clean from a taint perspective.
func sanitizeModelID(model string) (string, error) {
	safe := modelIDRe.FindString(model)
	if safe != model {
		return "", fmt.Errorf("%w: %q", errInvalidModelID, model)
	}

	return safe, nil
}

// CLIResult holds the output of a non-streaming claude invocation.
type CLIResult struct {
	Text         string
	InputTokens  int
	OutputTokens int
}

// cliJSONResult is the full JSON shape from claude --output-format json.
type cliJSONResult struct {
	Result string `json:"result"`
	Usage  struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// StreamChunk carries a single piece of streamed text.
type StreamChunk struct {
	Text string
	Err  error
}

// parseBlockingOutput parses the raw bytes from a claude --output-format json invocation.
func parseBlockingOutput(out []byte) (*CLIResult, error) {
	raw := strings.TrimSpace(string(out))

	start := strings.Index(raw, "{")
	if start == -1 {
		return nil, errNoJSON
	}

	var res cliJSONResult

	err := json.Unmarshal([]byte(raw[start:]), &res)
	if err != nil {
		return nil, fmt.Errorf("claude: parse error: %w", err)
	}

	return &CLIResult{
		Text:         res.Result,
		InputTokens:  res.Usage.InputTokens,
		OutputTokens: res.Usage.OutputTokens,
	}, nil
}

// RunBlocking invokes claude in non-streaming mode and returns the full result.
func RunBlocking(ctx context.Context, model, prompt string) (*CLIResult, error) {
	safeModel, err := sanitizeModelID(model)
	if err != nil {
		return nil, err
	}

	cmd := newCommand(ctx,
		"claude",
		"--print",
		"--output-format", "json",
		"--model", safeModel,
		"--no-session-persistence",
	)
	cmd.Stdin = strings.NewReader(prompt)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("claude: %w", err)
	}

	return parseBlockingOutput(out)
}

// streamResultLine is the terminal `result` line from stream-json output.
type streamResultLine struct {
	Type   string `json:"type"`
	Result string `json:"result"`
	Usage  struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// RunBlockingImages invokes claude with stream-json input (carrying image
// content blocks) and accumulates the final result. stream-json input requires
// stream-json output, so the result is read from the terminal `result` line.
func RunBlockingImages(ctx context.Context, model, payload string) (*CLIResult, error) {
	safeModel, err := sanitizeModelID(model)
	if err != nil {
		return nil, err
	}

	cmd := newCommand(ctx,
		"claude",
		"--print",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--model", safeModel,
		"--no-session-persistence",
	)
	cmd.Stdin = strings.NewReader(payload)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("claude: %w", err)
	}

	return parseStreamJSONResult(out)
}

// parseStreamJSONResult extracts the terminal `result` line from stream-json output.
func parseStreamJSONResult(out []byte) (*CLIResult, error) {
	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var rl streamResultLine

		err := json.Unmarshal([]byte(line), &rl)
		if err != nil {
			continue
		}

		if rl.Type == "result" {
			return &CLIResult{
				Text:         rl.Result,
				InputTokens:  rl.Usage.InputTokens,
				OutputTokens: rl.Usage.OutputTokens,
			}, nil
		}
	}

	return nil, errNoJSON
}

// streamLine is the shape of a single line from claude --output-format stream-json.
type streamLine struct {
	Type    string `json:"type"`
	Message *struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
}

// RunStreaming invokes claude in streaming mode (text input) and returns a
// channel of text chunks. The channel is closed when the stream ends or the
// context is cancelled.
func RunStreaming(ctx context.Context, model, prompt string) (<-chan StreamChunk, error) {
	safeModel, err := sanitizeModelID(model)
	if err != nil {
		return nil, err
	}

	return runClaudeStream(ctx, []string{
		"--print",
		"--output-format", "stream-json",
		"--verbose",
		"--model", safeModel,
		"--no-session-persistence",
	}, prompt)
}

// RunStreamingImages invokes claude in streaming mode with stream-json input
// (carrying image content blocks) and returns a channel of text chunks.
func RunStreamingImages(ctx context.Context, model, payload string) (<-chan StreamChunk, error) {
	safeModel, err := sanitizeModelID(model)
	if err != nil {
		return nil, err
	}

	return runClaudeStream(ctx, []string{
		"--print",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--model", safeModel,
		"--no-session-persistence",
	}, payload)
}

// runClaudeStream starts `claude` with the given args, feeds stdin, and streams
// assistant text blocks parsed from its stream-json output.
func runClaudeStream(ctx context.Context, args []string, stdin string) (<-chan StreamChunk, error) {
	cmd := newCommand(ctx, "claude", args...)
	cmd.Stdin = strings.NewReader(stdin)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("claude stream: stdout pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("claude stream: start: %w", err)
	}

	ch := make(chan StreamChunk, 32)

	go func() {
		defer close(ch)
		defer func() { _ = cmd.Wait() }()

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var sl streamLine

			err := json.Unmarshal([]byte(line), &sl)
			if err != nil {
				continue // skip unparseable lines
			}

			if sl.Type == "assistant" && sl.Message != nil {
				for _, block := range sl.Message.Content {
					if block.Type == "text" && block.Text != "" {
						select {
						case ch <- StreamChunk{Text: block.Text}:
						case <-ctx.Done():
							return
						}
					}
				}
			}

			if sl.Type == "result" {
				return
			}
		}

		err := scanner.Err()
		if err != nil {
			select {
			case ch <- StreamChunk{Err: err}:
			case <-ctx.Done():
			}
		}
	}()

	return ch, nil
}
