// Package taggedmp3 produces broadly compatible MP3 files from shared tracks.
package taggedmp3

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sleroq/music-link/internal/navidrome"
)

// Source supplies the audio and artwork protected by a Navidrome share.
type Source interface {
	DownloadStream(context.Context, string, io.Writer) error
	DownloadArtwork(context.Context, string, io.Writer) error
}

// Generator uses FFmpeg to transcode shared audio and attach its metadata.
type Generator struct {
	source Source
	ffmpeg string
}

// NewGenerator creates an MP3 generator using the named FFmpeg executable.
func NewGenerator(source Source, ffmpeg string) *Generator {
	return &Generator{source: source, ffmpeg: ffmpeg}
}

// Generate returns a seekable temporary MP3. Closing it removes all temporary files.
func (generator *Generator) Generate(ctx context.Context, track navidrome.Track) (io.ReadSeekCloser, error) {
	directory, err := os.MkdirTemp("", "music-link-mp3-*")
	if err != nil {
		return nil, fmt.Errorf("create MP3 workspace: %w", err)
	}
	removeDirectory := true
	defer func() {
		if removeDirectory {
			_ = os.RemoveAll(directory)
		}
	}()

	audioPath := filepath.Join(directory, "audio")
	if err := download(ctx, audioPath, func(destination io.Writer) error {
		return generator.source.DownloadStream(ctx, track.ID, destination)
	}); err != nil {
		return nil, fmt.Errorf("download shared audio: %w", err)
	}
	artworkPath := filepath.Join(directory, "artwork")
	if err := download(ctx, artworkPath, func(destination io.Writer) error {
		return generator.source.DownloadArtwork(ctx, track.ID, destination)
	}); err != nil {
		return nil, fmt.Errorf("download shared artwork: %w", err)
	}

	outputPath := filepath.Join(directory, "track.mp3")
	arguments := []string{
		"-nostdin", "-v", "error", "-y",
		"-i", audioPath,
		"-i", artworkPath,
		"-map", "0:a:0", "-map", "1:v:0",
		"-map_metadata", "0",
		"-c:a", "libmp3lame", "-b:a", "192k",
		"-c:v", "mjpeg",
		"-id3v2_version", "3",
		"-metadata", "title=" + track.Title,
		"-metadata", "artist=" + track.Artist,
		"-metadata", "album=" + track.Album,
		"-metadata:s:v", "title=Album cover",
		"-metadata:s:v", "comment=Cover (front)",
		"-disposition:v", "attached_pic",
		outputPath,
	}
	command := exec.CommandContext(ctx, generator.ffmpeg, arguments...)
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("encode tagged MP3: %w: %s", err, strings.TrimSpace(string(output)))
	}
	file, err := os.Open(outputPath)
	if err != nil {
		return nil, fmt.Errorf("open tagged MP3: %w", err)
	}
	removeDirectory = false
	return &temporaryFile{File: file, directory: directory}, nil
}

func download(ctx context.Context, path string, copyTo func(io.Writer) error) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create temporary input: %w", err)
	}
	if err := copyTo(file); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary input: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

type temporaryFile struct {
	*os.File
	directory string
}

func (file *temporaryFile) Close() error {
	closeErr := file.File.Close()
	removeErr := os.RemoveAll(file.directory)
	if closeErr != nil {
		return closeErr
	}
	if removeErr != nil {
		return fmt.Errorf("remove MP3 workspace: %w", removeErr)
	}
	return nil
}
