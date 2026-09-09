// Package navidrome owns the version-pinned public-share integration.
package navidrome

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Track is one track made available by a public Navidrome share.
type Track struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Artist   string  `json:"artist"`
	Album    string  `json:"album"`
	Duration float64 `json:"duration"`
}

// Share is the data the player needs to display and play a public share.
type Share struct {
	ID               string  `json:"id"`
	Description      string  `json:"description"`
	Downloadable     bool    `json:"downloadable"`
	DownloadsEnabled bool    `json:"downloadsEnabled"`
	Tracks           []Track `json:"tracks"`
}

type sharePayload struct {
	ID           string  `json:"id"`
	Description  string  `json:"description"`
	Downloadable bool    `json:"downloadable"`
	Tracks       []Track `json:"tracks"`
}

type appConfig struct {
	EnableDownloads bool `json:"enableDownloads"`
}

var scriptBlock = regexp.MustCompile(`(?is)<script\s*>(.*?)</script\s*>`)

func exactAssignment(name string) *regexp.Regexp {
	return regexp.MustCompile(
		`^\s*window\.` + regexp.QuoteMeta(name) + `\s*=\s*("(\\.|[^"\\])*"|null)\s*;?\s*$`,
	)
}

func injectedJSON[T any](html, name string) (T, error) {
	var value T
	assignment := exactAssignment(name)
	literals := make([]string, 0, 1)
	for _, script := range scriptBlock.FindAllStringSubmatch(html, -1) {
		match := assignment.FindStringSubmatch(script[1])
		if match != nil {
			literals = append(literals, match[1])
		}
	}
	if len(literals) != 1 || literals[0] == "null" {
		return value, fmt.Errorf("missing %s injection", name)
	}

	var encoded string
	if err := json.Unmarshal([]byte(literals[0]), &encoded); err != nil {
		return value, fmt.Errorf("decode %s string: %w", name, err)
	}
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		return value, fmt.Errorf("decode %s payload: %w", name, err)
	}
	return value, nil
}

// ParseShareHTML extracts Navidrome's inert, injected public-share payload. It
// must be checked against Navidrome whenever that server is upgraded.
func ParseShareHTML(html string) (Share, error) {
	payload, err := injectedJSON[sharePayload](html, "__SHARE_INFO__")
	if err != nil {
		return Share{}, err
	}
	config, err := injectedJSON[appConfig](html, "__APP_CONFIG__")
	if err != nil {
		return Share{}, err
	}
	if payload.ID == "" || len(payload.Tracks) == 0 {
		return Share{}, fmt.Errorf("share has no tracks")
	}

	for index := range payload.Tracks {
		if payload.Tracks[index].Title == "" {
			payload.Tracks[index].Title = "Untitled track"
		}
	}
	return Share{
		ID:               payload.ID,
		Description:      payload.Description,
		Downloadable:     payload.Downloadable,
		DownloadsEnabled: config.EnableDownloads,
		Tracks:           payload.Tracks,
	}, nil
}

// Client loads public share data from Navidrome.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewClient makes a client for a Navidrome origin, such as
// http://127.0.0.1:4533.
func NewClient(rawURL string) (*Client, error) {
	baseURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse Navidrome URL %q: %w", rawURL, err)
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("navidrome URL %q must be absolute", rawURL)
	}
	return &Client{baseURL: baseURL, httpClient: http.DefaultClient}, nil
}

// Load fetches and decodes a public share.
func (c *Client) Load(ctx context.Context, id string) (Share, error) {
	shareURL := c.baseURL.JoinPath("share", id)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, shareURL.String(), nil)
	if err != nil {
		return Share{}, fmt.Errorf("create Navidrome share request: %w", err)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return Share{}, fmt.Errorf("fetch Navidrome share: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return Share{}, &ResponseError{StatusCode: response.StatusCode}
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Share{}, fmt.Errorf("read Navidrome share: %w", err)
	}
	share, err := ParseShareHTML(string(body))
	if err != nil {
		return Share{}, fmt.Errorf("parse Navidrome share: %w", err)
	}
	return share, nil
}

// DownloadStream copies a public share's signed track stream.
func (c *Client) DownloadStream(ctx context.Context, trackID string, destination io.Writer) error {
	return c.download(ctx, StreamPath(trackID), destination)
}

// DownloadArtwork copies the artwork associated with a signed shared track.
func (c *Client) DownloadArtwork(ctx context.Context, trackID string, destination io.Writer) error {
	return c.download(ctx, ArtworkPath(trackID), destination)
}

func (c *Client) download(ctx context.Context, path string, destination io.Writer) error {
	assetPath, err := url.Parse(path)
	if err != nil {
		return fmt.Errorf("parse Navidrome asset path: %w", err)
	}
	assetURL := c.baseURL.JoinPath(strings.TrimPrefix(assetPath.Path, "/"))
	assetURL.RawQuery = assetPath.RawQuery
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL.String(), nil)
	if err != nil {
		return fmt.Errorf("create Navidrome asset request: %w", err)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("fetch Navidrome asset: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return &ResponseError{StatusCode: response.StatusCode}
	}
	if _, err := io.Copy(destination, response.Body); err != nil {
		return fmt.Errorf("read Navidrome asset: %w", err)
	}
	return nil
}

// ResponseError preserves an upstream public-share status for the browser.
type ResponseError struct {
	StatusCode int
}

func (error *ResponseError) Error() string {
	return fmt.Sprintf("Navidrome share response: %d", error.StatusCode)
}

// StreamPath is Navidrome's public stream capability route.
func StreamPath(trackID string) string {
	return "/share/s/" + url.PathEscape(trackID)
}

// ArtworkPath is Navidrome's public artwork capability route.
func ArtworkPath(trackID string) string {
	return "/share/img/" + url.PathEscape(trackID) + "?size=600&square=true"
}

// ValidShareID matches the public route accepted by Caddy and Navidrome.
func ValidShareID(id string) bool {
	return shareID.MatchString(id)
}

var shareID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

// ShareIDFromPath returns the single share ID owned by this service.
func ShareIDFromPath(path string) string {
	id := strings.TrimSuffix(strings.TrimPrefix(path, "/share/"), "/")
	if strings.Contains(id, "/") || !ValidShareID(id) {
		return ""
	}
	return id
}
