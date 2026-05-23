package streamer

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr                     string
	PublicBaseURL            string
	DataDir                  string
	AssetsDir                string
	YTDLPPath                string
	YTDLPCookiesFile         string
	YTDLPCookiesFromBrowser  string
	YTDLPReferer             string
	YTDLPUserAgent           string
	YTDLPExtraArgs           []string
	BilibiliCookie           string
	BilibiliQuality          int
	BilibiliQualityFallbacks []int
	FormatSelector           string
	FFmpegPath               string
	MaxConcurrentJobs        int
	JobTimeout               time.Duration
	AllowedHosts             []string
	R2Endpoint               string
	R2AccessKeyID            string
	R2SecretAccessKey        string
	R2Bucket                 string
	R2PublicBaseURL          string
	R2KeyPrefix              string
	R2CacheControl           string
	R2UploadTimeout          time.Duration
}

func LoadConfig() Config {
	loadDotEnv(".env")

	return Config{
		Addr:                     envString("ADDR", ":8090"),
		PublicBaseURL:            strings.TrimRight(envString("PUBLIC_BASE_URL", "http://localhost:8090"), "/"),
		DataDir:                  envString("DATA_DIR", "data"),
		AssetsDir:                envString("ASSETS_DIR", "web/assets"),
		YTDLPPath:                envString("YTDLP_PATH", "yt-dlp"),
		YTDLPCookiesFile:         envString("YTDLP_COOKIES_FILE", ""),
		YTDLPCookiesFromBrowser:  envString("YTDLP_COOKIES_FROM_BROWSER", ""),
		YTDLPReferer:             envString("YTDLP_REFERER", "https://www.bilibili.com/"),
		YTDLPUserAgent:           envString("YTDLP_USER_AGENT", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
		YTDLPExtraArgs:           envFields("YTDLP_EXTRA_ARGS"),
		BilibiliCookie:           envString("BILIBILI_COOKIE", ""),
		BilibiliQuality:          envInt("BILIBILI_QUALITY", 80),
		BilibiliQualityFallbacks: envIntList("BILIBILI_QUALITY_FALLBACKS", []int{80, 64, 32, 16}),
		FormatSelector:           envString("FORMAT_SELECTOR", "bv*[vcodec^=avc1]+ba[ext=m4a]/b[vcodec^=avc1]/bv*[vcodec^=avc1]+ba/bv*+ba/b"),
		FFmpegPath:               envString("FFMPEG_PATH", "ffmpeg"),
		MaxConcurrentJobs:        envInt("MAX_CONCURRENT_JOBS", 1),
		JobTimeout:               time.Duration(envInt("JOB_TIMEOUT_MINUTES", 90)) * time.Minute,
		AllowedHosts: strings.Split(envString("ALLOWED_HOSTS",
			"bilibili.com,www.bilibili.com,m.bilibili.com,b23.tv,youtube.com,www.youtube.com,m.youtube.com,music.youtube.com,youtu.be"), ","),
		R2Endpoint:        strings.TrimRight(envString("R2_ENDPOINT", ""), "/"),
		R2AccessKeyID:     envString("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey: envString("R2_SECRET_ACCESS_KEY", ""),
		R2Bucket:          envString("R2_BUCKET", ""),
		R2PublicBaseURL:   strings.TrimRight(envString("R2_PUBLIC_BASE_URL", ""), "/"),
		R2KeyPrefix:       strings.Trim(envString("R2_KEY_PREFIX", "vrchat"), "/"),
		R2CacheControl:    envString("R2_CACHE_CONTROL", "public, max-age=86400"),
		R2UploadTimeout:   time.Duration(envInt("R2_UPLOAD_TIMEOUT_SECONDS", 600)) * time.Second,
	}
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

func (c Config) r2Enabled() bool {
	return c.R2Endpoint != "" &&
		c.R2AccessKeyID != "" &&
		c.R2SecretAccessKey != "" &&
		c.R2Bucket != "" &&
		c.R2PublicBaseURL != ""
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

func envIntList(key string, fallback []int) []int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return append([]int(nil), fallback...)
	}
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	})
	var values []int
	for _, field := range fields {
		parsed, err := strconv.Atoi(strings.TrimSpace(field))
		if err != nil || parsed <= 0 {
			continue
		}
		values = append(values, parsed)
	}
	if len(values) == 0 {
		return append([]int(nil), fallback...)
	}
	return values
}

func envFields(key string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}
	return strings.Fields(value)
}
