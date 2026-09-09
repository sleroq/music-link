package navidrome

import (
	"fmt"
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
