# Themes

A theme is a directory containing `styles.css` and, optionally, `theme.ts`.
Styles define the visual tokens consumed by `../styles.css`; shared layout,
components, and accessibility rules stay there. The build bundles `theme.ts`
and its imports through the application's Vite build. Add third-party packages
to the root `package.json`; do not create a package manager inside a theme.

Every theme supplies these properties on `:root`:

```css
--color-scheme
--surface
--page-background
--surface-raised
--text
--muted
--accent
--accent-contrast
--font-display
--font-body
--radius
--artwork-treatment
--line
--surface-soft
--shadow
--danger
```

Theme colors are fixed by the selected stylesheet; artwork never changes them.
A theme can use private custom properties with `color-mix()`, and can include
media queries—for example, `daylight` changes its tokens for
`prefers-color-scheme: dark`.

For a local Node or Docker build, create `src/themes/local/styles.css`, then build with
`MUSIC_LINK_THEMES=base,local MUSIC_LINK_DEFAULT_THEME=local` and
`MUSIC_LINK_THEME_COLOR=<light-color>`. Add
`MUSIC_LINK_THEME_COLOR_DARK=<dark-color>` when it responds to a dark browser
preference. Contribute a theme by adding the same directory and stylesheet to
the repository. Nix builds can use the flake's `lib.<system>.makePackage` with
an overridden `src`; Git-backed flake sources include only tracked files, so
track the theme or use `path:` for development. See the root README for a
copyable custom-theme flake.

## Optional JavaScript

Theme JavaScript is trusted deployment code, not sandboxed content. It has the
same origin and browser privileges as the player, so operators must only build
and deploy themes they trust. Only scripts belonging to names in
`MUSIC_LINK_THEMES` are built. A missing `theme.ts` is valid. The application
loads the default theme's lazy chunk, or the matching chunk selected by Go's
validated `music-link-theme` metadata when `MUSIC_LINK_THEME` names an
additional theme.

`window.musicLink` is installed by the player before a theme script runs. It is
a frozen, versioned API; its snapshots are readonly and never expose Solid
signals or setters:

```ts
interface MusicLinkThemeAPI {
  readonly version: 1;
  readonly state: {
    readonly share: {
      readonly id: string;
      readonly description: string;
      readonly downloadable: boolean;
      readonly downloadsEnabled: boolean;
      readonly trackCount: number;
    };
    readonly song: {
      readonly id: string;
      readonly title: string;
      readonly artist: string;
      readonly album: string;
      readonly duration: number; // source metadata, in seconds
      readonly index: number;
    };
    readonly player: {
      readonly playing: boolean;
      readonly position: number; // current media time, in seconds
      readonly duration: number; // loaded media duration, or 0
      readonly volume: number;   // 0 through 1
    };
  } | null;
  readonly elements: {
    readonly page: HTMLElement | null;
    readonly player: HTMLElement | null;
    readonly artwork: HTMLImageElement | null;
    readonly audio: HTMLAudioElement | null;
  };
  subscribe(listener: (api: MusicLinkThemeAPI) => void): () => void;
}
```

`subscribe` invokes the listener immediately, then after state or element
changes, and returns an unsubscribe function. Elements can be `null` before
the player mounts or after it unmounts. For CSS and small DOM integrations,
stable hooks are also available as `[data-music-link="page"]`, `player`,
`artwork-frame`, `artwork`, and `audio`; the page hook exposes `data-playing="true|false"` and
`data-song-id`. Do not depend on internal classes or DOM nesting.

```ts
const stop = window.musicLink.subscribe(({ state, elements }) => {
  elements.page?.style.setProperty('--current-title-length', String(state?.song.title.length ?? 0));
});
addEventListener('pagehide', stop, { once: true });
```

For animation, honor `matchMedia('(prefers-reduced-motion: reduce)')` and stop
work when playback pauses. Web Audio analysis should be created only after
playback begins (a user gesture), use the existing `elements.audio`, reconnect
its source to `AudioContext.destination`, and catch unsupported or denied API
access without changing playback. Never create a second audio element, poll the
DOM, observe mutations, patch media methods, or alter autoplay behavior. See
`phosphor/theme.ts` for a small `AnalyserNode` example controlled by the
`--beat-glow` custom property.
