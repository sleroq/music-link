import type { SharedMusic, SharedTrack } from './share';
import { loadSelectedTheme } from 'virtual:music-link-theme-loader';

export interface MusicLinkThemeState {
  readonly share: Readonly<Pick<SharedMusic, 'id' | 'description' | 'downloadable' | 'downloadsEnabled'>> & {
    readonly trackCount: number;
  };
  readonly song: Readonly<SharedTrack> & {
    readonly index: number;
  };
  readonly player: {
    readonly playing: boolean;
    readonly position: number;
    readonly duration: number;
    readonly volume: number;
  };
}

export interface MusicLinkThemeElements {
  readonly page: HTMLElement | null;
  readonly player: HTMLElement | null;
  readonly artwork: HTMLImageElement | null;
  readonly audio: HTMLAudioElement | null;
}

export interface MusicLinkThemeAPI {
  readonly version: 1;
  readonly state: MusicLinkThemeState | null;
  readonly elements: MusicLinkThemeElements;
  subscribe(listener: (api: MusicLinkThemeAPI) => void): () => void;
}

declare global {
  interface Window {
    readonly musicLink: MusicLinkThemeAPI;
  }
}

let state: MusicLinkThemeState | null = null;
let page: HTMLElement | null = null;
let player: HTMLElement | null = null;
let artwork: HTMLImageElement | null = null;
let audio: HTMLAudioElement | null = null;
const listeners = new Set<(api: MusicLinkThemeAPI) => void>();

const elements = Object.freeze({
  get page() { return page; },
  get player() { return player; },
  get artwork() { return artwork; },
  get audio() { return audio; },
});

const api: MusicLinkThemeAPI = Object.freeze({
  version: 1 as const,
  get state() { return state; },
  elements,
  subscribe(listener: (current: MusicLinkThemeAPI) => void) {
    listeners.add(listener);
    listener(api);
    return () => listeners.delete(listener);
  },
});

if (!import.meta.env.SSR) {
  Object.defineProperty(window, 'musicLink', { value: api, enumerable: true });
  void loadSelectedTheme();
}

function notify() {
  for (const listener of listeners) listener(api);
}

export function publishThemeState(next: MusicLinkThemeState) {
  state = Object.freeze({
    share: Object.freeze(next.share),
    song: Object.freeze(next.song),
    player: Object.freeze(next.player),
  });
  notify();
}

export function setThemePage(element: HTMLElement) {
  page = element;
  notify();
}

export function setThemePlayer(element: HTMLElement) {
  player = element;
  notify();
}

export function setThemeArtwork(element: HTMLImageElement) {
  artwork = element;
  notify();
}

export function setThemeAudio(element: HTMLAudioElement) {
  audio = element;
  notify();
}

export function disconnectThemePlayer() {
  state = null;
  page = null;
  player = null;
  artwork = null;
  audio = null;
  notify();
}
