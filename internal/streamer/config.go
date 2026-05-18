package streamer

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr              string
	PublicBaseURL     string
	DataDir           string
	YTDLPPath         string
	YTDLPCookiesFile  string
	FFmpegPath        string
	MaxConcurrentJobs int
	JobTimeout        time.Duration
	AllowedHosts      []string
}

func LoadConfig() Config {
	return Config{
		Addr:              envString("ADDR", ":8090"),
		PublicBaseURL:     strings.TrimRight(envString("PUBLIC_BASE_URL", "http://localhost:8090"), "/"),
		DataDir:           envString("DATA_DIR", "data"),
		YTDLPPath:         envString("YTDLP_PATH", "yt-dlp"),
		YTDLPCookiesFile:  envString("YTDLP_COOKIES_FILE", ""),
		FFmpegPath:        envString("FFMPEG_PATH", "ffmpeg"),
		MaxConcurrentJobs: envInt("MAX_CONCURRENT_JOBS", 1),
		JobTimeout:        time.Duration(envInt("JOB_TIMEOUT_MINUTES", 90)) * time.Minute,
		AllowedHosts: strings.Split(envString("ALLOWED_HOSTS",
			"bilibili.com,www.bilibili.com,m.bilibili.com,b23.tv"), ","),
	}
}

func (c Config) mediaDir() string {
	return filepath.Join(c.DataDir, "media")
}

func (c Config) jobsDir() string {
	return filepath.Join(c.DataDir, "jobs")
}

func envString(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
