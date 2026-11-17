package ffmpeg

import (
	"os"
	"os/exec"
)

// FFmpegBinary returns the path to the ffmpeg executable, checking current directory first
func FFmpegBinary() string {
	if _, err := os.Stat("./ffmpeg"); err == nil {
		return "./ffmpeg"
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p
	}
	return "ffmpeg"
}

// FFprobeBinary returns the path to the ffprobe executable, checking current directory first
func FFprobeBinary() string {
	if _, err := os.Stat("./ffprobe"); err == nil {
		return "./ffprobe"
	}
	if p, err := exec.LookPath("ffprobe"); err == nil {
		return p
	}
	return "ffprobe"
}
