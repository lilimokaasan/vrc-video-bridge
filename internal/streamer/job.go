package streamer

import "time"

type OutputFormat string

const (
	FormatHLS OutputFormat = "hls"
	FormatMP4 OutputFormat = "mp4"
)

func (f OutputFormat) String() string {
	return string(f)
}

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
	Message     string       `json:"message,omitempty"`
	Progress    JobProgress  `json:"progress,omitempty"`
	DirectURL   string       `json:"direct_url,omitempty"`
	PlaybackURL string       `json:"playback_url,omitempty"`
	Error       string       `json:"error,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type JobProgress map[string]ProgressStep

type ProgressStep struct {
	Label      string `json:"label"`
	State      string `json:"state"`
	Percent    int    `json:"percent,omitempty"`
	BytesDone  int64  `json:"bytes_done,omitempty"`
	BytesTotal int64  `json:"bytes_total,omitempty"`
	Message    string `json:"message,omitempty"`
}

type CreateJobRequest struct {
	URL    string       `json:"url"`
	Format OutputFormat `json:"format"`
}

type CreateJobResponse struct {
	ID          string    `json:"id"`
	Status      JobStatus `json:"status"`
	StatusURL   string    `json:"status_url"`
	DirectURL   string    `json:"direct_url,omitempty"`
	PlaybackURL string    `json:"playback_url,omitempty"`
}
