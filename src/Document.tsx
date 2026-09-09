import { HydrationScript } from '@solidjs/web';
import type { ParentProps } from 'solid-js';
import { themeColors } from './theme-color';

export default function Document(props: ParentProps) {
  return (
    <html lang="en">
      <head>
        <meta charset="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <meta name="theme-color" media="(prefers-color-scheme: light)" content={themeColors.light} data-music-link-theme-color="light" />
        <meta name="theme-color" media="(prefers-color-scheme: dark)" content={themeColors.dark} data-music-link-theme-color="dark" />
        <meta name="robots" content="noindex, nofollow" />
        <title>music-link — Navidrome share</title>
        <HydrationScript />
      </head>
      <body>{props.children}</body>
    </html>
  );
}
