package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

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
		return nil, fmt.Errorf("claude: no JSON in output")
	}

	var res cliJSONResult
	if err := json.Unmarshal([]byte(raw[start:]), &res); err != nil {
		return nil, fmt.Errorf("claude: parse error: %w", err)
	}

	return &CLIResult{
		Text:         res.Result,
		InputTokens:  res.Usage.InputTokens,
		OutputTokens: res.Usage.OutputTokens,
	}, nil
}

// RunBlocking invokes claude in non-streaming mode and returns the full result.
func RunBlocking(model, prompt string) (*CLIResult, error) {
	cmd := exec.Command("claude",
		"--print",
		"--output-format", "json",
		"--model", model,
		"--no-session-persistence",
	)
	cmd.Stdin = strings.NewReader(prompt)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("claude: %w", err)
	}

	return parseBlockingOutput(out)
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

// RunStreaming invokes claude in streaming mode and returns a channel of text chunks.
// The channel is closed when the stream ends or the context is cancelled.
func RunStreaming(ctx context.Context, model, prompt string) (<-chan StreamChunk, error) {
	cmd := exec.CommandContext(ctx,
		"claude",
		"--print",
		"--output-format", "stream-json",
		"--verbose",
		"--model", model,
		"--no-session-persistence",
	)
	cmd.Stdin = strings.NewReader(prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("claude stream: stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("claude stream: start: %w", err)
	}

	ch := make(chan StreamChunk, 32)

	go func() {
		defer close(ch)
		defer cmd.Wait()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var sl streamLine
			if err := json.Unmarshal([]byte(line), &sl); err != nil {
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

		if err := scanner.Err(); err != nil {
			select {
			case ch <- StreamChunk{Err: err}:
			case <-ctx.Done():
			}
		}
	}()

	return ch, nil
}
