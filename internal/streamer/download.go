package streamer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DirectDownloadOptions struct {
	URL       string
	Format    OutputFormat
	OutputDir string
}

type DirectDownloadResult struct {
	Job         Job
	OutputPath  string
	DirectURL   string
	PlaybackURL string
}

func (s *Server) DirectDownload(opts DirectDownloadOptions) (*DirectDownloadResult, error) {
	sourceURL, err := s.normalizeSourceURL(opts.URL)
	if err != nil {
		return nil, err
	}
	opts.URL = sourceURL
	if opts.Format == "" {
		opts.Format = FormatMP4
	}
	if opts.Format != FormatMP4 && opts.Format != FormatHLS {
		return nil, fmt.Errorf("format must be %q or %q", FormatMP4, FormatHLS)
	}
	if opts.OutputDir == "" {
		opts.OutputDir = "downloads"
	}
	if err := s.validateSourceURL(opts.URL); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	job := &Job{
		ID:        newJobID(),
		SourceURL: opts.URL,
		Format:    opts.Format,
		Status:    StatusRunning,
		CreatedAt: now,
		UpdatedAt: now,
	}

	directURL, err := s.prepareMedia(job, nil)
	if err != nil {
		job.Status = StatusFailed
		job.Error = err.Error()
		job.UpdatedAt = time.Now().UTC()
		return nil, err
	}

	job.Status = StatusReady
	job.UpdatedAt = time.Now().UTC()
	playbackURL, err := s.publishMedia(job)
	if err != nil {
		return nil, err
	}
	job.DirectURL = directURL
	job.PlaybackURL = playbackURL

	workDir := filepath.Join(s.cfg.mediaDir(), job.ID)
	baseName := safeOutputBaseName(opts.URL, job.ID)
	if opts.Format == FormatMP4 {
		sourcePath := filepath.Join(workDir, "video.mp4")
		outputPath := filepath.Join(opts.OutputDir, baseName+".mp4")
		if err := copyFile(sourcePath, outputPath); err != nil {
			return nil, err
		}
		return &DirectDownloadResult{Job: *job, OutputPath: outputPath, DirectURL: directURL, PlaybackURL: playbackURL}, nil
	}

	sourceDir := workDir
	outputPath := filepath.Join(opts.OutputDir, baseName+"-hls")
	if err := copyDir(sourceDir, outputPath); err != nil {
		return nil, err
	}
	return &DirectDownloadResult{Job: *job, OutputPath: filepath.Join(outputPath, "index.m3u8"), DirectURL: directURL, PlaybackURL: playbackURL}, nil
}

func safeOutputBaseName(rawURL, fallback string) string {
	if bvid := bvidPattern.FindString(rawURL); bvid != "" {
		return bvid
	}
	if youtubeID := youtubeVideoID(rawURL); youtubeID != "" {
		return "youtube-" + youtubeID
	}
	value := strings.NewReplacer(
		":", "-",
		"/", "-",
		"\\", "-",
		"?", "-",
		"&", "-",
		"=", "-",
	).Replace(rawURL)
	value = strings.Trim(value, "-. ")
	if value == "" {
		return fallback
	}
	if len(value) > 80 {
		return value[:80]
	}
	return value
}

func copyFile(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	target, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return err
	}
	return target.Close()
}

func copyDir(sourceDir, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".m3u8" && ext != ".ts" {
			continue
		}
		if err := copyFile(filepath.Join(sourceDir, entry.Name()), filepath.Join(targetDir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
