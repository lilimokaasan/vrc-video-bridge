package streamer

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr                    string
	PublicBaseURL           string
	DataDir                 string
	YTDLPPath               string
	YTDLPCookiesFile        string
	YTDLPCookiesFromBrowser string
	YTDLPReferer            string
	YTDLPUserAgent          string
	YTDLPExtraArgs          []string
	FormatSelector          string
	FFmpegPath              string
	MaxConcurrentJobs       int
	JobTimeout              time.Duration
	AllowedHosts            []string
}

func LoadConfig() Config {
	return Config{
		Addr:                    envString("ADDR", ":8090"),
		PublicBaseURL:           strings.TrimRight(envString("PUBLIC_BASE_URL", "http://localhost:8090"), "/"),
		DataDir:                 envString("DATA_DIR", "data"),
		YTDLPPath:               envString("YTDLP_PATH", "yt-dlp"),
		YTDLPCookiesFile:        envString("YTDLP_COOKIES_FILE", ""),
		YTDLPCookiesFromBrowser: envString("YTDLP_COOKIES_FROM_BROWSER", ""),
		YTDLPReferer:            envString("YTDLP_REFERER", "https://www.bilibili.com/"),
		YTDLPUserAgent:          envString("YTDLP_USER_AGENT", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
		YTDLPExtraArgs:          envFields("YTDLP_EXTRA_ARGS"),
		FormatSelector:          envString("FORMAT_SELECTOR", "bv*[vcodec^=avc1]+ba/b[vcodec^=avc1]/bv*+ba/b"),
		FFmpegPath:              envString("FFMPEG_PATH", "ffmpeg"),
		MaxConcurrentJobs:       envInt("MAX_CONCURRENT_JOBS", 1),
		JobTimeout:              time.Duration(envInt("JOB_TIMEOUT_MINUTES", 90)) * time.Minute,
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

func envFields(key string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}
	return strings.Fields(value)
}
