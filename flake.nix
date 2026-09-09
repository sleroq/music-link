{
  description = "music-link static Navidrome share player";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs =
    { nixpkgs, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      # nixpkgs unstable 26.11 has removed x86_64-darwin entirely; importing
      # its package set fails before music-link's dependencies can evaluate.
      forAllSystems = nixpkgs.lib.genAttrs systems;
      packageLib = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          makePackage =
            {
              src ? ./.,
              themes ? [ "base" ],
              defaultTheme ? "base",
              themeColor ? null,
              themeColorDark ? null,
              npmDepsHash ? "sha256-lZykwnRQluN58byeMWNqOG0X6s1exLIYUnDaI0QAZfs=",
              version ? "0.0.0",
              pname ?
                if themes == [ "base" ] && defaultTheme == "base" then
                  "music-link"
                else if builtins.length themes == 1 then
                  "music-link-${defaultTheme}"
                else
                  "music-link-${defaultTheme}-with-${
                    nixpkgs.lib.concatStringsSep "-" (builtins.filter (theme: theme != defaultTheme) themes)
                  }",
            }:
            assert nixpkgs.lib.assertMsg (
              builtins.isList themes && themes != [ ]
            ) "music-link: themes must be a non-empty list";
            assert nixpkgs.lib.assertMsg (builtins.all (
              theme: builtins.isString theme && builtins.match "[A-Za-z0-9][A-Za-z0-9._-]*" theme != null
            ) themes) "music-link: every theme must be a safe theme name";
            assert nixpkgs.lib.assertMsg (
              builtins.length themes == builtins.length (nixpkgs.lib.unique themes)
            ) "music-link: themes must not contain duplicates";
            assert nixpkgs.lib.assertMsg (builtins.elem defaultTheme themes)
              "music-link: defaultTheme must be included in themes";
            assert nixpkgs.lib.assertMsg (
              builtins.elem defaultTheme [
                "base"
                "daylight"
                "phosphor"
              ]
              || themeColor != null
            ) "music-link: themeColor is required for a custom default theme";
            assert nixpkgs.lib.assertMsg (
              themeColor == null || builtins.isString themeColor
            ) "music-link: themeColor must be a string";
            assert nixpkgs.lib.assertMsg (
              themeColorDark == null || builtins.isString themeColorDark
            ) "music-link: themeColorDark must be a string";
            pkgs.buildNpmPackage {
              inherit
                npmDepsHash
                pname
                src
                version
                ;
              npmBuildScript = "build";
              env = {
                MUSIC_LINK_THEMES = nixpkgs.lib.concatStringsSep "," themes;
                MUSIC_LINK_DEFAULT_THEME = defaultTheme;
              }
              // nixpkgs.lib.optionalAttrs (themeColor != null) {
                MUSIC_LINK_THEME_COLOR = themeColor;
              }
              // nixpkgs.lib.optionalAttrs (themeColorDark != null) {
                MUSIC_LINK_THEME_COLOR_DARK = themeColorDark;
              };
              nativeBuildInputs = [
                pkgs.go
                pkgs.makeWrapper
              ];

              installPhase = ''
                runHook preInstall
                mkdir -p $out/bin $out/libexec $out/share/music-link
                cp -r dist/client $out/share/music-link/client
                go build -o $out/libexec/music-link ./cmd/music-link
                makeWrapper $out/libexec/music-link $out/bin/music-link \
                  --set-default MUSIC_LINK_SHELL $out/share/music-link/client/index.html
                runHook postInstall
              '';
            };
        }
      );
    in
    {
      lib = packageLib;

      packages = forAllSystems (
        system:
        let
          makePackage = packageLib.${system}.makePackage;
          base = makePackage { };
        in
        {
          inherit base;
          default = base;
          daylight = makePackage {
            themes = [ "daylight" ];
            defaultTheme = "daylight";
          };
          phosphor = makePackage {
            themes = [ "phosphor" ];
            defaultTheme = "phosphor";
          };
        }
      );
    };
}
