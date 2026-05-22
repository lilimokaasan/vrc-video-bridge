package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"bili-vrc-streamer/internal/streamer"
)

func main() {
	downloadURL := flag.String("download", "", "download one video URL and exit")
	format := flag.String("format", "mp4", "output format for direct download: mp4 or hls")
	outputDir := flag.String("output", "downloads", "output directory for direct downloads")
	checkR2 := flag.Bool("check-r2", false, "upload a tiny healthcheck object to configured R2 storage and exit")
	flag.Parse()

	if err := streamer.LoadDotEnv(".env"); err != nil {
		log.Fatal(err)
	}

	cfg := streamer.LoadConfig()
	app, err := streamer.NewServer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if *checkR2 {
		publicURL, err := app.CheckStorage()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stdout, "r2_ok: %s\n", publicURL)
		return
	}

	url := *downloadURL
	if url == "" && flag.NArg() > 0 {
		url = flag.Arg(0)
	}
	if url != "" {
		result, err := app.DirectDownload(streamer.DirectDownloadOptions{
			URL:       url,
			Format:    streamer.OutputFormat(*format),
			OutputDir: *outputDir,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stdout, "downloaded: %s\n", result.OutputPath)
		if result.DirectURL != "" {
			fmt.Fprintf(os.Stdout, "direct_url: %s\n", result.DirectURL)
		}
		if result.PlaybackURL != "" {
			fmt.Fprintf(os.Stdout, "playback_url: %s\n", result.PlaybackURL)
		}
		return
	}

	log.Printf("bili-vrc-streamer listening on %s", cfg.Addr)
	log.Printf("public base url: %s", cfg.PublicBaseURL)
	log.Fatal(http.ListenAndServe(cfg.Addr, app.Routes()))
}
