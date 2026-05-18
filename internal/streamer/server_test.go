package streamer

import "testing"

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
