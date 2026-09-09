# music-link

`music-link` is a statically built SolidJS 2 player for public Navidrome shares. Go serves dynamic `/share/<id>` pages and `/_music-link/` assets. Caddy sends `/share/s/*`, `/share/img/*`, `/share/d/*`, and `/share/<id>/m3u` directly to Navidrome; it proxies everything else to Go.

The Go server securely extracts Navidrome's injected share data in `internal/navidrome/share.go`; it never renders or executes the fetched HTML. It serializes that version-pinned payload into the page for Solid. Navidrome remains responsible for validity, expiry, revocation, media, and download permissions. Keep the Go adapter isolated and verify its asserted types and fixtures whenever Navidrome is upgraded.

Run `go test ./cmd/... ./internal/...`, `npm test`, `npm run typecheck`, `npm run lint`, and `npm run build` after functional changes.
