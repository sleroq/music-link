// Package sharepage renders public-share HTML around the static Solid player.
package sharepage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sleroq/music-link/internal/navidrome"
)

// Loader supplies current public-share data.
type Loader interface {
	Load(context.Context, string) (navidrome.Share, error)
}

// MP3Generator produces a seekable tagged MP3 for one shared track.
type MP3Generator interface {
	Generate(context.Context, navidrome.Track) (io.ReadSeekCloser, error)
}

// Server renders dynamic metadata while the player itself remains static.
type Server struct {
	loader   Loader
	mp3      MP3Generator
	mp3Slots chan struct{}
	siteURL  string
	shell    string
	static   http.Handler
	theme    string
}

type themeColors struct {
	Light string `json:"light"`
	Dark  string `json:"dark"`
}

var crawlerUserAgents = []*regexp.Regexp{
	regexp.MustCompile(`TelegramBot.*`),
}

var themeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type themeManifest struct {
	Version int                           `json:"version"`
	Themes  map[string]themeManifestEntry `json:"themes"`
}

type themeManifestEntry struct {
	Stylesheet string       `json:"stylesheet"`
	Script     bool         `json:"script"`
	Colors     *themeColors `json:"colors"`
}

// NewServer loads the generated static document shell once at startup.
func NewServer(loader Loader, mp3 MP3Generator, siteURL, shellPath, theme, lightThemeColor, darkThemeColor string) (*Server, error) {
	if mp3 == nil {
		return nil, fmt.Errorf("MP3 generator is required")
	}
	siteURL = strings.TrimRight(siteURL, "/")
	parsedURL, err := url.Parse(siteURL)
	if err != nil {
		return nil, fmt.Errorf("parse public site URL %q: %w", siteURL, err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("public site URL %q must be absolute", siteURL)
	}
	shell, err := os.ReadFile(shellPath)
	if err != nil {
		return nil, fmt.Errorf("read static document shell: %w", err)
	}
	if !strings.Contains(string(shell), "</head>") {
		return nil, fmt.Errorf("static document shell has no closing head tag")
	}
	if theme != "" {
		if !themeName.MatchString(theme) {
			return nil, fmt.Errorf("selected theme %q has an invalid name", theme)
		}
		manifestPath := filepath.Join(filepath.Dir(shellPath), "themes", "manifest.json")
		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("read theme manifest: %w", err)
		}
		var manifest themeManifest
		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			return nil, fmt.Errorf("decode theme manifest: %w", err)
		}
		entry, included := manifest.Themes[theme]
		expectedStylesheet := filepath.ToSlash(filepath.Join("themes", theme, "styles.css"))
		if manifest.Version != 1 || !included || entry.Stylesheet != expectedStylesheet {
			return nil, fmt.Errorf("selected theme %q is not an emitted runtime theme", theme)
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(shellPath), filepath.FromSlash(entry.Stylesheet))); err != nil {
			return nil, fmt.Errorf("selected theme stylesheet %q: %w", entry.Stylesheet, err)
		}
		colors := themeColors{}
		if entry.Colors != nil {
			colors = *entry.Colors
		}
		if lightThemeColor != "" {
			colors.Light = lightThemeColor
		}
		if darkThemeColor != "" {
			colors.Dark = darkThemeColor
		}
		if colors.Light == "" {
			return nil, fmt.Errorf("MUSIC_LINK_THEME_COLOR is required for selected local theme %q", theme)
		}
		if colors.Dark == "" {
			colors.Dark = colors.Light
		}
		shell, err = setThemeColors(string(shell), colors)
		if err != nil {
			return nil, err
		}
	}
	static := http.StripPrefix("/_music-link/", http.FileServer(http.Dir(filepath.Dir(shellPath))))
	return &Server{
		loader:   loader,
		mp3:      mp3,
		mp3Slots: make(chan struct{}, 2),
		siteURL:  siteURL,
		shell:    string(shell),
		static:   static,
		theme:    theme,
	}, nil
}

