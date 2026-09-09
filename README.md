# music-link

A statically built SolidJS player for public Navidrome shares. Go renders
share metadata and serves the player assets; Caddy sends Navidrome media routes
straight to Navidrome.

## Build

Requires Node `20.19+` or `22.12+` and Go 1.26+.

```sh
npm install
npm test
npm run typecheck
npm run build
go test ./cmd/... ./internal/...
go build -o /usr/local/bin/music-link ./cmd/music-link
```

## Themes

The player has no user-facing theme switcher: its default theme is chosen when the
frontend is built. `base` preserves the original appearance. `daylight` is a
paper-like theme that follows the browser's dark-mode preference, and `phosphor`
is a high-contrast terminal theme. Phosphor's optional theme script uses real
Web Audio frequency data to pulse its background while music plays. It stays
static when reduced motion is requested or audio analysis is unavailable.

`MUSIC_LINK_THEMES` is the comma-separated set of source themes to include;
`MUSIC_LINK_DEFAULT_THEME` must name one of them. The default is bundled into
the player stylesheet. Its browser chrome color is selected automatically for
built-in themes; a local default must also set `MUSIC_LINK_THEME_COLOR` (and
optionally `MUSIC_LINK_THEME_COLOR_DARK`).

```sh
MUSIC_LINK_THEMES=base,daylight \
MUSIC_LINK_DEFAULT_THEME=daylight \
npm run build
```

Additional selected themes are emitted at
`dist/client/themes/<name>/styles.css`; an optional `theme.ts` and its
dependencies are emitted as a lazy application chunk for any selected theme
that has one. Unselected theme scripts are not built. To use one, build it alongside a
different default, then set `MUSIC_LINK_THEME=<name>` when starting the Go
server. The server verifies the theme and its exact artifacts against the build
manifest, links its CSS, selects its matching application-owned JS chunk, and
updates browser chrome colors from that same manifest. A local
extra theme must also set `MUSIC_LINK_THEME_COLOR` (and optionally
`MUSIC_LINK_THEME_COLOR_DARK`) at server startup. This is host-side deployment
selection, not a user-facing runtime switcher.

```sh
MUSIC_LINK_THEMES=base,phosphor MUSIC_LINK_DEFAULT_THEME=base npm run build
MUSIC_LINK_THEME=phosphor MUSIC_LINK_SHELL=$PWD/dist/client/index.html \
  MUSIC_LINK_SITE_URL=https://music.example.com /usr/local/bin/music-link
```

Add a local theme at `src/themes/<name>/styles.css`, optionally add `theme.ts`,
and include its name in the build variables. Theme scripts use dependencies
installed in the application's root `package.json`; themes do not have separate
package managers. If it is the default,
set `MUSIC_LINK_THEME_COLOR` to its light browser-chrome color. The small token
contract and an example of preference-aware colors are documented in
[`src/themes/README.md`](src/themes/README.md), including the stable browser API
and semantic DOM hooks. Theme JavaScript is trusted deployment code with the
same page privileges as the player; only deploy themes you trust.

### Nix

The flake exposes minimal packages containing one built-in theme each. `default`
is an alias for `base`, preserving the original package behavior:

```sh
nix build .#base
nix build .#daylight
nix build .#phosphor
```

Downstream flakes can build any included set through
`lib.<system>.makePackage`. The selected `defaultTheme` is compiled into the
main stylesheet; the other themes remain available for host-side selection with
`MUSIC_LINK_THEME`.

The flake exposes packages and its builder for `x86_64-linux`, `aarch64-linux`,
and `aarch64-darwin`. Its nixpkgs unstable input has removed
`x86_64-darwin`; Intel macOS consumers must use a nixpkgs release that still
supports that platform.

For example:

```nix
{
  inputs.music-link.url = "github:sleroq/music-link";

  outputs = { self, music-link, ... }:
    let
      system = "x86_64-linux";
    in {
      packages.${system}.default = music-link.lib.${system}.makePackage {
        src = ./.;
        themes = [ "base" "local" ];
        defaultTheme = "local";
        themeColor = "#f4efe2";
        themeColorDark = "#181512";
      };
    };
}
```

The supplied source must be a music-link source tree containing
`src/themes/<name>/styles.css` and, optionally, `theme.ts`. `themeColor` and
`themeColorDark` map to `MUSIC_LINK_THEME_COLOR` and
`MUSIC_LINK_THEME_COLOR_DARK`; only `themeColor` is required for a custom
default. Override `npmDepsHash` if the source changes `package-lock.json` or its
dependencies. Git-backed flakes only copy tracked files, so custom theme files
must be committed or otherwise tracked. The same applies to this repository's
implementation files: until they are tracked, use the path form of every build
command during development (for example, `nix build path:.#daylight`) so Nix
receives the complete current source tree.

For the multi-theme package above, starting the packaged server with
`MUSIC_LINK_THEME=base` selects the emitted base theme. When selecting a custom
non-default theme at runtime, also provide its browser colors through
`MUSIC_LINK_THEME_COLOR` and optionally `MUSIC_LINK_THEME_COLOR_DARK`.

## Deploy

Build the frontend, put it at the path configured by `MUSIC_LINK_SHELL`, and
run Go alongside Navidrome:

```sh
npm run build
mkdir -p /srv/music-link/dist
cp -r dist/client /srv/music-link/dist/client

MUSIC_LINK_SITE_URL=https://music.example.com \
MUSIC_LINK_SHELL=/srv/music-link/dist/client/index.html \
/usr/local/bin/music-link
```

`MUSIC_LINK_ADDR` defaults to `127.0.0.1:8787` and
`MUSIC_LINK_NAVIDROME_URL` to `http://127.0.0.1:4533`. Replace
`music.example.com` in `Caddyfile` and reload Caddy. Caddy proxies
`/share/s/*`, `/share/img/*`, `/share/d/*`, and `/share/<id>/m3u` directly to
Navidrome; all other requests go to Go.

### Docker

`compose.yml` runs music-link only; configure Caddy yourself with the included
`Caddyfile`.

```sh
MUSIC_LINK_SITE_URL=https://music.example.com \
MUSIC_LINK_NAVIDROME_URL=http://host.docker.internal:4533 \
MUSIC_LINK_THEMES=base,phosphor \
MUSIC_LINK_DEFAULT_THEME=phosphor \
docker compose up -d --build
```

Docker receives the build variables above as build arguments. A local theme is
available to Docker only when it is present in the Docker build context. Set
`MUSIC_LINK_THEME=phosphor` in the Compose environment to activate an emitted
non-default theme.

## Development

For a remote Navidrome instance, build and run Go locally, then start the
development Caddy proxy:

```sh
npm run build
MUSIC_LINK_SITE_URL=http://localhost:8080 \
MUSIC_LINK_NAVIDROME_URL=https://navidrome.example.com \
MUSIC_LINK_SHELL=$PWD/dist/client/index.html \
go run ./cmd/music-link

NAVIDROME_HOST=navidrome.example.com docker compose -f compose.dev.yml up
```

Open `http://localhost:8080/share/<share-id>`.

## Notes

Go adds Open Graph metadata and embeds the share payload for Solid. Single-track
`TelegramBot.*` crawler requests redirect to that track's direct audio URL.

Navidrome has no supported public-share metadata API. The Go adapter extracts
its inert injected JSON without executing the page. It is version-pinned: verify
`internal/navidrome` types and fixtures whenever Navidrome is upgraded.
