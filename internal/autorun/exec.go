package autorun

import (
	"os/exec"
)

var (
	execCommand  = exec.CommandContext
	execLookPath = exec.LookPath
)
