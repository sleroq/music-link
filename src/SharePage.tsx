import { Show, createEffect, createSignal } from 'solid-js';
import {
  publicUrls,
  type SharedMusic,
} from './share';
import Queue from './Queue';
import './styles.css';
import '@music-link/theme';
import {
  disconnectThemePlayer,
  publishThemeState,
  setThemeArtwork,
  setThemeAudio,
  setThemePage,
  setThemePlayer,
} from './theme-api';

const defaultVolume = 0.5;
const volumeStorageKey = 'music-link.volume';
const keyboardSeekStep = 5;
const keyboardVolumeStep = 0.05;

const savedVolume = () => {
  try {
    const stored = localStorage.getItem(volumeStorageKey);
    if (stored === null) return defaultVolume;
    const volume = Number(stored);
    return Number.isFinite(volume) && volume >= 0 && volume <= 1
      ? volume
      : defaultVolume;
  } catch {
    return defaultVolume;
  }
};

const saveVolume = (volume: number) => {
  try {
    localStorage.setItem(volumeStorageKey, String(volume));
  } catch {
    // Playback should still work when browser storage is unavailable.
  }
};

const clock = (seconds: number) => {
  if (!Number.isFinite(seconds)) return '0:00';
  const whole = Math.max(0, Math.floor(seconds));
  return `${Math.floor(whole / 60)}:${String(whole % 60).padStart(2, '0')}`;
};

