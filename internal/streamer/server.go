package streamer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	s.mux.HandleFunc("GET /favicon.png", s.handleFaviconPNG)
	s.mux.HandleFunc("GET /favicon.ico", s.handleFaviconICO)
	s.mux.Handle("GET /media/", http.StripPrefix("/media/", http.FileServer(http.Dir(s.cfg.mediaDir()))))
	s.mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(s.cfg.AssetsDir))))
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if value := strings.TrimSpace(r.URL.Query().Get("v")); value != "" {
		s.handleDirectRedirect(w, r, value)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, indexHTML)
}

func (s *Server) handleFaviconPNG(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(s.cfg.AssetsDir, "favicon.png"))
}

func (s *Server) handleFaviconICO(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(s.cfg.AssetsDir, "favicon.ico"))
}

func (s *Server) handleDirectRedirect(w http.ResponseWriter, r *http.Request, value string) {
	sourceURL, err := normalizeBilibiliValue(value)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.validateSourceURL(sourceURL); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	directURL, err := s.resolveBilibiliDirectURL(sourceURL)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	http.Redirect(w, r, directURL, http.StatusFound)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil {
		writeError(w, http.StatusServiceUnavailable, "R2 storage is not configured")
		return
	}

	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if req.Format == "" {
		req.Format = FormatMP4
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
		j.Message = "正在寻找这段视频..."
		j.Error = ""
		j.Progress = JobProgress{}
	})

	directURL, err := s.prepareMedia(job, func(directURL string) {
		s.updateJob(job.ID, func(j *Job) {
			j.DirectURL = directURL
			if directURL != "" {
				j.Message = "先到的小纸条已经找到，正在整理视频..."
			}
		})
	})
	if err != nil {
		s.cleanupJobMedia(job.ID)
		s.updateJob(job.ID, func(j *Job) {
			j.Status = StatusFailed
			j.Message = ""
			j.Error = err.Error()
		})
		return
	}
	defer s.cleanupJobMedia(job.ID)

	s.updateJob(job.ID, func(j *Job) {
		j.DirectURL = directURL
		if directURL != "" {
			j.Message = "视频已经整理好，正在准备分享链接..."
		} else {
			j.Message = "视频已经整理好，正在准备分享链接..."
		}
	})
	s.setJobProgress(job.ID, "upload", "上传到 R2", "active", 0, 0, "正在准备分享链接...")

	playbackURL, err := s.publishMedia(job)
	if err != nil {
		s.updateJob(job.ID, func(j *Job) {
			j.Status = StatusFailed
			j.Message = ""
			j.Error = err.Error()
		})
		return
	}

	s.updateJob(job.ID, func(j *Job) {
		j.Status = StatusReady
		j.Message = ""
		j.PlaybackURL = playbackURL
		setProgressStep(j, "download", ProgressStep{Label: "下载 MP4", State: "done", Percent: 100, Message: "MP4 已下载完成"})
		setProgressStep(j, "upload", ProgressStep{Label: "上传到 R2", State: "done", Percent: 100, Message: "分享链接已准备好"})
	})
}

func (s *Server) cleanupJobMedia(id string) {
	if id == "" {
		return
	}
	workDir := filepath.Join(s.cfg.mediaDir(), id)
	if err := os.RemoveAll(workDir); err != nil {
		log.Printf("failed to cleanup media for job %s: %v", id, err)
		return
	}
	log.Printf("cleaned up media for job %s", id)
}

type directURLCallback func(string)

func (s *Server) prepareMedia(job *Job, onDirectURL directURLCallback) (string, error) {
	workDir, err := filepath.Abs(filepath.Join(s.cfg.mediaDir(), job.ID))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return "", err
	}

	sourcePath, directURL, err := s.downloadVideo(job, workDir, onDirectURL)
	if err != nil {
		return "", err
	}

	if job.Format == FormatMP4 {
		target := filepath.Join(workDir, "video.mp4")
		if filepath.Clean(sourcePath) == filepath.Clean(target) {
			return directURL, nil
		}
		s.setJobProgress(job.ID, "download", "下载 MP4", "active", 0, 0, "正在整理视频文件...")
		if err := runCommand(s.cfg.JobTimeout, workDir, s.cfg.FFmpegPath,
			"-y", "-i", sourcePath, "-c", "copy", "-movflags", "+faststart", target); err != nil {
			return "", err
		}
		s.setJobProgress(job.ID, "download", "下载 MP4", "done", 100, 100, "MP4 已下载完成")
		return directURL, nil
	}

	indexPath := filepath.Join(workDir, "index.m3u8")
	s.setJobProgress(job.ID, "download", "下载 MP4", "active", 0, 0, "正在整理 HLS 片段...")
	if err := runCommand(s.cfg.JobTimeout, workDir, s.cfg.FFmpegPath,
		"-y", "-i", sourcePath,
		"-c", "copy",
		"-start_number", "0",
		"-hls_time", "6",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", filepath.Join(workDir, "segment_%05d.ts"),
		indexPath); err != nil {
		return "", err
	}
	s.setJobProgress(job.ID, "download", "下载 MP4", "done", 100, 100, "HLS 片段已整理完成")
	return directURL, nil
}

