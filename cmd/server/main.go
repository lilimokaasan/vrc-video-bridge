package main

import (
	"log"
	"net/http"

	"bili-vrc-streamer/internal/streamer"
)

func main() {
	cfg := streamer.LoadConfig()
	app, err := streamer.NewServer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("bili-vrc-streamer listening on %s", cfg.Addr)
	log.Printf("public base url: %s", cfg.PublicBaseURL)
	log.Fatal(http.ListenAndServe(cfg.Addr, app.Routes()))
}
