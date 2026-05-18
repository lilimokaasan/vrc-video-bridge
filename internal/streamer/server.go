package streamer

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Server struct {
	cfg     Config
	mux     *http.ServeMux
	sem     chan struct{}
	storage objectStorage
	mu      sync.RWMutex
	jobs    map[string]*Job
}

func NewServer(cfg Config) (*Server, error) {
	if err := os.MkdirAll(cfg.mediaDir(), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.jobsDir(), 0755); err != nil {
		return nil, err
	}
	if _, err := exec.LookPath(cfg.YTDLPPath); err != nil {
		log.Printf("warning: yt-dlp not found at %q: %v", cfg.YTDLPPath, err)
	}
	if _, err := exec.LookPath(cfg.FFmpegPath); err != nil {
		log.Printf("warning: ffmpeg not found at %q: %v", cfg.FFmpegPath, err)
	}

	storage, err := newObjectStorage(cfg)
	if err != nil {
		return nil, err
	}
	if storage != nil {
		log.Printf("R2 upload enabled for bucket %q", cfg.R2Bucket)
	}

	s := &Server{
		cfg:     cfg,
		mux:     http.NewServeMux(),
		sem:     make(chan struct{}, cfg.MaxConcurrentJobs),
		storage: storage,
		jobs:    map[string]*Job{},
	}
	if err := s.loadJobs(); err != nil {
		return nil, err
	}
	s.registerRoutes()
	return s, nil
}

func (s *Server) Routes() http.Handler {
	return logRequests(s.mux)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("POST /api/jobs", s.handleCreateJob)
	s.mux.HandleFunc("GET /api/jobs/{id}", s.handleGetJob)
	s.mux.Handle("GET /media/", http.StripPrefix("/media/", http.FileServer(http.Dir(s.cfg.mediaDir()))))
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, indexHTML)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if req.Format == "" {
		req.Format = FormatHLS
	}
	if req.Format != FormatHLS && req.Format != FormatMP4 {
		writeError(w, http.StatusBadRequest, "format must be hls or mp4")
		return
	}
	if err := s.validateSourceURL(req.URL); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now().UTC()
	job := &Job{
		ID:        newJobID(),
		SourceURL: req.URL,
		Format:    req.Format,
		Status:    StatusQueued,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()
	if err := s.saveJob(job); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save job")
		return
	}

	go s.runJob(job.ID)

	writeJSON(w, http.StatusAccepted, CreateJobResponse{
		ID:        job.ID,
		Status:    job.Status,
		StatusURL: s.publicURL("/api/jobs/" + job.ID),
	})
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, ok := s.getJob(id)
	if !ok {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) runJob(id string) {
	s.sem <- struct{}{}
	defer func() { <-s.sem }()

	job, ok := s.getJob(id)
	if !ok {
		return
	}
	s.updateJob(job.ID, func(j *Job) {
		j.Status = StatusRunning
		j.Error = ""
	})

	if err := s.prepareMedia(job); err != nil {
		s.updateJob(job.ID, func(j *Job) {
			j.Status = StatusFailed
			j.Error = err.Error()
		})
		return
	}

	playbackURL, err := s.publishMedia(job)
	if err != nil {
		s.updateJob(job.ID, func(j *Job) {
			j.Status = StatusFailed
			j.Error = err.Error()
		})
		return
	}

	s.updateJob(job.ID, func(j *Job) {
		j.Status = StatusReady
		j.PlaybackURL = playbackURL
	})
}

func (s *Server) prepareMedia(job *Job) error {
	workDir, err := filepath.Abs(filepath.Join(s.cfg.mediaDir(), job.ID))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return err
	}

	sourcePath, err := s.downloadVideo(job.SourceURL, workDir)
	if err != nil {
		return err
	}

	if job.Format == FormatMP4 {
		target := filepath.Join(workDir, "video.mp4")
		if filepath.Clean(sourcePath) == filepath.Clean(target) {
			return nil
		}
		return runCommand(s.cfg.JobTimeout, workDir, s.cfg.FFmpegPath,
			"-y", "-i", sourcePath, "-c", "copy", "-movflags", "+faststart", target)
	}

	indexPath := filepath.Join(workDir, "index.m3u8")
	return runCommand(s.cfg.JobTimeout, workDir, s.cfg.FFmpegPath,
		"-y", "-i", sourcePath,
		"-c", "copy",
		"-start_number", "0",
		"-hls_time", "6",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", filepath.Join(workDir, "segment_%05d.ts"),
		indexPath)
}

func (s *Server) downloadVideo(sourceURL, workDir string) (string, error) {
	args := []string{
		"--no-playlist",
		"--restrict-filenames",
		"--merge-output-format", "mp4",
		"--add-header", "Referer:" + s.cfg.YTDLPReferer,
		"--user-agent", s.cfg.YTDLPUserAgent,
		"-f", s.cfg.FormatSelector,
		"-o", "source.%(ext)s",
	}
	if s.cfg.YTDLPCookiesFile != "" {
		args = append(args, "--cookies", s.cfg.YTDLPCookiesFile)
	} else if s.cfg.YTDLPCookiesFromBrowser != "" {
		args = append(args, "--cookies-from-browser", s.cfg.YTDLPCookiesFromBrowser)
	}
	args = append(args, s.cfg.YTDLPExtraArgs...)
	args = append(args, sourceURL)

	err := runCommand(s.cfg.JobTimeout, workDir, s.cfg.YTDLPPath, args...)
	if err != nil {
		if sourcePath, fallbackErr := s.downloadVideoWithBilibiliAPI(sourceURL, workDir); fallbackErr == nil {
			return sourcePath, nil
		} else if s.cfg.YTDLPCookiesFile == "" && s.cfg.YTDLPCookiesFromBrowser == "" {
			return "", fmt.Errorf("%w\nBilibili API fallback also failed: %v\nTry exporting browser cookies and setting YTDLP_COOKIES_FILE, or set YTDLP_COOKIES_FROM_BROWSER when yt-dlp can read your browser profile", err, fallbackErr)
		}
		return "", err
	}

	matches, err := filepath.Glob(filepath.Join(workDir, "source.*"))
	if err != nil {
		return "", err
	}
	for _, match := range matches {
		ext := strings.ToLower(filepath.Ext(match))
		if ext == ".mp4" || ext == ".mkv" || ext == ".webm" {
			return match, nil
		}
	}
	return "", errors.New("yt-dlp completed but no media file was produced")
}

