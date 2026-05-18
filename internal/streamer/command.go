package streamer

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

func runCommand(timeout time.Duration, dir, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stdout = &stderr
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%s timed out after %s", name, timeout)
		}
		return fmt.Errorf("%s failed: %w: %s", name, err, trimCommandOutput(stderr.String()))
	}
	return nil
}

func trimCommandOutput(value string) string {
	const limit = 4000
	if len(value) <= limit {
		return value
	}
	return value[len(value)-limit:]
}
