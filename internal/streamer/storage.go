package streamer

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type progressCallback func(done, total int64)

type objectStorage interface {
	UploadFile(sourcePath, key string, onProgress progressCallback) error
	PublicURL(key string) string
}

type r2Storage struct {
	cfg    Config
	client *s3.S3
}

func newObjectStorage(cfg Config) (objectStorage, error) {
	if !cfg.r2Enabled() {
		return nil, nil
	}

	sess, err := session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(cfg.R2AccessKeyID, cfg.R2SecretAccessKey, ""),
		Endpoint:         aws.String(cfg.R2Endpoint),
		HTTPClient:       &http.Client{Timeout: cfg.R2UploadTimeout},
		MaxRetries:       aws.Int(2),
		Region:           aws.String("auto"),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	return &r2Storage{
		cfg:    cfg,
		client: s3.New(sess),
	}, nil
}

func (s *r2Storage) UploadFile(sourcePath, key string, onProgress progressCallback) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	total := info.Size()
	body := io.ReadSeeker(file)
	if onProgress != nil {
		onProgress(0, total)
		body = &progressReadSeeker{file: file, total: total, onProgress: onProgress}
	}

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(sourcePath)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.R2UploadTimeout)
	defer cancel()

	log.Printf("uploading %s to R2 key %s", sourcePath, key)
	_, err = s.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(s.cfg.R2Bucket),
		Key:          aws.String(key),
		Body:         body,
		ContentType:  aws.String(contentType),
		CacheControl: aws.String(s.cfg.R2CacheControl),
	})
	if err != nil {
		return fmt.Errorf("upload %s to R2 key %s: %w", sourcePath, key, err)
	}
	log.Printf("uploaded %s to R2 key %s", sourcePath, key)
	if onProgress != nil {
		onProgress(total, total)
	}
	return nil
}

type progressReadSeeker struct {
	file       *os.File
	total      int64
	read       int64
	onProgress progressCallback
}

func (r *progressReadSeeker) Read(p []byte) (int, error) {
	n, err := r.file.Read(p)
	if n > 0 {
		r.read += int64(n)
		r.onProgress(r.read, r.total)
	}
	return n, err
}

func (r *progressReadSeeker) Seek(offset int64, whence int) (int64, error) {
	pos, err := r.file.Seek(offset, whence)
	if err == nil {
		r.read = pos
		r.onProgress(r.read, r.total)
	}
	return pos, err
}

func (s *r2Storage) PublicURL(key string) string {
	return s.cfg.R2PublicBaseURL + "/" + strings.TrimLeft(key, "/")
}

func (s *Server) publishMedia(job *Job) (string, error) {
	workDir := filepath.Join(s.cfg.mediaDir(), job.ID)
	if s.storage == nil {
		if job.Format == FormatHLS {
			return s.publicURL("/media/" + job.ID + "/index.m3u8"), nil
		}
		return s.publicURL("/media/" + job.ID + "/video.mp4"), nil
	}

	prefix := s.mediaObjectPrefix(job)
	if job.Format == FormatMP4 {
		key := prefix + "/video.mp4"
		if err := s.storage.UploadFile(filepath.Join(workDir, "video.mp4"), key, s.throttledProgress(job.ID, "upload", "上传到 R2")); err != nil {
			return "", err
		}
		return cacheBustURL(s.storage.PublicURL(key), job.ID), nil
	}

	entries, err := os.ReadDir(workDir)
	if err != nil {
		return "", err
	}
	var totalBytes int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".m3u8" && ext != ".ts" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return "", err
		}
		totalBytes += info.Size()
	}

	var uploadedBytes int64
	reportUpload := s.throttledProgress(job.ID, "upload", "上传到 R2")
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".m3u8" && ext != ".ts" {
			continue
		}
		key := prefix + "/" + entry.Name()
		base := uploadedBytes
		sourcePath := filepath.Join(workDir, entry.Name())
		if err := s.storage.UploadFile(sourcePath, key, func(done, total int64) {
			reportUpload(base+done, totalBytes)
			if done == total {
				uploadedBytes = base + total
			}
		}); err != nil {
			return "", err
		}
	}
	return cacheBustURL(s.storage.PublicURL(prefix+"/index.m3u8"), job.ID), nil
}

func cacheBustURL(rawURL, token string) string {
	if rawURL == "" || token == "" {
		return rawURL
	}
	separator := "?"
	if strings.Contains(rawURL, "?") {
		separator = "&"
	}
	return rawURL + separator + "v=" + url.QueryEscape(token)
}

func (s *Server) CheckStorage() (string, error) {
	if s.storage == nil {
		return "", fmt.Errorf("R2 is not configured")
	}

	file, err := os.CreateTemp("", "bili-vrc-r2-check-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())
	defer file.Close()

	if _, err := file.WriteString("bili-vrc-streamer R2 check\n"); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}

	key := strings.Trim(strings.Join([]string{s.cfg.R2KeyPrefix, "_healthcheck.txt"}, "/"), "/")
	if err := s.storage.UploadFile(file.Name(), key, nil); err != nil {
		return "", err
	}
	return s.storage.PublicURL(key), nil
}

func (s *Server) mediaObjectPrefix(job *Job) string {
	parts := []string{}
	if s.cfg.R2KeyPrefix != "" {
		parts = append(parts, s.cfg.R2KeyPrefix)
	}
	if bvid := bilibiliMediaID(job.SourceURL); bvid != "" {
		parts = append(parts, bvid)
	} else if youtubeID := youtubeVideoID(job.SourceURL); youtubeID != "" {
		parts = append(parts, "youtube-"+youtubeID)
	} else {
		parts = append(parts, job.ID)
	}
	parts = append(parts, job.Format.String())
	return strings.Join(parts, "/")
}
