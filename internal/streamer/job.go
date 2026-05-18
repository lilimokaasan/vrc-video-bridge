package streamer

import "time"

type OutputFormat string

const (
	FormatHLS OutputFormat = "hls"
	FormatMP4 OutputFormat = "mp4"
)

type JobStatus string

const (
	StatusQueued  JobStatus = "queued"
	StatusRunning JobStatus = "running"
	StatusReady   JobStatus = "ready"
	StatusFailed  JobStatus = "failed"
)

type Job struct {
	ID          string       `json:"id"`
	SourceURL   string       `json:"source_url"`
	Format      OutputFormat `json:"format"`
	Status      JobStatus    `json:"status"`
	PlaybackURL string       `json:"playback_url,omitempty"`
	Error       string       `json:"error,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type CreateJobRequest struct {
	URL    string       `json:"url"`
	Format OutputFormat `json:"format"`
}

type CreateJobResponse struct {
	ID          string    `json:"id"`
	Status      JobStatus `json:"status"`
	StatusURL   string    `json:"status_url"`
	PlaybackURL string    `json:"playback_url,omitempty"`
}