var bvidPattern = regexp.MustCompile(`(?i)BV[0-9A-Za-z]+`)

type bilibiliViewResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		CID int64 `json:"cid"`
	} `json:"data"`
}

type bilibiliPlayURLResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Dash struct {
			Video []bilibiliDashStream `json:"video"`
			Audio []bilibiliDashStream `json:"audio"`
		} `json:"dash"`
	} `json:"data"`
}

type bilibiliDashStream struct {
	BaseURL   string `json:"baseUrl"`
	Codecs    string `json:"codecs"`
	Bandwidth int    `json:"bandwidth"`
}

func (s *Server) downloadVideoWithBilibiliAPI(sourceURL, workDir string) (string, error) {
	bvid := bvidPattern.FindString(sourceURL)
	if bvid == "" {
		return "", errors.New("no BV id found in URL")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	var view bilibiliViewResponse
	if err := s.fetchBilibiliJSON(client, "https://api.bilibili.com/x/web-interface/view?bvid="+url.QueryEscape(bvid), sourceURL, &view); err != nil {
		return "", err
	}
	if view.Code != 0 || view.Data.CID == 0 {
		return "", fmt.Errorf("view API returned code %d: %s", view.Code, view.Message)
	}

	playURL := fmt.Sprintf("https://api.bilibili.com/x/player/playurl?bvid=%s&cid=%d&qn=64&fnval=16&fourk=1", url.QueryEscape(bvid), view.Data.CID)
	var play bilibiliPlayURLResponse
	if err := s.fetchBilibiliJSON(client, playURL, sourceURL, &play); err != nil {
		return "", err
	}
	if play.Code != 0 {
		return "", fmt.Errorf("playurl API returned code %d: %s", play.Code, play.Message)
	}

	video := selectDashStream(play.Data.Dash.Video, "avc1")
	audio := selectDashStream(play.Data.Dash.Audio, "mp4a")
	if video.BaseURL == "" || audio.BaseURL == "" {
		return "", errors.New("playurl API did not return usable H.264/AAC streams")
	}

	target := filepath.Join(workDir, "source.mp4")
	headers := fmt.Sprintf("Referer: %s\r\nUser-Agent: %s\r\n", sourceURL, s.cfg.YTDLPUserAgent)
	if err := runCommand(s.cfg.JobTimeout, workDir, s.cfg.FFmpegPath,
		"-y",
		"-headers", headers,
		"-i", video.BaseURL,
		"-headers", headers,
		"-i", audio.BaseURL,
		"-c", "copy",
		"-movflags", "+faststart",
		target); err != nil {
		return "", err
	}
	return target, nil
}

func (s *Server) fetchBilibiliJSON(client *http.Client, rawURL, referer string, target any) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", s.cfg.YTDLPUserAgent)
	req.Header.Set("Referer", referer)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("Bilibili API returned HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func selectDashStream(streams []bilibiliDashStream, codecPrefix string) bilibiliDashStream {
	var selected bilibiliDashStream
	for _, stream := range streams {
		if !strings.HasPrefix(stream.Codecs, codecPrefix) {
			continue
		}
		if stream.Bandwidth > selected.Bandwidth {
			selected = stream
		}
	}
	return selected
}

func (s *Server) validateSourceURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("url must be an absolute http or https URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("url must use http or https")
	}
	host := strings.ToLower(parsed.Hostname())
	for _, allowed := range s.cfg.AllowedHosts {
		allowed = strings.TrimSpace(strings.ToLower(allowed))
		if allowed == "" {
			continue
		}
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return nil
		}
	}
	return fmt.Errorf("host %q is not allowed", host)
}

func (s *Server) getJob(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	copy := *job
	return &copy, true
}

func (s *Server) updateJob(id string, mutate func(*Job)) {
	s.mu.Lock()
	job := s.jobs[id]
	if job != nil {
		mutate(job)
		job.UpdatedAt = time.Now().UTC()
	}
	s.mu.Unlock()
	if job != nil {
		if err := s.saveJob(job); err != nil {
			log.Printf("failed to save job %s: %v", id, err)
		}
	}
}

func (s *Server) loadJobs() error {
	entries, err := os.ReadDir(s.cfg.jobsDir())
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.cfg.jobsDir(), entry.Name()))
		if err != nil {
			return err
		}
		var job Job
		if err := json.Unmarshal(data, &job); err != nil {
			return err
		}
		if job.Status == StatusRunning || job.Status == StatusQueued {
			job.Status = StatusFailed
			job.Error = "server restarted before this job completed"
			job.UpdatedAt = time.Now().UTC()
		}
		s.jobs[job.ID] = &job
	}
	return nil
}

func (s *Server) saveJob(job *Job) error {
	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.cfg.jobsDir(), job.ID+".json"), data, 0644)
}

func (s *Server) publicURL(path string) string {
	return s.cfg.PublicBaseURL + path
}

func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func init() {
	_ = mime.AddExtensionType(".m3u8", "application/vnd.apple.mpegurl")
	_ = mime.AddExtensionType(".ts", "video/mp2t")
}
