FROM node:22-bookworm AS frontend

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY src ./src
COPY tsconfig.json vite.config.ts theme-build.ts ./
ARG MUSIC_LINK_THEMES=base
ARG MUSIC_LINK_DEFAULT_THEME=base
ARG MUSIC_LINK_THEME_COLOR
ARG MUSIC_LINK_THEME_COLOR_DARK
ENV MUSIC_LINK_THEMES=$MUSIC_LINK_THEMES
ENV MUSIC_LINK_DEFAULT_THEME=$MUSIC_LINK_DEFAULT_THEME
ENV MUSIC_LINK_THEME_COLOR=$MUSIC_LINK_THEME_COLOR
ENV MUSIC_LINK_THEME_COLOR_DARK=$MUSIC_LINK_THEME_COLOR_DARK
RUN npm run build

FROM golang:1.26-bookworm AS backend

WORKDIR /app
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /music-link ./cmd/music-link

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates ffmpeg \
    && rm -rf /var/lib/apt/lists/*

COPY --from=frontend /app/dist/client /app/client
COPY --from=backend /music-link /music-link

ENV MUSIC_LINK_ADDR=0.0.0.0:8787
ENV MUSIC_LINK_SHELL=/app/client/index.html

EXPOSE 8787
USER 65532:65532
ENTRYPOINT ["/music-link"]
