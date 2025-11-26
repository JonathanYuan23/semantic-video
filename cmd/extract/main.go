package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"semanticvideo/internal/extract"

	cli "github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:  "semanticvideo-extract",
		Usage: "Extract frames for every video in a directory",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "video-dir",
				Aliases: []string{"i"},
				Usage:   "Directory containing source videos",
				Value:   "videos",
			},
			&cli.StringFlag{
				Name:    "frames-dir",
				Aliases: []string{"o"},
				Usage:   "Directory where extracted frames will be written",
				Value:   "frames",
			},
			&cli.Float64Flag{
				Name:    "frame-rate",
				Aliases: []string{"r"},
				Usage:   "Frame sampling rate in frames per second",
				Value:   1.0,
			},
			&cli.IntFlag{
				Name:  "frame-width",
				Usage: "Output frame width in pixels",
				Value: 224,
			},
			&cli.IntFlag{
				Name:  "frame-height",
				Usage: "Output frame height in pixels",
				Value: 224,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			frameRate := cmd.Float64("frame-rate")
			if frameRate <= 0 {
				return cli.Exit("frame-rate must be greater than zero", 2)
			}

			frameWidth := cmd.Int("frame-width")
			if frameWidth <= 0 {
				return cli.Exit("frame-width must be greater than zero", 2)
			}

			frameHeight := cmd.Int("frame-height")
			if frameHeight <= 0 {
				return cli.Exit("frame-height must be greater than zero", 2)
			}

			videoDir := cmd.String("video-dir")
			framesDir := cmd.String("frames-dir")

			cfg := extract.Config{
				FrameRate: frameRate,
				FrameSize: [2]int{frameWidth, frameHeight},
			}

			return processDirectory(videoDir, framesDir, cfg)
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func processDirectory(videoDir, framesDir string, cfg extract.Config) error {
	entries, err := os.ReadDir(videoDir)
	if err != nil {
		return fmt.Errorf("read video directory: %w", err)
	}

	var processed int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !isVideoFile(entry.Name()) {
			continue
		}

		inputPath := filepath.Join(videoDir, entry.Name())
		log.Printf("Extracting frames for %s", inputPath)

		if err := extract.ExtractFramesForVideo(inputPath, framesDir, cfg); err != nil {
			return fmt.Errorf("extract frames for %s: %w", entry.Name(), err)
		}

		processed++
	}

	if processed == 0 {
		return fmt.Errorf("no video files found in %s", videoDir)
	}

	return nil
}

func isVideoFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp4", ".mov", ".mkv", ".avi", ".m4v", ".webm":
		return true
	default:
		return false
	}
}