func setThemeColors(shell string, colors themeColors) ([]byte, error) {
	for scheme, color := range map[string]string{"light": colors.Light, "dark": colors.Dark} {
		marker := `data-music-link-theme-color="` + scheme + `"`
		markerIndex := strings.Index(shell, marker)
		if markerIndex == -1 {
			return nil, fmt.Errorf("static document shell has no %s theme-color marker", scheme)
		}
		contentStart := strings.LastIndex(shell[:markerIndex], `content="`)
		if contentStart == -1 {
			return nil, fmt.Errorf("static document shell has no %s theme-color content", scheme)
		}
		contentStart += len(`content="`)
		contentEnd := strings.Index(shell[contentStart:markerIndex], `"`)
		if contentEnd == -1 {
			return nil, fmt.Errorf("static document shell has an invalid %s theme-color", scheme)
		}
		shell = shell[:contentStart] + html.EscapeString(color) + shell[contentStart+contentEnd:]
	}
	return []byte(shell), nil
}

// ServeHTTP serves public share pages and the player assets they reference.
func (server *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if strings.HasPrefix(request.URL.Path, "/_music-link/") {
		server.static.ServeHTTP(response, request)
		return
	}
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		response.Header().Set("Allow", "GET, HEAD")
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	response.Header().Set("X-Robots-Tag", "noindex, nofollow")
	mp3Request, servesMP3 := parseMP3Request(request.URL.Path)
	id := mp3Request.shareID
	if !servesMP3 {
		id = navidrome.ShareIDFromPath(request.URL.Path)
	}
	if id == "" {
		http.NotFound(response, request)
		return
	}
	share, err := server.loader.Load(request.Context(), id)
	if err != nil {
		server.writeLoadError(response, err)
		return
	}
	if share.ID != id {
		http.NotFound(response, request)
		return
	}
	if servesMP3 {
		server.serveMP3(response, request, share, mp3Request)
		return
	}
	if len(share.Tracks) == 1 && isCrawler(request.UserAgent()) {
		http.Redirect(response, request, "/share/"+url.PathEscape(share.ID)+"/preview.mp3", http.StatusTemporaryRedirect)
		return
	}
	page, err := server.render(share)
	if err != nil {
		http.Error(response, "could not render share", http.StatusInternalServerError)
		return
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Cache-Control", "no-store")
	_, _ = response.Write(page)
}

type mp3Request struct {
	shareID  string
	track    int
	download bool
}

func parseMP3Request(path string) (mp3Request, bool) {
	remainder, included := strings.CutPrefix(path, "/share/")
	if !included {
		return mp3Request{}, false
	}
	parts := strings.Split(remainder, "/")
	if len(parts) == 2 && navidrome.ValidShareID(parts[0]) && parts[1] == "preview.mp3" {
		return mp3Request{shareID: parts[0]}, true
	}
	if len(parts) != 3 || !navidrome.ValidShareID(parts[0]) || parts[1] != "tracks" || !strings.HasSuffix(parts[2], ".mp3") {
		return mp3Request{}, false
	}
	track, err := strconv.Atoi(strings.TrimSuffix(parts[2], ".mp3"))
	if err != nil || track < 0 {
		return mp3Request{}, false
	}
	return mp3Request{shareID: parts[0], track: track, download: true}, true
}

func (server *Server) serveMP3(response http.ResponseWriter, request *http.Request, share navidrome.Share, media mp3Request) {
	if media.track >= len(share.Tracks) || (media.download && (!share.Downloadable || !share.DownloadsEnabled)) {
		http.NotFound(response, request)
		return
	}
	track := share.Tracks[media.track]
	select {
	case server.mp3Slots <- struct{}{}:
		defer func() { <-server.mp3Slots }()
	default:
		response.Header().Set("Retry-After", "10")
		http.Error(response, "audio conversion is busy", http.StatusServiceUnavailable)
		return
	}
	file, err := server.mp3.Generate(request.Context(), track)
	if err != nil {
		http.Error(response, "could not prepare audio", http.StatusBadGateway)
		return
	}
	defer func() { _ = file.Close() }()

	disposition := "inline"
	if media.download {
		disposition = "attachment"
	}
	response.Header().Set("Content-Type", "audio/mpeg")
	response.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": mp3Filename(track)}))
	response.Header().Set("Cache-Control", "private, no-store")
	http.ServeContent(response, request, mp3Filename(track), time.Time{}, file)
}

func mp3Filename(track navidrome.Track) string {
	name := track.Title
	if track.Artist != "" {
		name = track.Artist + " - " + name
	}
	name = strings.Map(func(character rune) rune {
		if character < ' ' || character == '/' || character == '\\' {
			return '_'
		}
		return character
	}, name)
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Track"
	}
	return name + ".mp3"
}

