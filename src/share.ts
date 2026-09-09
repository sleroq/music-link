export interface SharedTrack {
  id: string;
  title: string;
  artist: string;
  album: string;
  duration: number;
}

export interface SharedMusic {
  id: string;
  description: string;
  downloadable: boolean;
  downloadsEnabled: boolean;
  tracks: SharedTrack[];
}

export function pageShare(): SharedMusic {
  const payload = document.getElementById('music-link-share')!.textContent!;
  // SAFETY: Go serializes SharedMusic with encoding/json into this inert element before the bundle loads.
  return JSON.parse(payload) as SharedMusic;
}

export const publicUrls = {
  stream: (trackToken: string) => `/share/s/${encodeURIComponent(trackToken)}`,
  artwork: (trackToken: string) =>
    `/share/img/${encodeURIComponent(trackToken)}?size=600&square=true`,
  download: (shareId: string) => `/share/d/${encodeURIComponent(shareId)}`,
  m3u: (shareId: string) => `/share/${encodeURIComponent(shareId)}/m3u`,
};