export default function SharePage(props: { share: SharedMusic }) {
  const tracks = () => props.share.tracks;
  const isCollection = () => tracks().length > 1;
  const collectionTitle = () => {
    const sharedTracks = tracks();
    const album = sharedTracks[0].album;
    return album && sharedTracks.every((item) => item.album === album)
      ? album
      : 'A shared listening room';
  };
  const [current, setCurrent] = createSignal(0);
  const [playing, setPlaying] = createSignal(false);
  const [position, setPosition] = createSignal(0);
  const [duration, setDuration] = createSignal(0);
  const [volume, setVolume] = createSignal(savedVolume());
  const [audioError, setAudioError] = createSignal(false);
  const [artworkFailed, setArtworkFailed] = createSignal(false);
  let audio: HTMLAudioElement;

  const track = () => tracks()[current()];
  const seekable = () => Number.isFinite(duration()) && duration() > 0;
  const displayDuration = () =>
    seekable() ? duration() : Math.max(0, track().duration);

  const start = () => {
    setAudioError(false);
    void audio.play().catch(() => {
      setPlaying(false);
      setAudioError(true);
    });
  };

  const select = (index: number, autoplay = playing()) => {
    if (index < 0 || index >= tracks().length) return;
    setCurrent(index);
    setPosition(0);
    setDuration(0);
    setAudioError(false);
    queueMicrotask(() => {
      audio.load();
      if (autoplay) start();
    });
  };

  const next = (autoplay = playing()) => {
    if (current() < tracks().length - 1) select(current() + 1, autoplay);
    else setPlaying(false);
  };

  const seekTo = (value: number) => {
    if (!seekable() || !Number.isFinite(value)) return;
    const nextPosition = Math.min(Math.max(value, 0), duration());
    try {
      audio.currentTime = nextPosition;
      setPosition(nextPosition);
    } catch {
      // Media can become unavailable between choosing a position and assignment.
    }
  };

  const seek = (event: InputEvent & { currentTarget: HTMLInputElement }) => {
    seekTo(Number(event.currentTarget.value));
  };

  const updateDuration = () => {
    setDuration(
      Number.isFinite(audio.duration) && audio.duration > 0 ? audio.duration : 0,
    );
  };

  const setPlayerVolume = (value: number) => {
    if (!Number.isFinite(value)) return;
    const nextVolume = Math.min(Math.max(value, 0), 1);
    audio.volume = nextVolume;
    setVolume(nextVolume);
    saveVolume(nextVolume);
  };

  const updateVolume = (
    event: InputEvent & { currentTarget: HTMLInputElement },
  ) => {
    setPlayerVolume(Number(event.currentTarget.value));
  };

  const handlePlayerKeyDown = (event: KeyboardEvent) => {
    const target = event.target;
    if (target instanceof Element && target.closest(
      'button, a, input, select, textarea, summary, [contenteditable]:not([contenteditable="false"])',
    )) return;

    switch (event.key) {
      case ' ':
        event.preventDefault();
        if (!event.repeat) {
          if (playing()) audio.pause();
          else start();
        }
        break;
      case 'ArrowLeft':
        event.preventDefault();
        seekTo(audio.currentTime - keyboardSeekStep);
        break;
      case 'ArrowRight':
        event.preventDefault();
        seekTo(audio.currentTime + keyboardSeekStep);
        break;
      case 'ArrowUp':
        event.preventDefault();
        setPlayerVolume(volume() + keyboardVolumeStep);
        break;
      case 'ArrowDown':
        event.preventDefault();
        setPlayerVolume(volume() - keyboardVolumeStep);
        break;
    }
  };

  createEffect(
    () => track().title,
    (title) => {
      document.title = `${title} — music-link`;
    },
  );

  createEffect(() => track().id, () => {
    setArtworkFailed(false);
  });

  createEffect(
    () => ({
      share: {
        id: props.share.id,
        description: props.share.description,
        downloadable: props.share.downloadable,
        downloadsEnabled: props.share.downloadsEnabled,
        trackCount: tracks().length,
      },
      song: { ...track(), index: current() },
      player: {
        playing: playing(),
        position: position(),
        duration: duration(),
        volume: volume(),
      },
    }),
    publishThemeState,
  );

  createEffect(() => undefined, () => disconnectThemePlayer);

  createEffect(
    () => handlePlayerKeyDown,
    (handleKeyDown) => {
      document.addEventListener('keydown', handleKeyDown, true);
      return () => document.removeEventListener('keydown', handleKeyDown, true);
    },
  );

  return (
    <main
      class="page-shell"
      data-music-link="page"
      data-playing={playing() ? 'true' : 'false'}
      data-song-id={track().id}
      ref={setThemePage}
    >
      <section class="hero" id="player" data-music-link="player" ref={setThemePlayer}>
        <div class="artwork-frame" data-music-link="artwork-frame">
          <img
            class={{ artwork: true, 'is-missing': artworkFailed() }}
            data-music-link="artwork"
            ref={setThemeArtwork}
            src={publicUrls.artwork(track().id)}
            alt={`Artwork for ${track().album || track().title}`}
            onLoad={() => setArtworkFailed(false)}
            onError={() => {
              setArtworkFailed(true);
            }}
          />
          <span class="artwork-monogram" aria-hidden="true">M</span>
        </div>

        <div class="player-panel">
          <Show when={isCollection()}>
            <p class="eyebrow">Shared collection · {tracks().length} tracks</p>
          </Show>
          <h1>{collectionTitle()}</h1>
          <Show when={props.share.description}>
            <p class="description">{props.share.description}</p>
          </Show>

          <div class="now-playing" aria-live="polite">
            <p class="track-title">{track().title}</p>
            <p class="track-meta">
              {track().artist || 'Unknown artist'}
              <Show when={track().album}> · {track().album}</Show>
            </p>
          </div>

          <audio
            ref={(element) => {
              audio = element;
              audio.volume = volume();
              setThemeAudio(element);
            }}
            data-music-link="audio"
            src={publicUrls.stream(track().id)}
            preload="metadata"
            onPlay={() => setPlaying(true)}
            onPause={() => setPlaying(false)}
            onTimeUpdate={() =>
              setPosition(Number.isFinite(audio.currentTime) ? audio.currentTime : 0)
            }
            onLoadedMetadata={updateDuration}
            onDurationChange={updateDuration}
            onEnded={() => next(true)}
            onError={() => {
              setPlaying(false);
              setAudioError(true);
            }}
          />

          <div class="timeline">
            <input
              aria-label={`Seek in ${track().title}`}
              type="range"
              min="0"
              max={seekable() ? duration() : 1}
              step="0.1"
              value={seekable() ? Math.min(Math.max(position(), 0), duration()) : 0}
              style={{ '--progress': `${seekable() ? (position() / duration()) * 100 : 0}%` }}
              disabled={!seekable()}
              aria-disabled={seekable() ? 'false' : 'true'}
              aria-describedby="seek-status"
              onInput={seek}
            />
            <div class="time-row">
              <span>{clock(position())}</span>
              <span>{clock(displayDuration())}</span>
            </div>
            <span id="seek-status" class="visually-hidden" aria-live="polite">
              {seekable()
                ? 'Seek control ready.'
                : 'Seeking is unavailable until media duration loads.'}
            </span>
          </div>

          <div class="controls">
            <Show when={isCollection()}>
              <button
                class="icon-button"
                type="button"
                aria-label="Previous track"
                disabled={current() === 0}
                onClick={() => select(current() - 1)}
              >
                <span aria-hidden="true">‹</span>
              </button>
            </Show>
            <button
              class="play-button"
              type="button"
              aria-label={playing() ? 'Pause' : 'Play'}
              onClick={() => (playing() ? audio.pause() : start())}
            >
              <span aria-hidden="true">{playing() ? 'Ⅱ' : '▶'}</span>
            </button>
            <Show when={isCollection()}>
              <button
                class="icon-button"
                type="button"
                aria-label="Next track"
                disabled={current() === tracks().length - 1}
                onClick={() => next()}
              >
                <span aria-hidden="true">›</span>
              </button>
            </Show>
          </div>

          <div class="volume-control">
            <label for="volume">Volume</label>
            <input
              id="volume"
              aria-label="Volume"
              type="range"
              min="0"
              max="1"
              step="0.01"
              value={volume()}
              onInput={updateVolume}
            />
            <output for="volume">{Math.round(volume() * 100)}%</output>
          </div>

          <Show when={audioError()}>
            <p class="audio-error" role="status">This track could not be played. The share may no longer be available.</p>
          </Show>

          <div class="share-actions">
            <a href={publicUrls.stream(track().id)}>Open audio link</a>
            <a href={publicUrls.m3u(props.share.id)} download>Open M3U</a>
            <Show when={props.share.downloadable && props.share.downloadsEnabled}>
              <a href={publicUrls.mp3Download(props.share.id, current())} download>
                Download current MP3
              </a>
              <a href={publicUrls.download(props.share.id)} download>Download collection</a>
            </Show>
          </div>
        </div>
      </section>

      <Show when={isCollection()}>
        <Queue tracks={tracks()} current={current} onSelect={select} />
      </Show>
    </main>
  );
}