func isCrawler(userAgent string) bool {
	for _, crawlerUserAgent := range crawlerUserAgents {
		if crawlerUserAgent.MatchString(userAgent) {
			return true
		}
	}
	return false
}

func (server *Server) writeLoadError(response http.ResponseWriter, err error) {
	var upstreamError *navidrome.ResponseError
	if errors.As(err, &upstreamError) && upstreamError.StatusCode >= http.StatusBadRequest && upstreamError.StatusCode < http.StatusInternalServerError {
		http.Error(response, "share unavailable", upstreamError.StatusCode)
		return
	}
	http.Error(response, "share unavailable", http.StatusBadGateway)
}

type metadata struct {
	Title       string
	Description string
	URL         string
	Image       string
	Audio       string
	Type        string
}

var metadataTemplate = template.Must(template.New("metadata").Parse(`<meta property="og:title" content="{{.Title}}">
<meta property="og:description" content="{{.Description}}">
<meta property="og:type" content="{{.Type}}">
<meta property="og:url" content="{{.URL}}">
<meta property="og:image" content="{{.Image}}">
<meta property="og:image:alt" content="Artwork for {{.Title}}">
<meta property="og:audio" content="{{.Audio}}">
<meta property="og:audio:secure_url" content="{{.Audio}}">
<meta property="og:audio:type" content="audio/mpeg">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="{{.Title}}">
<meta name="twitter:description" content="{{.Description}}">
<meta name="twitter:image" content="{{.Image}}">
<title>{{.Title}} — music-link</title>
`))

func (server *Server) render(share navidrome.Share) ([]byte, error) {
	data, err := json.Marshal(share)
	if err != nil {
		return nil, fmt.Errorf("encode embedded share data: %w", err)
	}
	meta := server.metadata(share)
	var injected bytes.Buffer
	if err := metadataTemplate.Execute(&injected, meta); err != nil {
		return nil, fmt.Errorf("render metadata: %w", err)
	}
	if server.theme != "" {
		injected.WriteString(`<link rel="stylesheet" href="/_music-link/`)
		injected.WriteString("themes/")
		injected.WriteString(server.theme)
		injected.WriteString("/styles.css")
		injected.WriteString(`">`)
		injected.WriteString(`<meta name="music-link-theme" content="`)
		injected.WriteString(server.theme)
		injected.WriteString(`">`)
	}
	// encoding/json escapes '<', '>' and '&', so untrusted share text cannot end
	// this inert JSON script element.
	injected.WriteString(`<script id="music-link-share" type="application/json">`)
	injected.Write(data)
	injected.WriteString(`</script>`)

	before, after, _ := strings.Cut(server.shell, "</head>")
	return []byte(before + injected.String() + "</head>" + after), nil
}

func (server *Server) metadata(share navidrome.Share) metadata {
	track := share.Tracks[0]
	title := track.Title
	if track.Artist != "" {
		title += " — " + track.Artist
	}
	descriptionParts := make([]string, 0, 3)
	if track.Album != "" {
		descriptionParts = append(descriptionParts, track.Album)
	}
	if share.Description != "" {
		descriptionParts = append(descriptionParts, share.Description)
	}
	if len(share.Tracks) > 1 {
		descriptionParts = append(descriptionParts, fmt.Sprintf("%d shared tracks", len(share.Tracks)))
	}
	description := strings.Join(descriptionParts, " · ")
	if description == "" {
		description = "1 shared track"
	}
	shareURL := server.siteURL + "/share/" + url.PathEscape(share.ID)
	return metadata{
		Title:       title,
		Description: description,
		URL:         shareURL,
		Image:       server.siteURL + navidrome.ArtworkPath(track.ID),
		Audio:       shareURL + "/preview.mp3",
		Type:        openGraphType(share),
	}
}

func openGraphType(share navidrome.Share) string {
	if len(share.Tracks) == 1 {
		return "music.song"
	}
	return "website"
}
