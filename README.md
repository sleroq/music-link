# music-link

music-link is a statically built SolidJS player for public Navidrome shares. Go adds share metadata and serves the player, while Caddy sends media requests directly to Navidrome.

Single-track previews and per-track MP3 downloads are transcoded at request time with FFmpeg. The generated MP3 includes title, artist, album, and embedded Navidrome artwork; the container image and Nix package include FFmpeg automatically.

Build it with Node `20.19+` or `22.12+` and Go `1.26+`: run `npm install`, `npm run build`, and `go build ./cmd/music-link`. Tests and checks are available through `npm test`, `npm run typecheck`, and `go test ./cmd/... ./internal/...`.

Deploy the frontend beside the Go server, set `MUSIC_LINK_SITE_URL` and `MUSIC_LINK_SHELL`, then configure the included `Caddyfile` for your Navidrome host. Docker Compose and Nix flake packages are also included.

The built-in themes are `base`, `daylight`, and `phosphor`; select them at build time with `MUSIC_LINK_THEMES` and `MUSIC_LINK_DEFAULT_THEME`. See [`src/themes/README.md`](src/themes/README.md) for custom themes and runtime theme selection.

Navidrome has no supported public-share metadata API, so the Go adapter safely extracts its injected JSON without executing the page. Verify the types and fixtures in `internal/navidrome` whenever Navidrome is upgraded.
