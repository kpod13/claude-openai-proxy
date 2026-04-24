package autorun

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"testing"
)

var (
	errMockNotFound = errors.New("not found")
)

// mockExec replaces execCommand and execLookPath for the duration of the test.
// cmdFn is used when execCommand is called. lookPath controls whether LookPath
// succeeds (non-empty string = found at that path) or fails (empty string).
func mockExec(
	t *testing.T,
	lookPath string,
	cmdFn func(ctx context.Context, name string, arg ...string) *exec.Cmd,
) {
	t.Helper()

	origCmd := execCommand
	origLook := execLookPath

	if cmdFn != nil {
		execCommand = cmdFn
	}

	execLookPath = func(file string) (string, error) {
		if lookPath == "" {
			return "", fmt.Errorf("%w: %s", errMockNotFound, file)
		}

		return lookPath, nil
	}

	t.Cleanup(func() {
		execCommand = origCmd
		execLookPath = origLook
	})
}

// cmdSuccess returns an execCommand func whose command exits 0 and prints output.
func cmdSuccess(output string) func(context.Context, string, ...string) *exec.Cmd {
	return func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		if output == "" {
			return exec.CommandContext(ctx, "true")
		}

		return exec.CommandContext(ctx, "echo", output)
	}
}

// cmdFail returns an execCommand func whose command exits non-zero and prints output.
func cmdFail(output string) func(context.Context, string, ...string) *exec.Cmd {
	return func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		if output == "" {
			return exec.CommandContext(ctx, "false")
		}

		return exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("echo %s; exit 1", output))
	}
}
