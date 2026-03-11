package add

import (
	"context"
	"errors"
	"os/exec"
	"time"
)

func compilePythonScript(interpreterPath, scriptPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, interpreterPath, "-m", "py_compile", scriptPath)
	out, err := cmd.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return string(out), errors.New("timed out")
	}
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}
