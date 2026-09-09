import { describe, expect, it } from 'vitest';

import { createThemeManifest, resolveThemeBuild } from './theme-build';

describe('theme build configuration', () => {
  const availableThemes = ['base', 'daylight', 'phosphor', 'local'];

  it('uses the selected built-in theme and its preference-aware browser colors', () => {
    expect(resolveThemeBuild({
      MUSIC_LINK_THEMES: 'base, daylight',
      MUSIC_LINK_DEFAULT_THEME: 'daylight',
    }, availableThemes)).toMatchObject({
      themes: ['base', 'daylight'],
      defaultTheme: 'daylight',
      themeColors: { light: '#f7f0df', dark: '#17130f' },
    });
  });

  it('describes only emitted optional assets and scripts from selected themes', () => {
    expect(createThemeManifest(
      ['base', 'phosphor', 'local'],
      'base',
      new Set(['phosphor']),
    )).toEqual({
      version: 1,
      themes: {
        base: { colors: { light: '#171815', dark: '#171815' } },
        phosphor: {
          stylesheet: 'themes/phosphor/styles.css',
          script: true,
          colors: { light: '#000000', dark: '#000000' },
        },
        local: { stylesheet: 'themes/local/styles.css' },
      },
    });
  });

  it('requires a browser color for an unregistered local default theme', () => {
    expect(() => resolveThemeBuild({
      MUSIC_LINK_THEMES: 'base,local',
      MUSIC_LINK_DEFAULT_THEME: 'local',
    }, availableThemes)).toThrow('MUSIC_LINK_THEME_COLOR is required');
  });

  it('rejects selected names that cannot safely become asset paths', () => {
    expect(() => resolveThemeBuild({
      MUSIC_LINK_THEMES: 'base,<script>',
    }, ['base', '<script>'])).toThrow('Invalid MUSIC_LINK_THEMES name');
  });
});
