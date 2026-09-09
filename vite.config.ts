import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig, type Plugin } from 'vite';
import solid from '@solidjs/vite-plugin';
import { createThemeManifest, resolveThemeBuild } from './theme-build.js';

const themesDirectory = fileURLToPath(new URL('./src/themes', import.meta.url));
const availableThemes = readdirSync(themesDirectory, { withFileTypes: true })
  .filter((entry) => entry.isDirectory())
  .map((entry) => entry.name)
  .filter((theme) => existsSync(resolve(themesDirectory, theme, 'styles.css')));
const { themes, defaultTheme, themeColors } = resolveThemeBuild(process.env, availableThemes);
const themesWithScripts = new Set(themes.filter((theme) =>
  existsSync(resolve(themesDirectory, theme, 'theme.ts')),
));
const virtualThemeLoader = 'virtual:music-link-theme-loader';
const resolvedVirtualThemeLoader = `\0${virtualThemeLoader}`;

function packageThemes(): Plugin {
  let isServerBuild = false;

  return {
    name: 'music-link-themes',
    configResolved(config) {
      isServerBuild = config.build.ssr === true;
    },
    resolveId(id) {
      if (id === virtualThemeLoader) return resolvedVirtualThemeLoader;
    },
    load(id) {
      if (id !== resolvedVirtualThemeLoader) return;
      if (isServerBuild) return 'export const loadSelectedTheme = () => undefined;';
      const cases = [...themesWithScripts].map((theme) =>
        `case ${JSON.stringify(theme)}: return report(import(${JSON.stringify(resolve(themesDirectory, theme, 'theme.ts'))}));`,
      ).join('\n');
      return `const report = (loading) => loading.then(() => undefined).catch((error) => {
        console.error('Could not load the selected music-link theme.', error);
      });
      export const loadSelectedTheme = () => {
        const selected = document.querySelector('meta[name="music-link-theme"]')?.content ?? ${JSON.stringify(defaultTheme)};
        switch (selected) { ${cases} default: return Promise.resolve(); }
      };`;
    },
    generateBundle(_options, bundle) {
      if (isServerBuild) return;

      for (const theme of themes) {
        if (theme === defaultTheme) continue;
        this.emitFile({
          type: 'asset',
          fileName: `themes/${theme}/styles.css`,
          source: readFileSync(resolve(themesDirectory, theme, 'styles.css')),
        });
      }
      this.emitFile({
        type: 'asset',
        fileName: 'themes/manifest.json',
        source: JSON.stringify(createThemeManifest(themes, defaultTheme, themesWithScripts)),
      });
    },
  };
}

export default defineConfig({
  base: '/_music-link/',
  plugins: [solid({ start: true }), packageThemes()],
  define: {
    __MUSIC_LINK_THEME_COLOR_LIGHT__: JSON.stringify(themeColors.light),
    __MUSIC_LINK_THEME_COLOR_DARK__: JSON.stringify(themeColors.dark),
  },
  resolve: {
    alias: {
      '@music-link/theme': resolve(themesDirectory, defaultTheme, 'styles.css'),
    },
  },
  build: {
    target: 'esnext',
    // Keep images as asset files instead of inlining them into the JS bundle.
    assetsInlineLimit: 0,
  },
});
