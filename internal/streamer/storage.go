package streamer

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type objectStorage interface {
	UploadFile(sourcePath, key string) error
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

func (s *r2Storage) UploadFile(sourcePath, key string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(sourcePath)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = s.client.PutObject(&s3.PutObjectInput{
		Bucket:       aws.String(s.cfg.R2Bucket),
		Key:          aws.String(key),
		Body:         file,
		ContentType:  aws.String(contentType),
		CacheControl: aws.String(s.cfg.R2CacheControl),
	})
	if err != nil {
		return fmt.Errorf("upload %s to R2 key %s: %w", sourcePath, key, err)
	}
	return nil
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
		if err := s.storage.UploadFile(filepath.Join(workDir, "video.mp4"), key); err != nil {
			return "", err
		}
		return s.storage.PublicURL(key), nil
	}

	entries, err := os.ReadDir(workDir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".m3u8" && ext != ".ts" {
			continue
		}
		key := prefix + "/" + entry.Name()
		if err := s.storage.UploadFile(filepath.Join(workDir, entry.Name()), key); err != nil {
			return "", err
		}
	}
	return s.storage.PublicURL(prefix + "/index.m3u8"), nil
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
	if err := s.storage.UploadFile(file.Name(), key); err != nil {
		return "", err
	}
	return s.storage.PublicURL(key), nil
}

func (s *Server) mediaObjectPrefix(job *Job) string {
	parts := []string{}
	if s.cfg.R2KeyPrefix != "" {
		parts = append(parts, s.cfg.R2KeyPrefix)
	}
	if bvid := bvidPattern.FindString(job.SourceURL); bvid != "" {
		parts = append(parts, bvid)
	} else {
		parts = append(parts, job.ID)
	}
	parts = append(parts, job.Format.String())
	return strings.Join(parts, "/")
}
