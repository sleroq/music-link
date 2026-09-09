package sharepage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sleroq/music-link/internal/navidrome"
)

type loader struct{}

func (loader) Load(context.Context, string) (navidrome.Share, error) {
	return navidrome.Share{
		ID:          "share-id",
		Description: "Music </script> description",
		Tracks: []navidrome.Track{{
			ID: "signed-track", Title: "Track", Artist: "Artist", Album: "Album",
		}},
	}, nil
}

type generatedMP3 struct {
	data string
}

func (generator generatedMP3) Generate(_ context.Context, _ navidrome.Track) (io.ReadSeekCloser, error) {
	return readSeekCloser{Reader: strings.NewReader(generator.data)}, nil
}

type readSeekCloser struct {
	*strings.Reader
}

func (readSeekCloser) Close() error { return nil }

func TestServerEmbedsMetadataAndShareData(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	shellPath := filepath.Join(directory, "index.html")
	if err := os.WriteFile(shellPath, []byte("<html><head><title>Shell</title></head><body></body></html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(loader{}, generatedMP3{}, "https://music.example.com", shellPath, "", "", "")
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/share/share-id", nil))
	body := response.Body.String()
	for _, want := range []string{
		`property="og:title" content="Track — Artist"`,
		`property="og:audio" content="https://music.example.com/share/share-id/preview.mp3"`,
		`property="og:image" content="https://music.example.com/share/img/signed-track?size=600&amp;square=true"`,
		`id="music-link-share" type="application/json"`,
		`"id":"share-id"`,
		`\u003c/script\u003e`,
		`</head><body>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response body does not contain %q:\n%s", want, body)
		}
	}
	if response.Code != http.StatusOK {
		t.Errorf("status = %d", response.Code)
	}
	if got := response.Header().Get("X-Robots-Tag"); got != "noindex, nofollow" {
		t.Errorf("X-Robots-Tag = %q", got)
	}
}

func TestTelegramReceivesTaggedMP3(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	shellPath := filepath.Join(directory, "index.html")
	if err := os.WriteFile(shellPath, []byte("<html><head></head><body></body></html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(loader{}, generatedMP3{data: "tagged mp3"}, "https://music.example.com", shellPath, "", "", "")
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	redirect := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/share/share-id", nil)
	request.Header.Set("User-Agent", "TelegramBot (like TwitterBot)")
	server.ServeHTTP(redirect, request)
	if redirect.Code != http.StatusTemporaryRedirect || redirect.Header().Get("Location") != "/share/share-id/preview.mp3" {
		t.Fatalf("Telegram redirect = %d %q", redirect.Code, redirect.Header().Get("Location"))
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, redirect.Header().Get("Location"), nil))
	if response.Code != http.StatusOK || response.Body.String() != "tagged mp3" {
		t.Fatalf("MP3 response = %d %q", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "audio/mpeg" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := response.Header().Get("Content-Disposition"); got != `inline; filename="Artist - Track.mp3"` {
		t.Errorf("Content-Disposition = %q", got)
	}
}

func TestMP3DownloadRequiresDownloadPermission(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	shellPath := filepath.Join(directory, "index.html")
	if err := os.WriteFile(shellPath, []byte("<html><head></head><body></body></html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(loader{}, generatedMP3{}, "https://music.example.com", shellPath, "", "", "")
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/share/share-id/tracks/0.mp3", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestServerServesStaticAssets(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	shellPath := filepath.Join(directory, "index.html")
	if err := os.WriteFile(shellPath, []byte("<html><head></head><body></body></html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(directory, "assets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "assets", "player.js"), []byte("player"), 0o600); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(loader{}, generatedMP3{}, "https://music.example.com", shellPath, "", "", "")
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/_music-link/assets/player.js", nil))
	if response.Code != http.StatusOK || response.Body.String() != "player" {
		t.Fatalf("static response = %d %q", response.Code, response.Body.String())
	}
}

func TestServerLinksSelectedTheme(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	shellPath := filepath.Join(directory, "index.html")
	if err := os.WriteFile(shellPath, []byte(`<html><head><meta name="theme-color" media="(prefers-color-scheme: light)" content="#171815" data-music-link-theme-color="light"><meta name="theme-color" media="(prefers-color-scheme: dark)" content="#171815" data-music-link-theme-color="dark"></head><body></body></html>`), 0o600); err != nil {
		t.Fatal(err)
	}
	themeDirectory := filepath.Join(directory, "themes", "daylight")
	if err := os.MkdirAll(themeDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDirectory, "styles.css"), []byte(":root {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := `{"version":1,"themes":{"daylight":{"stylesheet":"themes/daylight/styles.css","script":true,"colors":{"light":"#f7f0df","dark":"#17130f"}}}}`
	if err := os.WriteFile(filepath.Join(directory, "themes", "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(loader{}, generatedMP3{}, "https://music.example.com", shellPath, "daylight", "", "")
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/share/share-id", nil))
	if !strings.Contains(response.Body.String(), `href="/_music-link/themes/daylight/styles.css"`) {
		t.Errorf("response does not link the selected theme:\n%s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `<meta name="music-link-theme" content="daylight">`) {
		t.Errorf("response does not select the matching theme script:\n%s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `content="#f7f0df" data-music-link-theme-color="light"`) {
		t.Errorf("response does not use the selected theme browser color:\n%s", response.Body.String())
	}
}

func TestServerRejectsThemeArtifactsOutsideKnownPaths(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	shellPath := filepath.Join(directory, "index.html")
	shell := `<html><head><meta content="#000" data-music-link-theme-color="light"><meta content="#000" data-music-link-theme-color="dark"></head></html>`
	if err := os.WriteFile(shellPath, []byte(shell), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(directory, "themes"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `{"version":1,"themes":{"local":{"stylesheet":"../index.html","colors":{"light":"#000","dark":"#000"}}}}`
	if err := os.WriteFile(filepath.Join(directory, "themes", "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := NewServer(loader{}, generatedMP3{}, "https://music.example.com", shellPath, "local", "", ""); err == nil || !strings.Contains(err.Error(), "not an emitted runtime theme") {
		t.Fatalf("NewServer() error = %v", err)
	}
}
