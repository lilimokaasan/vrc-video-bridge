package streamer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
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

func runFFmpegWithProgress(timeout time.Duration, dir, name string, duration time.Duration, onProgress func(time.Duration), args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ffmpegArgs := append([]string{"-progress", "pipe:1", "-nostats"}, args...)
	cmd := exec.CommandContext(ctx, name, ffmpegArgs...)
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if outTime, ok := parseFFmpegOutTime(scanner.Text()); ok && onProgress != nil {
				if duration > 0 && outTime > duration {
					outTime = duration
				}
				onProgress(outTime)
			}
		}
		done <- scanner.Err()
	}()

	waitErr := cmd.Wait()
	scanErr := <-done
	if waitErr != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%s timed out after %s", name, timeout)
		}
		return fmt.Errorf("%s failed: %w: %s", name, waitErr, trimCommandOutput(stderr.String()))
	}
	if scanErr != nil {
		return scanErr
	}
	if duration > 0 && onProgress != nil {
		onProgress(duration)
	}
	return nil
}

func parseFFmpegOutTime(line string) (time.Duration, bool) {
	key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
	if !ok || key != "out_time_ms" {
		return 0, false
	}
	microseconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || microseconds < 0 {
		return 0, false
	}
	return time.Duration(microseconds) * time.Microsecond, true
}

func trimCommandOutput(value string) string {
	const limit = 4000
	if len(value) <= limit {
		return value
	}
	return value[len(value)-limit:]
}
