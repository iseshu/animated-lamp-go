package ffmpeg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

type ProbeData struct {
	Streams []Stream `json:"streams"`
	Format  Format   `json:"format"`
}

type Stream struct {
	Index         int    `json:"index"`
	CodecType     string `json:"codec_type"`
	CodecName     string `json:"codec_name"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	BitRate       string `json:"bit_rate,omitempty"`
	Channels      int    `json:"channels,omitempty"`
	SampleRate    string `json:"sample_rate,omitempty"`
	DisplayAspect string `json:"display_aspect_ratio,omitempty"`
}

type Format struct {
	Filename   string `json:"filename"`
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
	Size       string `json:"size"`
	BitRate    string `json:"bit_rate"`
}

func runFFprobe(ctx context.Context, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "ffprobe", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffprobe failed: %v (stderr: %s)", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func GetDuration(ctx context.Context, fileURL, iamHeader string) (int, error) {
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
	}
	if iamHeader != "" {
		args = append(args, "-headers", fmt.Sprintf("IAM:%s\r\n", iamHeader))
	}
	args = append(args,
		"-show_entries", "format=duration",
		"-of", "csv=p=0:s=x",
		"-select_streams", "v:0",
		"-i", fileURL,
	)

	out, err := runFFprobe(ctx, args)
	if err != nil {
		return 0, err
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return 0, fmt.Errorf("could not detect duration")
	}

	durFloat, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value '%s': %w", raw, err)
	}

	return int(durFloat), nil
}

func GetDimensions(ctx context.Context, fileURL, iamHeader string) (int, int, error) {
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
	}
	if iamHeader != "" {
		args = append(args, "-headers", fmt.Sprintf("IAM:%s\r\n", iamHeader))
	}
	args = append(args,
		"-show_entries", "stream=width,height",
		"-of", "csv=p=0:s=x",
		"-select_streams", "v:0",
		"-i", fileURL,
	)

	out, err := runFFprobe(ctx, args)
	if err != nil {
		return 1280, 720, nil // fallback standard resolution
	}

	parts := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(parts) >= 2 {
		w, _ := strconv.Atoi(parts[0])
		h, _ := strconv.Atoi(parts[1])
		if w > 0 && h > 0 {
			return w, h, nil
		}
	}

	return 1280, 720, nil
}

func GetMediaInfoJSON(ctx context.Context, fileURL, iamHeader string) ([]byte, error) {
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
	}
	if iamHeader != "" {
		args = append(args, "-headers", fmt.Sprintf("IAM:%s\r\n", iamHeader))
	}
	args = append(args,
		"-show_streams",
		"-show_format",
		"-show_chapters",
		"-of", "json",
		"-i", fileURL,
	)

	return runFFprobe(ctx, args)
}

func FormatMediaInfo(rawJSON []byte) (string, error) {
	var probe ProbeData
	if err := json.Unmarshal(rawJSON, &probe); err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("🎬 **Media Information**\n\n")

	durSec, _ := strconv.ParseFloat(probe.Format.Duration, 64)
	sizeBytes, _ := strconv.ParseInt(probe.Format.Size, 10, 64)
	sizeMB := float64(sizeBytes) / (1024 * 1024)

	sb.WriteString(fmt.Sprintf("⏱ **Duration:** %d seconds\n", int(durSec)))
	sb.WriteString(fmt.Sprintf("📦 **Format:** %s\n", probe.Format.FormatName))
	sb.WriteString(fmt.Sprintf("💾 **Size:** %.2f MB\n", sizeMB))

	for _, s := range probe.Streams {
		if s.CodecType == "video" {
			sb.WriteString(fmt.Sprintf("\n📹 **Video:** %s | %dx%d", s.CodecName, s.Width, s.Height))
			if s.DisplayAspect != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", s.DisplayAspect))
			}
			sb.WriteString("\n")
		} else if s.CodecType == "audio" {
			sb.WriteString(fmt.Sprintf("🔊 **Audio:** %s | %d channels | %s Hz\n", s.CodecName, s.Channels, s.SampleRate))
		} else if s.CodecType == "subtitle" {
			sb.WriteString(fmt.Sprintf("💬 **Subtitle:** %s\n", s.CodecName))
		}
	}

	return sb.String(), nil
}
