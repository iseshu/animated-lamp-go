package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
)

// ExtractFrameToMemory decodes a single frame at the given timestamp and returns it as a JPEG []byte in RAM
func ExtractFrameToMemory(ctx context.Context, fileURL, iamHeader string, sec int, filterOpts string) ([]byte, error) {
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
	}

	if iamHeader != "" {
		args = append(args, "-headers", fmt.Sprintf("IAM:%s\r\n", iamHeader))
	}

	args = append(args,
		"-ss", strconv.Itoa(sec),
		"-i", fileURL,
		"-vframes", "1",
	)

	if filterOpts != "" {
		args = append(args, "-vf", filterOpts)
	}

	// Output: JPEG image streamed straight to stdout (pipe:1)
	args = append(args, "-f", "image2", "-c:v", "mjpeg", "-q:v", "2", "pipe:1")

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg extraction failed: %v (stderr: %s)", err, stderr.String())
	}

	if stdout.Len() == 0 {
		return nil, fmt.Errorf("ffmpeg produced empty output (stderr: %s)", stderr.String())
	}

	return stdout.Bytes(), nil
}
