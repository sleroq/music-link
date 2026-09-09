import { For } from 'solid-js';
import type { SharedTrack } from './share';

const formatDuration = (seconds: number) => {
  if (!Number.isFinite(seconds)) return '0:00';
  const whole = Math.max(0, Math.floor(seconds));
  return `${Math.floor(whole / 60)}:${String(whole % 60).padStart(2, '0')}`;
};

export default function Queue(props: {
  tracks: SharedTrack[];
  current: () => number;
  onSelect: (index: number) => void;
}) {
  return (
    <section class="queue" aria-labelledby="queue-heading">
      <div class="queue-heading">
        <p class="eyebrow">Up next</p>
        <h2 id="queue-heading">The listening queue</h2>
      </div>
      <ol>
        <For each={props.tracks}>
          {(item, index) => (
            <li>
              <button
                type="button"
                class={{ 'queue-track': true, active: index() === props.current() }}
                aria-current={index() === props.current() ? 'true' : undefined}
                onClick={() => props.onSelect(index())}
              >
                <span class="track-number">{String(index() + 1).padStart(2, '0')}</span>
                <span class="queue-copy">
                  <strong>{item.title}</strong>
                  <span>{item.artist || item.album || 'Unknown artist'}</span>
                </span>
                <span class="queue-duration">{formatDuration(item.duration)}</span>
              </button>
            </li>
          )}
        </For>
      </ol>
    </section>
  );
}