func (s *Server) downloadVideo(job *Job, workDir string, onDirectURL directURLCallback) (string, string, error) {
	sourceURL := job.SourceURL
	s.setJobProgress(job.ID, "download", "下载 MP4", "active", 0, 0, "正在寻找视频入口...")
	if bvidPattern.MatchString(sourceURL) {
		if sourcePath, directURL, err := s.downloadVideoWithBilibiliAPI(job, workDir, onDirectURL); err == nil {
			s.setJobProgress(job.ID, "download", "下载 MP4", "done", 100, 100, "MP4 已下载完成")
			return sourcePath, directURL, nil
		}
	}

	args := []string{
		"--no-playlist",
		"--restrict-filenames",
		"--merge-output-format", "mp4",
		"--add-header", "Referer:" + s.cfg.YTDLPReferer,
		"--user-agent", s.cfg.YTDLPUserAgent,
		"-f", s.cfg.FormatSelector,
		"-o", "source.%(ext)s",
	}
	if s.cfg.BilibiliCookie != "" {
		args = append(args, "--add-header", "Cookie:"+s.cfg.BilibiliCookie)
	}
	if s.cfg.YTDLPCookiesFile != "" {
		args = append(args, "--cookies", s.cfg.YTDLPCookiesFile)
	} else if s.cfg.YTDLPCookiesFromBrowser != "" {
		args = append(args, "--cookies-from-browser", s.cfg.YTDLPCookiesFromBrowser)
	}
	args = append(args, s.cfg.YTDLPExtraArgs...)
	args = append(args, sourceURL)

	s.setJobProgress(job.ID, "download", "下载 MP4", "active", 0, 0, "正在下载 MP4...")
	err := runCommand(s.cfg.JobTimeout, workDir, s.cfg.YTDLPPath, args...)
	if err != nil {
		if sourcePath, directURL, fallbackErr := s.downloadVideoWithBilibiliAPI(job, workDir, onDirectURL); fallbackErr == nil {
			s.setJobProgress(job.ID, "download", "下载 MP4", "done", 100, 100, "MP4 已下载完成")
			return sourcePath, directURL, nil
		} else if s.cfg.YTDLPCookiesFile == "" && s.cfg.YTDLPCookiesFromBrowser == "" {
			return "", "", fmt.Errorf("%w\nBilibili API fallback also failed: %v\nTry exporting browser cookies and setting YTDLP_COOKIES_FILE, or set YTDLP_COOKIES_FROM_BROWSER when yt-dlp can read your browser profile", err, fallbackErr)
		}
		return "", "", err
	}

	matches, err := filepath.Glob(filepath.Join(workDir, "source.*"))
	if err != nil {
		return "", "", err
	}
	for _, match := range matches {
		ext := strings.ToLower(filepath.Ext(match))
		if ext == ".mp4" || ext == ".mkv" || ext == ".webm" {
			s.setJobProgress(job.ID, "download", "下载 MP4", "done", 100, 100, "MP4 已下载完成")
			return match, "", nil
		}
	}
	return "", "", errors.New("yt-dlp completed but no media file was produced")
}

var bvidPattern = regexp.MustCompile(`(?i)BV[0-9A-Za-z]+`)

func normalizeBilibiliValue(value string) (string, error) {
	if bvid := bvidPattern.FindString(value); bvid != "" {
		if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			return value, nil
		}
		return "https://www.bilibili.com/video/" + bvid, nil
	}
	return "", errors.New("v must be a Bilibili BV id or video URL")
}

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
		DURL []bilibiliDirectStream `json:"durl"`
		Dash struct {
			Video []bilibiliDashStream `json:"video"`
			Audio []bilibiliDashStream `json:"audio"`
		} `json:"dash"`
	} `json:"data"`
}

