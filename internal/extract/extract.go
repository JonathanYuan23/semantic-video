package extract

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type Config struct {
	FrameRate float64 `json:"frame_rate"`
	FrameSize [2]int  `json:"frame_size"`
}

func ExtractFramesForVideo(inputPath, framesRoot string, cfg Config) error {
	base := filepath.Base(inputPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	videoDir := filepath.Join(framesRoot, name)

	if err := os.MkdirAll(videoDir, 0o755); err != nil {
		return fmt.Errorf("create frames dir: %w", err)
	}

	fpsStr := strconv.FormatFloat(cfg.FrameRate, 'f', -1, 64)
	// Preserve aspect ratio: scale to fit inside a square target, then pad to square
	target := cfg.FrameSize[0]
	if target <= 0 {
		target = 384
	}
	scaleStr := fmt.Sprintf("%d:%d", target, target)
	padStr := fmt.Sprintf("%d:%d:(%d-iw)/2:(%d-ih)/2", target, target, target, target)

	outputPattern := filepath.Join(videoDir, "frame_%05d.jpg")

	return ffmpeg.
		Input(inputPath).
		// Sample at cfg.FrameRate fps
		Filter("fps", ffmpeg.Args{fpsStr}).
		// Resize preserving aspect, then pad to square
		Filter("scale", ffmpeg.Args{scaleStr}, ffmpeg.KwArgs{"force_original_aspect_ratio": "decrease"}).
		Filter("pad", ffmpeg.Args{padStr}, ffmpeg.KwArgs{"color": "black"}).
		// Write each frame as an image
		Output(outputPattern, ffmpeg.KwArgs{"qscale:v": 1}).
		OverWriteOutput().
		Run()
}
