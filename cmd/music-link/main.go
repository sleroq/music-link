package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/sleroq/music-link/internal/navidrome"
	"github.com/sleroq/music-link/internal/sharepage"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	navidromeURL := os.Getenv("MUSIC_LINK_NAVIDROME_URL")
	if navidromeURL == "" {
		navidromeURL = "http://127.0.0.1:4533"
	}
	siteURL := os.Getenv("MUSIC_LINK_SITE_URL")
	if siteURL == "" {
		return fmt.Errorf("MUSIC_LINK_SITE_URL is required")
	}
	shellPath := os.Getenv("MUSIC_LINK_SHELL")
	if shellPath == "" {
		shellPath = "/srv/music-link/dist/client/index.html"
	}
	address := os.Getenv("MUSIC_LINK_ADDR")
	if address == "" {
		address = "127.0.0.1:8787"
	}

	client, err := navidrome.NewClient(navidromeURL)
	if err != nil {
		return err
	}
	server, err := sharepage.NewServer(
		client,
		siteURL,
		shellPath,
		os.Getenv("MUSIC_LINK_THEME"),
		os.Getenv("MUSIC_LINK_THEME_COLOR"),
		os.Getenv("MUSIC_LINK_THEME_COLOR_DARK"),
	)
	if err != nil {
		return err
	}
	log.Printf("music-link metadata server listening on %s", address)
	return http.ListenAndServe(address, server)
}
