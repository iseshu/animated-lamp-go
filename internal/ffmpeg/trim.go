package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// GenerateSampleVideo extracts a short clip using stream copy (-c copy)
func GenerateSampleVideo(ctx context.Context, fileURL, iamHeader string, startAt, duration int) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "clip-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmpDir) }

	outFile := filepath.Join(tmpDir, "sample.mkv")

	args := []string{
		"-hide_banner",
		"-loglevel", "error",
	}

	if iamHeader != "" {
		args = append(args, "-headers", fmt.Sprintf("IAM:%s\r\n", iamHeader))
	}

	args = append(args,
		"-ss", strconv.Itoa(startAt),
		"-i", fileURL,
		"-t", strconv.Itoa(duration),
		"-map", "0",
		"-c", "copy",
		"-y",
		outFile,
	)

	cmd := exec.CommandContext(ctx, FFmpegBinary(), args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("sample generation failed: %v (stderr: %s)", err, stderr.String())
	}

	fi, err := os.Stat(outFile)
	if err != nil || fi.Size() == 0 {
		cleanup()
		return "", nil, fmt.Errorf("sample file is empty or missing")
	}

	return outFile, cleanup, nil
}

// TrimVideo extracts a specific range [startSec, endSec] using stream copy
func TrimVideo(ctx context.Context, fileURL, iamHeader string, startSec, endSec int) (string, func(), error) {
	if startSec < 0 || endSec <= startSec {
		return "", nil, fmt.Errorf("invalid range: start (%d) must be non-negative and less than end (%d)", startSec, endSec)
	}

	duration := endSec - startSec
	return GenerateSampleVideo(ctx, fileURL, iamHeader, startSec, duration)
}
