package streamer

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestCreateJobRequiresR2Storage(t *testing.T) {
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

	body := `{"url":"https://www.bilibili.com/video/BV1xx411c7mD","format":"mp4"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected create job without R2 to return 503, got %d", rec.Code)
	}
}

func TestR2EnabledRequiresCompleteConfig(t *testing.T) {
	incomplete := Config{
		R2Endpoint:        "https://example.r2.cloudflarestorage.com",
		R2AccessKeyID:     "key",
		R2SecretAccessKey: "secret",
		R2Bucket:          "bucket",
	}
	if incomplete.r2Enabled() {
		t.Fatal("expected incomplete R2 config to be disabled")
	}

	complete := incomplete
	complete.R2PublicBaseURL = "https://video.example.com"
	if !complete.r2Enabled() {
		t.Fatal("expected complete R2 config to be enabled")
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

func TestNormalizeBilibiliValue(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "bvid",
			in:   "BV1Fj411p7cP",
			want: "https://www.bilibili.com/video/BV1Fj411p7cP",
		},
		{
			name: "video url",
			in:   "https://www.bilibili.com/video/BV1Fj411p7cP/?spm_id_from=333.788",
			want: "https://www.bilibili.com/video/BV1Fj411p7cP/?spm_id_from=333.788",
		},
		{
			name: "mobile share short link",
			in:   "【兔兔兔-哔哩哔哩】 https://b23.tv/GIxeXkP",
			want: "https://b23.tv/GIxeXkP",
		},
		{
			name: "mobile share short link with chinese suffix",
			in:   "【兔兔兔-哔哩哔哩】 https://b23.tv/GIxeXkP，复制打开",
			want: "https://b23.tv/GIxeXkP",
		},
		{
			name: "mobile share video url",
			in:   "【兔兔兔-哔哩哔哩】 https://www.bilibili.com/video/BV1Fj411p7cP/ 复制打开",
			want: "https://www.bilibili.com/video/BV1Fj411p7cP/",
		},
		{
			name: "mobile share bvid text",
			in:   "分享一个视频 BV1Fj411p7cP 给你",
			want: "https://www.bilibili.com/video/BV1Fj411p7cP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeBilibiliValue(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}

	if _, err := normalizeBilibiliValue("not-a-video"); err == nil {
		t.Fatal("expected invalid value to fail")
	}
}
