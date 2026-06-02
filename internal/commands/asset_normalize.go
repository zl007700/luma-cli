package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type normalizedUploadAsset struct {
	Path       string
	Width      int
	Height     int
	TargetW    int
	TargetH    int
	Normalized bool
	Cleanup    func()
}

func prepareVideoAssetForUpload(inputPath string) (*normalizedUploadAsset, error) {
	if !isVideoAssetFile(inputPath) {
		return &normalizedUploadAsset{Path: inputPath, Cleanup: func() {}}, nil
	}
	ffmpeg, err := installedFFmpegPath()
	if err != nil {
		return nil, err
	}
	ffprobe, err := installedFFprobePath(ffmpeg)
	if err != nil {
		return nil, err
	}
	info, err := probeMedia(ffprobe, inputPath)
	if err != nil {
		return nil, fmt.Errorf("probe video failed: %w", err)
	}
	targetW, targetH := target1080Size(info.Width, info.Height)
	if info.Width == targetW && info.Height == targetH && strings.EqualFold(filepath.Ext(inputPath), ".mp4") {
		return &normalizedUploadAsset{
			Path:    inputPath,
			Width:   info.Width,
			Height:  info.Height,
			TargetW: targetW,
			TargetH: targetH,
			Cleanup: func() {},
		}, nil
	}
	temp, err := os.CreateTemp("", "luma-asset-1080p-*.mp4")
	if err != nil {
		return nil, err
	}
	outputPath := temp.Name()
	_ = temp.Close()

	filter := fmt.Sprintf(
		"scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1,fps=30,format=yuv420p",
		targetW, targetH, targetW, targetH,
	)
	cmd := exec.Command(
		ffmpeg,
		"-y",
		"-i", inputPath,
		"-map", "0:v:0",
		"-map", "0:a?",
		"-vf", filter,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "18",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		outputPath,
	)
	if data, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(outputPath)
		return nil, fmt.Errorf("normalize video to 1080p failed: %w\n%s", err, strings.TrimSpace(string(data)))
	}
	return &normalizedUploadAsset{
		Path:       outputPath,
		Width:      info.Width,
		Height:     info.Height,
		TargetW:    targetW,
		TargetH:    targetH,
		Normalized: true,
		Cleanup: func() {
			_ = os.Remove(outputPath)
		},
	}, nil
}

func normalizeVideoAssetToFile(source, target string, mode os.FileMode) (bool, error) {
	prepared, err := prepareVideoAssetForUpload(source)
	if err != nil {
		return false, err
	}
	defer prepared.Cleanup()
	if !prepared.Normalized {
		return false, copyMaterialFileRaw(source, target, mode)
	}
	if err := copyMaterialFileRaw(prepared.Path, forceMP4Extension(target), 0644); err != nil {
		return false, err
	}
	return true, nil
}

func target1080Size(width, height int) (int, int) {
	if width > height {
		return 1920, 1080
	}
	return 1080, 1920
}

func isVideoAssetFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".mov", ".m4v", ".avi", ".mkv", ".webm":
		return true
	default:
		return false
	}
}

func forceMP4Extension(path string) string {
	ext := filepath.Ext(path)
	if strings.EqualFold(ext, ".mp4") {
		return path
	}
	return strings.TrimSuffix(path, ext) + ".mp4"
}
