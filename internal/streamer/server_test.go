package streamer

import (
	"encoding/json"
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

func TestNormalizeDirectPlaybackMode(t *testing.T) {
	tests := map[string]string{
		"":         directPlaybackModeProxy,
		"proxy":    directPlaybackModeProxy,
		"redirect": directPlaybackModeRedirect,
		"REDIRECT": directPlaybackModeRedirect,
		"unknown":  directPlaybackModeProxy,
	}
	for in, want := range tests {
		if got := normalizeDirectPlaybackMode(in); got != want {
			t.Fatalf("expected %q for %q, got %q", want, in, got)
		}
	}
}

func TestEnqueueJobReturnsFalseWhenQueueIsFull(t *testing.T) {
	s := &Server{jobQueue: make(chan string, 1)}
	if !s.enqueueJob("job-1") {
		t.Fatal("expected first enqueue to succeed")
	}
	if s.enqueueJob("job-2") {
		t.Fatal("expected enqueue to fail when queue is full")
	}
}

func TestProxyDirectMP4ForwardsRangeAndHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Range"); got != "bytes=0-3" {
			t.Fatalf("expected Range to be forwarded, got %q", got)
		}
		if got := r.Header.Get("Referer"); got != "https://www.bilibili.com/video/BV1Fj411p7cP" {
			t.Fatalf("expected Referer to be forwarded, got %q", got)
		}
		if got := r.Header.Get("Cookie"); got != "SESSDATA=test" {
			t.Fatalf("expected Cookie to be forwarded, got %q", got)
		}
		if got := r.Header.Get("User-Agent"); got != "test-agent" {
			t.Fatalf("expected User-Agent to be forwarded, got %q", got)
		}
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Range", "bytes 0-3/8")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("test"))
	}))
	defer upstream.Close()

	s := &Server{cfg: Config{
		YTDLPUserAgent: "test-agent",
		BilibiliCookie: "SESSDATA=test",
	}}
	req := httptest.NewRequest(http.MethodGet, "/?v=BV1Fj411p7cP", nil)
	req.Header.Set("Range", "bytes=0-3")
	rec := httptest.NewRecorder()

	s.proxyDirectMP4(rec, req, "https://www.bilibili.com/video/BV1Fj411p7cP", upstream.URL)

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("expected 206, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Range"); got != "bytes 0-3/8" {
		t.Fatalf("expected Content-Range to be copied, got %q", got)
	}
	if got := rec.Body.String(); got != "test" {
		t.Fatalf("expected body to be proxied, got %q", got)
	}
}

func TestValidateSourceURL(t *testing.T) {
	s := &Server{cfg: Config{AllowedHosts: []string{"bilibili.com", "b23.tv", "youtube.com", "youtu.be"}}}

	valid := []string{
		"https://www.bilibili.com/video/BV1xx411c7mD",
		"https://m.bilibili.com/video/BV1xx411c7mD",
		"https://b23.tv/abc123",
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ",
		"https://m.youtube.com/shorts/dQw4w9WgXcQ",
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
		{
			name: "youtube watch url",
			in:   "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			want: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		},
		{
			name: "youtube mobile share text",
			in:   "I found this on YouTube: https://youtu.be/dQw4w9WgXcQ?si=abc",
			want: "https://youtu.be/dQw4w9WgXcQ?si=abc",
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

func TestYouTubeVideoID(t *testing.T) {
	tests := map[string]string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ":      "dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ?si=abc":              "dQw4w9WgXcQ",
		"https://m.youtube.com/shorts/dQw4w9WgXcQ?feature": "dQw4w9WgXcQ",
		"https://www.youtube.com/embed/dQw4w9WgXcQ":        "dQw4w9WgXcQ",
	}
	for rawURL, want := range tests {
		if got := youtubeVideoID(rawURL); got != want {
			t.Fatalf("expected %q for %s, got %q", want, rawURL, got)
		}
	}
}

func TestBilibiliQualityCandidates(t *testing.T) {
	s := &Server{cfg: Config{
		BilibiliQuality:          80,
		BilibiliQualityFallbacks: []int{80, 64, 32},
	}}
	got := s.bilibiliQualityCandidates()
	want := []int{80, 64, 32, 16}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestSelectedBilibiliPageUsesURLPage(t *testing.T) {
	view := mustBilibiliView(t, `{
		"code": 0,
		"data": {
			"cid": 1001,
			"duration": 60,
			"pages": [
				{"cid": 1001, "page": 1, "duration": 60, "part": "first"},
				{"cid": 2002, "page": 2, "duration": 120, "part": "second"}
			]
		}
	}`)

	page, err := selectedBilibiliPage("https://www.bilibili.com/video/BV1Fj411p7cP?p=2", view)
	if err != nil {
		t.Fatal(err)
	}
	if page.CID != 2002 || page.Page != 2 || page.Duration != 120 {
		t.Fatalf("expected page 2 cid/duration, got %+v", page)
	}
}

func TestSelectedBilibiliPageDefaultsToFirstPage(t *testing.T) {
	view := mustBilibiliView(t, `{
		"code": 0,
		"data": {
			"cid": 1001,
			"duration": 60,
			"pages": [
				{"cid": 1001, "page": 1, "duration": 60, "part": "first"},
				{"cid": 2002, "page": 2, "duration": 120, "part": "second"}
			]
		}
	}`)

	page, err := selectedBilibiliPage("https://www.bilibili.com/video/BV1Fj411p7cP", view)
	if err != nil {
		t.Fatal(err)
	}
	if page.CID != 1001 || page.Page != 1 {
		t.Fatalf("expected first page, got %+v", page)
	}
}

func TestSelectedBilibiliPageRejectsMissingRequestedPage(t *testing.T) {
	view := mustBilibiliView(t, `{
		"code": 0,
		"data": {
			"cid": 1001,
			"duration": 60,
			"pages": [
				{"cid": 1001, "page": 1, "duration": 60, "part": "first"}
			]
		}
	}`)

	if _, err := selectedBilibiliPage("https://www.bilibili.com/video/BV1Fj411p7cP?p=2", view); err == nil {
		t.Fatal("expected missing page to fail")
	}
}

func TestBilibiliMediaIDIncludesPageWhenNeeded(t *testing.T) {
	tests := map[string]string{
		"https://www.bilibili.com/video/BV1Fj411p7cP":     "BV1Fj411p7cP",
		"https://www.bilibili.com/video/BV1Fj411p7cP?p=1": "BV1Fj411p7cP",
		"https://www.bilibili.com/video/BV1Fj411p7cP?p=2": "BV1Fj411p7cP-p2",
	}
	for rawURL, want := range tests {
		if got := bilibiliMediaID(rawURL); got != want {
			t.Fatalf("expected %q for %s, got %q", want, rawURL, got)
		}
	}
}

func TestMediaObjectPrefixIncludesBilibiliPage(t *testing.T) {
	s := &Server{cfg: Config{R2KeyPrefix: "vrchat"}}
	job := &Job{
		SourceURL: "https://www.bilibili.com/video/BV1Fj411p7cP?p=12",
		Format:    FormatMP4,
	}

	if got, want := s.mediaObjectPrefix(job), "vrchat/BV1Fj411p7cP-p12/mp4"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func mustBilibiliView(t *testing.T, raw string) bilibiliViewResponse {
	t.Helper()
	var view bilibiliViewResponse
	if err := json.Unmarshal([]byte(raw), &view); err != nil {
		t.Fatal(err)
	}
	return view
}
