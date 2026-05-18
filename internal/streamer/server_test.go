package streamer

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoutesServeHealth(t *testing.T) {
	server, err := NewServer(Config{
		PublicBaseURL:     "http://example.test",
		DataDir:           t.TempDir(),
		YTDLPPath:         "yt-dlp",
		FFmpegPath:        "ffmpeg",
		MaxConcurrentJobs: 1,
		JobTimeout:        1,
		AllowedHosts:      []string{"bilibili.com"},
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected health route to return 200, got %d", rec.Code)
	}
}

func TestValidateSourceURL(t *testing.T) {
	s := &Server{cfg: Config{AllowedHosts: []string{"bilibili.com", "b23.tv"}}}

	valid := []string{
		"https://www.bilibili.com/video/BV1xx411c7mD",
		"https://m.bilibili.com/video/BV1xx411c7mD",
		"https://b23.tv/abc123",
	}
	for _, raw := range valid {
		if err := s.validateSourceURL(raw); err != nil {
			t.Fatalf("expected %s to be valid: %v", raw, err)
		}
	}

	invalid := []string{
		"not-a-url",
		"file:///tmp/video.mp4",
		"https://example.com/video",
	}
	for _, raw := range invalid {
		if err := s.validateSourceURL(raw); err == nil {
			t.Fatalf("expected %s to be invalid", raw)
		}
	}
}
