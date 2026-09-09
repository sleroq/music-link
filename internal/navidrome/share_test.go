package navidrome

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func injection(name, value string) string {
	return fmt.Sprintf(`<script>window.%s = %q;</script>`, name, value)
}

func TestParseShareHTML(t *testing.T) {
	t.Parallel()
	html := injection("__SHARE_INFO__", `{"id":"share-id","description":"A description","downloadable":true,"tracks":[{"id":"signed-track","title":"Track","artist":"Artist","album":"Album","duration":183.25}]}`) +
		injection("__APP_CONFIG__", `{"enableDownloads":true}`)

	share, err := ParseShareHTML(html)
	if err != nil {
		t.Fatalf("ParseShareHTML() error = %v", err)
	}
	if share.ID != "share-id" || !share.Downloadable || !share.DownloadsEnabled {
		t.Fatalf("ParseShareHTML() share = %#v", share)
	}
	if len(share.Tracks) != 1 || share.Tracks[0] != (Track{
		ID: "signed-track", Title: "Track", Artist: "Artist", Album: "Album", Duration: 183.25,
	}) {
		t.Fatalf("ParseShareHTML() tracks = %#v", share.Tracks)
	}
}

func TestParseShareHTMLRejectsExecutableScript(t *testing.T) {
	t.Parallel()
	html := injection("__SHARE_INFO__", `{"id":"share-id","tracks":[{"id":"signed-track"}]}`)
	html = html[:len(html)-len("</script>")] + `window.attack = true;</script>` +
		injection("__APP_CONFIG__", `{"enableDownloads":true}`)

	if _, err := ParseShareHTML(html); err == nil {
		t.Fatal("ParseShareHTML() succeeded for an executable script")
	}
}

func TestPublicPaths(t *testing.T) {
	t.Parallel()
	if got := StreamPath("signed token"); got != "/share/s/signed%20token" {
		t.Fatalf("StreamPath() = %q", got)
	}
	if got := ArtworkPath("signed token"); got != "/share/img/signed%20token?size=600&square=true" {
		t.Fatalf("ArtworkPath() = %q", got)
	}
}

func TestClientDownloadsAssetsBelowConfiguredBasePath(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if got := request.URL.EscapedPath(); got != "/navidrome/share/img/signed%20token" {
			t.Errorf("asset path = %q", got)
		}
		if got := request.URL.RawQuery; got != "size=600&square=true" {
			t.Errorf("asset query = %q", got)
		}
		_, _ = response.Write([]byte("artwork"))
	}))
	defer server.Close()

	client, err := NewClient(server.URL + "/navidrome")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	var artwork bytes.Buffer
	if err := client.DownloadArtwork(context.Background(), "signed token", &artwork); err != nil {
		t.Fatalf("DownloadArtwork() error = %v", err)
	}
	if got := artwork.String(); got != "artwork" {
		t.Fatalf("artwork = %q", got)
	}
}
