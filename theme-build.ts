export interface ThemeColors {
  light: string;
  dark: string;
}

export const builtInThemeColors = {
  base: { light: '#171815', dark: '#171815' },
  daylight: { light: '#f7f0df', dark: '#17130f' },
  phosphor: { light: '#000000', dark: '#000000' },
} satisfies Record<string, ThemeColors>;

function value(environment: NodeJS.ProcessEnv, name: string) {
  return environment[name]?.trim() || undefined;
}

function findBuiltInThemeColors(theme: string): ThemeColors | undefined {
  return Object.entries(builtInThemeColors).find(([name]) => name === theme)?.[1];
}
const safeThemeName = /^[A-Za-z0-9][A-Za-z0-9._-]*$/;

export function resolveThemeBuild(
  environment: NodeJS.ProcessEnv,
  availableThemes: readonly string[],
) {
  const themes = [
    ...new Set((value(environment, 'MUSIC_LINK_THEMES') ?? 'base').split(',').map((theme) => theme.trim()).filter(Boolean)),
  ];
  const defaultTheme = value(environment, 'MUSIC_LINK_DEFAULT_THEME') ?? 'base';
  const unsafeThemes = themes.filter((theme) => !safeThemeName.test(theme));
  if (unsafeThemes.length > 0) {
    throw new Error(`Invalid MUSIC_LINK_THEMES name: ${unsafeThemes.join(', ')}.`);
  }
  const invalidThemes = themes.filter((theme) => !availableThemes.includes(theme));

  if (invalidThemes.length > 0) {
    throw new Error(
      `Unknown MUSIC_LINK_THEMES value: ${invalidThemes.join(', ')}. Available themes: ${availableThemes.join(', ')}.`,
    );
  }
  if (!themes.includes(defaultTheme)) {
    throw new Error('MUSIC_LINK_DEFAULT_THEME must be included in MUSIC_LINK_THEMES.');
  }

  const builtInColors = findBuiltInThemeColors(defaultTheme);
  const light = value(environment, 'MUSIC_LINK_THEME_COLOR') ?? builtInColors?.light;
  if (!light) {
    throw new Error(
      `MUSIC_LINK_THEME_COLOR is required when MUSIC_LINK_DEFAULT_THEME=${defaultTheme} has no built-in browser theme color.`,
    );
  }

  return {
    themes,
    defaultTheme,
    themeColors: {
      light,
      dark: value(environment, 'MUSIC_LINK_THEME_COLOR_DARK') ?? builtInColors?.dark ?? light,
    },
  };
}

export interface ThemeManifestEntry {
  stylesheet?: string;
  script?: true;
  colors?: ThemeColors;
}

export interface ThemeManifest {
  version: 1;
  themes: Record<string, ThemeManifestEntry>;
}

function createThemeManifestEntry(
  theme: string,
  defaultTheme: string,
  themesWithScripts: ReadonlySet<string>,
): ThemeManifestEntry {
  const entry: ThemeManifestEntry = {};
  if (theme !== defaultTheme) entry.stylesheet = `themes/${theme}/styles.css`;
  if (themesWithScripts.has(theme)) entry.script = true;
  const colors = findBuiltInThemeColors(theme);
  if (colors) entry.colors = colors;
  return entry;
}

export function createThemeManifest(
  themes: readonly string[],
  defaultTheme: string,
  themesWithScripts: ReadonlySet<string>,
): ThemeManifest {
  return {
    version: 1,
    themes: Object.fromEntries(themes.map((theme) => [
      theme,
      createThemeManifestEntry(theme, defaultTheme, themesWithScripts),
    ])),
  };
}