type bilibiliDirectStream struct {
	URL       string   `json:"url"`
	Size      int64    `json:"size"`
	Length    int      `json:"length"`
	BackupURL []string `json:"backup_url"`
}

type bilibiliDashStream struct {
	BaseURL   string `json:"baseUrl"`
	Codecs    string `json:"codecs"`
	Bandwidth int    `json:"bandwidth"`
}

func (s *Server) resolveBilibiliDirectURL(sourceURL string) (string, error) {
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

	playURL := fmt.Sprintf("https://api.bilibili.com/x/player/playurl?bvid=%s&cid=%d&qn=64&platform=html5&high_quality=1", url.QueryEscape(bvid), view.Data.CID)
	var play bilibiliPlayURLResponse
	if err := s.fetchBilibiliJSON(client, playURL, sourceURL, &play); err != nil {
		return "", err
	}
	if play.Code != 0 {
		return "", fmt.Errorf("playurl API returned code %d: %s", play.Code, play.Message)
	}
	direct := selectDirectStream(play.Data.DURL)
	if direct.URL == "" {
		return "", errors.New("playurl API did not return a progressive MP4 URL")
	}
	return direct.URL, nil
}

func (s *Server) downloadVideoWithBilibiliAPI(job *Job, workDir string, onDirectURL directURLCallback) (string, string, error) {
	sourceURL := job.SourceURL
	bvid := bvidPattern.FindString(sourceURL)
	if bvid == "" {
		return "", "", errors.New("no BV id found in URL")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	var view bilibiliViewResponse
	if err := s.fetchBilibiliJSON(client, "https://api.bilibili.com/x/web-interface/view?bvid="+url.QueryEscape(bvid), sourceURL, &view); err != nil {
		return "", "", err
	}
	if view.Code != 0 || view.Data.CID == 0 {
		return "", "", fmt.Errorf("view API returned code %d: %s", view.Code, view.Message)
	}

	var fallbackDirectURL string
	progressiveURL := fmt.Sprintf("https://api.bilibili.com/x/player/playurl?bvid=%s&cid=%d&qn=64&platform=html5&high_quality=1", url.QueryEscape(bvid), view.Data.CID)
	var progressive bilibiliPlayURLResponse
	if err := s.fetchBilibiliJSON(client, progressiveURL, sourceURL, &progressive); err == nil {
		if progressive.Code != 0 {
			log.Printf("Bilibili progressive playurl API returned code %d: %s", progressive.Code, progressive.Message)
		} else if direct := selectDirectStream(progressive.Data.DURL); direct.URL != "" {
			fallbackDirectURL = direct.URL
			if onDirectURL != nil {
				onDirectURL(direct.URL)
			}
			target := filepath.Join(workDir, "source.mp4")
			if err := s.downloadDirectMP4WithRetry(job.ID, direct.URL, sourceURL, target, direct.Size); err == nil {
				return target, direct.URL, nil
			} else {
				_ = os.Remove(target)
				log.Printf("Bilibili progressive MP4 download failed, falling back to DASH: %v", err)
			}
		}
	} else {
		log.Printf("Bilibili progressive playurl API failed, falling back to DASH: %v", err)
	}

	playURL := fmt.Sprintf("https://api.bilibili.com/x/player/playurl?bvid=%s&cid=%d&qn=64&fnval=16&fourk=1", url.QueryEscape(bvid), view.Data.CID)
	var play bilibiliPlayURLResponse
	if err := s.fetchBilibiliJSON(client, playURL, sourceURL, &play); err != nil {
		return "", "", err
	}
	if play.Code != 0 {
		return "", "", fmt.Errorf("playurl API returned code %d: %s", play.Code, play.Message)
	}

	video := selectDashStream(play.Data.Dash.Video, "avc1")
	audio := selectDashStream(play.Data.Dash.Audio, "mp4a")
	if video.BaseURL == "" || audio.BaseURL == "" {
		return "", "", errors.New("playurl API did not return usable H.264/AAC streams")
	}

	target := filepath.Join(workDir, "source.mp4")
	headers := s.bilibiliHeaders(sourceURL)
	s.setJobProgress(job.ID, "download", "下载 MP4", "active", 0, 0, "正在下载并整理 MP4...")
	if err := runCommand(s.cfg.JobTimeout, workDir, s.cfg.FFmpegPath,
		"-y",
		"-headers", headers,
		"-i", video.BaseURL,
		"-headers", headers,
		"-i", audio.BaseURL,
		"-c", "copy",
		"-movflags", "+faststart",
		target); err != nil {
		return "", "", err
	}
	return target, fallbackDirectURL, nil
}

func (s *Server) downloadDirectMP4WithRetry(jobID, rawURL, referer, target string, expectedSize int64) error {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			s.setJobProgress(jobID, "download", "下载 MP4", "active", 0, expectedSize, fmt.Sprintf("连接中断，正在重试第 %d 次...", attempt))
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		_ = os.Remove(target)
		if err := s.downloadDirectMP4(jobID, rawURL, referer, target, expectedSize); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return lastErr
}

func (s *Server) downloadDirectMP4(jobID, rawURL, referer, target string, expectedSize int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.JobTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", s.cfg.YTDLPUserAgent)
	req.Header.Set("Referer", referer)
	if s.cfg.BilibiliCookie != "" {
		req.Header.Set("Cookie", s.cfg.BilibiliCookie)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("download MP4 returned HTTP %d", resp.StatusCode)
	}

	total := expectedSize
	if total <= 0 {
		total = resp.ContentLength
	}

	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()

	report := s.throttledProgress(jobID, "download", "下载 MP4")
	report(0, total)
	buf := make([]byte, 1024*128)
	var done int64
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := file.Write(buf[:n]); err != nil {
				return err
			}
			done += int64(n)
			report(done, total)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	report(done, done)
	return file.Close()
}

func (s *Server) fetchBilibiliJSON(client *http.Client, rawURL, referer string, target any) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", s.cfg.YTDLPUserAgent)
	req.Header.Set("Referer", referer)
	if s.cfg.BilibiliCookie != "" {
		req.Header.Set("Cookie", s.cfg.BilibiliCookie)
	}

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

func (s *Server) bilibiliHeaders(referer string) string {
	headers := fmt.Sprintf("Referer: %s\r\nUser-Agent: %s\r\n", referer, s.cfg.YTDLPUserAgent)
	if s.cfg.BilibiliCookie != "" {
		headers += "Cookie: " + s.cfg.BilibiliCookie + "\r\n"
	}
	return headers
}

func selectDirectStream(streams []bilibiliDirectStream) bilibiliDirectStream {
	var selected bilibiliDirectStream
	for _, stream := range streams {
		if stream.URL == "" {
			continue
		}
		if stream.Size > selected.Size {
			selected = stream
		}
	}
	return selected
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
	return cloneJob(job), true
}

func (s *Server) updateJob(id string, mutate func(*Job)) {
	var snapshot *Job
	s.mu.Lock()
	job := s.jobs[id]
	if job != nil {
		mutate(job)
		job.UpdatedAt = time.Now().UTC()
		snapshot = cloneJob(job)
	}
	s.mu.Unlock()
	if snapshot != nil {
		if err := s.saveJob(snapshot); err != nil {
			log.Printf("failed to save job %s: %v", id, err)
		}
	}
}

func cloneJob(job *Job) *Job {
	if job == nil {
		return nil
	}
	copy := *job
	if job.Progress != nil {
		copy.Progress = make(JobProgress, len(job.Progress))
		for key, step := range job.Progress {
			copy.Progress[key] = step
		}
	}
	return &copy
}

func setProgressStep(job *Job, key string, step ProgressStep) {
	if job.Progress == nil {
		job.Progress = JobProgress{}
	}
	if step.BytesTotal > 0 {
		percent := int(step.BytesDone * 100 / step.BytesTotal)
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}
		step.Percent = percent
	}
	job.Progress[key] = step
}

func (s *Server) setJobProgress(id, key, label, state string, done, total int64, message string) {
	s.updateJob(id, func(j *Job) {
		setProgressStep(j, key, ProgressStep{
			Label:      label,
			State:      state,
			BytesDone:  done,
			BytesTotal: total,
			Message:    message,
		})
	})
}

func (s *Server) throttledProgress(id, key, label string) progressCallback {
	var last time.Time
	return func(done, total int64) {
		now := time.Now()
		if done != total && now.Sub(last) < 500*time.Millisecond {
			return
		}
		last = now
		message := "正在进行中..."
		if total > 0 {
			message = fmt.Sprintf("%s / %s", formatBytes(done), formatBytes(total))
		}
		state := "active"
		if total > 0 && done >= total {
			state = "done"
		}
		s.setJobProgress(id, key, label, state, done, total, message)
	}
}

func formatBytes(value int64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	div, exp := int64(unit), 0
	for n := value / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(value)/float64(div), "KMGTPE"[exp])
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
