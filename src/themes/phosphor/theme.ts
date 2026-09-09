export {};

const root = document.documentElement;
const reducedMotion = matchMedia('(prefers-reduced-motion: reduce)');
let status: 'idle' | 'starting' | 'ready' | 'failed' | 'disposed' = 'idle';
let context: AudioContext | undefined;
let source: MediaElementAudioSourceNode | undefined;
let analyser: AnalyserNode | undefined;
let levels: Uint8Array<ArrayBuffer> | undefined;
let audio: HTMLAudioElement | undefined;
let frame = 0;
function disposed() {
  return status === 'disposed';
}

function reset() {
  root.style.setProperty('--beat-glow', '0%');
}

function stopVisual() {
  if (frame) cancelAnimationFrame(frame);
  frame = 0;
  reset();
}

function draw() {
  const state = window.musicLink.state;
  if (!analyser || !levels || !state?.player.playing || reducedMotion.matches) {
    stopVisual();
    return;
  }
  analyser.getByteFrequencyData(levels);
  const bassBins = Math.max(1, Math.floor(levels.length / 12));
  let total = 0;
  for (let index = 0; index < bassBins; index += 1) total += levels[index];
  const strength = total / bassBins / 255;
  root.style.setProperty('--beat-glow', `${Math.round(strength * 42)}%`);
  frame = requestAnimationFrame(draw);
}

async function initialize(element: HTMLAudioElement) {
  status = 'starting';
  try {
    context = new AudioContext();
    await context.resume();
    if (disposed()) {
      await context.close();
      return;
    }
    analyser = context.createAnalyser();
    analyser.fftSize = 256;
    analyser.smoothingTimeConstant = 0.72;
    source = context.createMediaElementSource(element);
    // Keep playback on a direct route; analysis is a passive branch.
    source.connect(context.destination);
    source.connect(analyser);
    levels = new Uint8Array(analyser.frequencyBinCount);
    status = 'ready';
  } catch (error) {
    if (disposed()) return;
    status = 'failed';
    reset();
    console.warn('Phosphor audio analysis is unavailable.', error);
    if (!source && context) {
      void context.close().catch((closeError) => {
        console.warn('Phosphor audio analysis could not close its context.', closeError);
      });
    }
  }
}

async function update() {
  if (status === 'disposed' || status === 'failed') return;
  const state = window.musicLink.state;
  if (reducedMotion.matches || !state?.player.playing) {
    stopVisual();
    return;
  }
  if (status === 'idle' && audio) await initialize(audio);
  if (status === 'ready' && context?.state === 'suspended') {
    try {
      await context.resume();
    } catch (error) {
      console.warn('Phosphor audio analysis could not resume.', error);
      return;
    }
  }
  if (status === 'ready' && !frame) frame = requestAnimationFrame(draw);
}

function onPlay() {
  void update();
}
const onPause = stopVisual;

function bindAudio(element: HTMLAudioElement | null) {
  if (!element || element === audio) return;
  audio?.removeEventListener('play', onPlay);
  audio?.removeEventListener('pause', onPause);
  audio = element;
  audio.addEventListener('play', onPlay);
  audio.addEventListener('pause', onPause);
}

const unsubscribe = window.musicLink.subscribe(({ elements }) => {
  bindAudio(elements.audio);
  void update();
});
function onMotionChange() {
  void update();
}
reducedMotion.addEventListener('change', onMotionChange);

function dispose() {
  if (status === 'disposed') return;
  status = 'disposed';
  stopVisual();
  unsubscribe();
  reducedMotion.removeEventListener('change', onMotionChange);
  audio?.removeEventListener('play', onPlay);
  audio?.removeEventListener('pause', onPause);
  source?.disconnect();
  analyser?.disconnect();
  if (context) {
    void context.close().catch((error) => {
      console.warn('Phosphor audio analysis could not close its context.', error);
    });
  }
}

addEventListener('pagehide', (event: PageTransitionEvent) => {
  stopVisual();
  if (event.persisted) {
    if (context?.state === 'running') {
      void context.suspend().catch((error) => {
        console.warn('Phosphor audio analysis could not suspend.', error);
      });
    }
  } else {
    dispose();
  }
});
addEventListener('pageshow', () => void update());
