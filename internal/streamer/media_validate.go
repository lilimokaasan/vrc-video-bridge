package streamer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type mediaDurationReport struct {
	Video time.Duration
	Audio time.Duration
}

type ffprobeDurationReport struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Duration  string `json:"duration"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func (s *Server) validateMP4Duration(path string, expected time.Duration) error {
	report, err := s.probeMediaDuration(path)
	if err != nil {
		return err
	}
	return validateMediaDuration(report, expected)
}

func (s *Server) probeMediaDuration(path string) (mediaDurationReport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ffprobePath(s.cfg.FFmpegPath),
		"-hide_banner",
		"-v", "error",
		"-show_entries", "stream=codec_type,duration:format=duration",
		"-of", "json",
		path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return mediaDurationReport{}, fmt.Errorf("ffprobe timed out while validating %s", filepath.Base(path))
		}
		return mediaDurationReport{}, fmt.Errorf("ffprobe failed while validating %s: %w: %s", filepath.Base(path), err, trimCommandOutput(stderr.String()))
	}

	var probe ffprobeDurationReport
	if err := json.Unmarshal(stdout.Bytes(), &probe); err != nil {
		return mediaDurationReport{}, err
	}

	var report mediaDurationReport
	var hasVideo bool
	for _, stream := range probe.Streams {
		duration := parseProbeDuration(stream.Duration)
		switch stream.CodecType {
		case "video":
			hasVideo = true
			if duration > report.Video {
				report.Video = duration
			}
		case "audio":
			if duration > report.Audio {
				report.Audio = duration
			}
		}
	}
	formatDuration := parseProbeDuration(probe.Format.Duration)
	if hasVideo && report.Video == 0 && formatDuration > 0 {
		report.Video = formatDuration
	}
	return report, nil
}

func validateMediaDuration(report mediaDurationReport, expected time.Duration) error {
	if report.Video <= 0 {
		return fmt.Errorf("MP4 validation failed: missing video stream")
	}
	if durationTooShort(report.Video, expected) {
		return fmt.Errorf("MP4 validation failed: video stream is too short (%s, expected about %s)", report.Video.Round(time.Millisecond), expected.Round(time.Millisecond))
	}
	if report.Audio > 0 && durationTooShort(report.Video, report.Audio) {
		return fmt.Errorf("MP4 validation failed: video stream is shorter than audio (%s video, %s audio)", report.Video.Round(time.Millisecond), report.Audio.Round(time.Millisecond))
	}
	return nil
}

func durationTooShort(actual, expected time.Duration) bool {
	if actual <= 0 || expected <= 0 {
		return false
	}
	tolerance := 2 * time.Second
	if expected >= time.Minute {
		tolerance = 5 * time.Second
	}
	minimum := expected * 9 / 10
	return actual+tolerance < minimum
}

func parseProbeDuration(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "N/A") {
		return 0
	}
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}

func ffprobePath(ffmpegPath string) string {
	if ffmpegPath == "" || ffmpegPath == "ffmpeg" {
		return "ffprobe"
	}
	dir := filepath.Dir(ffmpegPath)
	base := filepath.Base(ffmpegPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "ffmpeg" {
		return filepath.Join(dir, "ffprobe"+ext)
	}
	return "ffprobe"
}
