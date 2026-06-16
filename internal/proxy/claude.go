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

	"github.com/kpod13/claude-openai-proxy/internal/config"
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

// cliError wraps an exec error, appending captured stderr (from
// *exec.ExitError) so subprocess failures are diagnosable.
func cliError(err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
		return fmt.Errorf("claude: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
	}

	return fmt.Errorf("claude: %w", err)
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

// PermissionArgs converts a permission policy into claude CLI flags. The safe
// default (mode "default"/empty, empty lists) yields no flags, leaving the
// invocation identical to a build without a configured policy. Callers must
// pass a validated policy (see config.Config.validate); entries are forwarded
// verbatim as argv elements.
func PermissionArgs(p *config.Permission) []string {
	var args []string

	if p.Mode != "" && p.Mode != config.ModeDefault {
		args = append(args, "--permission-mode", p.Mode)
	}

	if len(p.AllowedTools) > 0 {
		args = append(args, "--allowedTools")
		args = append(args, p.AllowedTools...)
	}

	if len(p.DisallowedTools) > 0 {
		args = append(args, "--disallowedTools")
		args = append(args, p.DisallowedTools...)
	}

	for _, d := range p.AddDirs {
		args = append(args, "--add-dir", d)
	}

	return args
}

// BlockingRunner returns a non-streaming runner that applies the given
// permission flags to every invocation.
func BlockingRunner(perm []string) func(context.Context, string, string) (*CLIResult, error) {
	return func(ctx context.Context, model, prompt string) (*CLIResult, error) {
		return runBlocking(ctx, model, prompt, perm)
	}
}

// RunBlocking invokes claude in non-streaming mode with no permission flags
// (the safe default). It is the package-level default runner.
func RunBlocking(ctx context.Context, model, prompt string) (*CLIResult, error) {
	return runBlocking(ctx, model, prompt, nil)
}

// runBlocking invokes claude in non-streaming mode and returns the full result.
func runBlocking(ctx context.Context, model, prompt string, perm []string) (*CLIResult, error) {
	safeModel, err := sanitizeModelID(model)
	if err != nil {
		return nil, err
	}

	args := append([]string{
		"--print",
		"--output-format", "json",
		"--model", safeModel,
		"--no-session-persistence",
	}, perm...)

	cmd := newCommand(ctx, "claude", args...)
	cmd.Stdin = strings.NewReader(prompt)

	out, err := cmd.Output()
	if err != nil {
		return nil, cliError(err)
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

// BlockingImagesRunner returns a non-streaming image runner that applies the
// given permission flags to every invocation.
func BlockingImagesRunner(perm []string) func(context.Context, string, string) (*CLIResult, error) {
	return func(ctx context.Context, model, payload string) (*CLIResult, error) {
		return runBlockingImages(ctx, model, payload, perm)
	}
}

// RunBlockingImages invokes claude with image input and no permission flags
// (the safe default). It is the package-level default runner.
func RunBlockingImages(ctx context.Context, model, payload string) (*CLIResult, error) {
	return runBlockingImages(ctx, model, payload, nil)
}

// runBlockingImages invokes claude with stream-json input (carrying image
// content blocks) and accumulates the final result. stream-json input requires
// stream-json output, so the result is read from the terminal `result` line.
func runBlockingImages(ctx context.Context, model, payload string, perm []string) (*CLIResult, error) {
	safeModel, err := sanitizeModelID(model)
	if err != nil {
		return nil, err
	}

	args := append([]string{
		"--print",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--model", safeModel,
		"--no-session-persistence",
	}, perm...)

	cmd := newCommand(ctx, "claude", args...)
	cmd.Stdin = strings.NewReader(payload)

	out, err := cmd.Output()
	if err != nil {
		return nil, cliError(err)
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

// StreamingRunner returns a streaming runner that applies the given permission
// flags to every invocation.
func StreamingRunner(perm []string) func(context.Context, string, string) (<-chan StreamChunk, error) {
	return func(ctx context.Context, model, prompt string) (<-chan StreamChunk, error) {
		return runStreaming(ctx, model, prompt, perm)
	}
}

// RunStreaming invokes claude in streaming mode with no permission flags (the
// safe default). It is the package-level default runner.
func RunStreaming(ctx context.Context, model, prompt string) (<-chan StreamChunk, error) {
	return runStreaming(ctx, model, prompt, nil)
}

// runStreaming invokes claude in streaming mode (text input) and returns a
// channel of text chunks. The channel is closed when the stream ends or the
// context is cancelled.
func runStreaming(ctx context.Context, model, prompt string, perm []string) (<-chan StreamChunk, error) {
	safeModel, err := sanitizeModelID(model)
	if err != nil {
		return nil, err
	}

	args := append([]string{
		"--print",
		"--output-format", "stream-json",
		"--verbose",
		"--model", safeModel,
		"--no-session-persistence",
	}, perm...)

	return runClaudeStream(ctx, args, prompt)
}

// StreamingImagesRunner returns a streaming image runner that applies the given
// permission flags to every invocation.
func StreamingImagesRunner(perm []string) func(context.Context, string, string) (<-chan StreamChunk, error) {
	return func(ctx context.Context, model, payload string) (<-chan StreamChunk, error) {
		return runStreamingImages(ctx, model, payload, perm)
	}
}

// RunStreamingImages invokes claude in streaming mode with image input and no
// permission flags (the safe default). It is the package-level default runner.
func RunStreamingImages(ctx context.Context, model, payload string) (<-chan StreamChunk, error) {
	return runStreamingImages(ctx, model, payload, nil)
}

// runStreamingImages invokes claude in streaming mode with stream-json input
// (carrying image content blocks) and returns a channel of text chunks.
func runStreamingImages(ctx context.Context, model, payload string, perm []string) (<-chan StreamChunk, error) {
	safeModel, err := sanitizeModelID(model)
	if err != nil {
		return nil, err
	}

	args := append([]string{
		"--print",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--model", safeModel,
		"--no-session-persistence",
	}, perm...)

	return runClaudeStream(ctx, args, payload)
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
