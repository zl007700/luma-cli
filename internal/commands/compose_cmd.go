package commands

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/clientruntime"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
	"github.com/luma-cli/lumer-cli/project"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const defaultCoverFontResourceID = "font_22b2e39414"

func cmdBGM(args []string) {
	if len(args) < 1 || args[0] != "mix" {
		fmt.Println("usage: luma-cli bgm mix <video> --bgm <file_or_resource_id> [--output <mp4>] [--voice-volume 1.0] [--bgm-volume 0.25]")
		return
	}
	parsed := cmdutil.Parse(args[1:])
	videoPath := parsed.Pos(0)
	bgmValue := parsed.String("bgm", "")
	if videoPath == "" || bgmValue == "" {
		fmt.Println("usage: luma-cli bgm mix <video> --bgm <file_or_resource_id> [--output <mp4>]")
		return
	}
	outputPath := parsed.String("output", "bgm_video.mp4")
	voiceVolume := parsed.String("voice-volume", "1.0")
	bgmVolume := parsed.String("bgm-volume", "0.25")
	cfg := loadConfig()
	bgmPath, err := resolveLocalCachedOrCloudResource(bgmValue, cfg)
	if err != nil {
		fmt.Printf("Error: resolve bgm failed: %v\n", err)
		return
	}
	absOut, err := absoluteOutputPath(outputPath)
	if err != nil {
		fmt.Printf("Error: bad output path: %v\n", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(absOut), 0755); err != nil {
		fmt.Printf("Error: create output dir failed: %v\n", err)
		return
	}
	ffmpeg, err := installedFFmpegPath()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	filter := fmt.Sprintf("[0:a]volume=%s[a0];[1:a]volume=%s,aloop=loop=-1:size=2e+09[a1];[a0][a1]amix=inputs=2:duration=first:dropout_transition=2[aout]", voiceVolume, bgmVolume)
	cmd := exec.Command(ffmpeg, "-y", "-i", videoPath, "-i", bgmPath, "-filter_complex", filter, "-map", "0:v", "-map", "[aout]", "-c:v", "copy", "-c:a", "aac", "-shortest", absOut)
	if data, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Error: ffmpeg bgm mix failed: %v\n%s\n", err, string(data))
		return
	}
	recordProjectArtifact("bgm", absOut, "bgm.mix")
	writeSimpleResult(map[string]any{"output_path": absOut, "bgm_path": bgmPath})
}

func cmdCover(args []string) {
	if len(args) < 1 {
		printCoverUsage()
		return
	}
	switch args[0] {
	case "frame":
		cmdCoverFrame(args[1:])
	case "render":
		cmdCoverRender(args[1:])
	default:
		printCoverUsage()
	}
}

func cmdCoverFrame(raw []string) {
	parsed := cmdutil.Parse(raw)
	videoPath := parsed.Pos(0)
	if videoPath == "" {
		fmt.Println("usage: luma-cli cover frame <video> [--time 1.0] [--output cover_frame.png]")
		return
	}
	outputPath := parsed.String("output", "cover_frame.png")
	seek := parsed.String("time", "1.0")
	absOut, err := absoluteOutputPath(outputPath)
	if err != nil {
		fmt.Printf("Error: bad output path: %v\n", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(absOut), 0755); err != nil {
		fmt.Printf("Error: create output dir failed: %v\n", err)
		return
	}
	ffmpeg, err := installedFFmpegPath()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	cmd := exec.Command(ffmpeg, "-y", "-ss", seek, "-i", videoPath, "-frames:v", "1", absOut)
	if data, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Error: ffmpeg frame extract failed: %v\n%s\n", err, string(data))
		return
	}
	recordProjectArtifact("cover_frame", absOut, "cover.frame")
	writeSimpleResult(map[string]any{"output_path": absOut})
}

func cmdCoverRender(raw []string) {
	parsed := cmdutil.Parse(raw)
	imagePath := parsed.String("image", "")
	if imagePath == "" {
		imagePath = parsed.Pos(0)
	}
	if imagePath == "" {
		fmt.Println("usage: luma-cli cover render <image> --title <text> [--subtitle <text>] [--output title_cover.jpg]")
		return
	}
	outputPath := parsed.String("output", "title_cover.jpg")
	title := parsed.String("title", "")
	subtitle := parsed.String("subtitle", "")
	font := parsed.String("font", "")
	if font == "" {
		font = defaultCoverFontResourceID
	}
	cfg := loadConfig()
	if resolved, err := resolveLocalCachedOrCloudResource(font, cfg); err == nil {
		font = resolved
	} else if parsed.String("font", "") != "" {
		fmt.Printf("Error: resolve font failed: %v\n", err)
		return
	} else {
		font = ""
	}
	absOut, err := absoluteOutputPath(outputPath)
	if err != nil {
		fmt.Printf("Error: bad output path: %v\n", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(absOut), 0755); err != nil {
		fmt.Printf("Error: create output dir failed: %v\n", err)
		return
	}
	if err := renderCoverImage(imagePath, absOut, title, subtitle, font); err != nil {
		fmt.Printf("Error: cover render failed: %v\n", err)
		return
	}
	metaPath := strings.TrimSuffix(absOut, filepath.Ext(absOut)) + ".json"
	meta := map[string]any{"title": title, "subtitle": subtitle, "image_path": imagePath, "font_path": font, "output_path": absOut}
	if data, err := json.MarshalIndent(meta, "", "  "); err == nil {
		_ = os.WriteFile(metaPath, data, 0644)
	}
	recordProjectArtifact("cover", absOut, "cover.render")
	writeSimpleResult(map[string]any{"output_path": absOut, "meta_path": metaPath})
}

func printCoverUsage() {
	fmt.Println("luma-cli cover <subcommand>")
	fmt.Println("  frame <video> [--time 1.0] [--output cover_frame.png]")
	fmt.Println("  render <image> --title <text> [--subtitle <text>] [--font <path_or_resource_id>] [--output title_cover.jpg]")
}

func installedFFmpegPath() (string, error) {
	runtimeInfo, err := clientruntime.CurrentRuntime("ffmpeg")
	if err == nil && runtimeInfo.ExecutablePath != "" {
		return runtimeInfo.ExecutablePath, nil
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("ffmpeg not found. Run: luma-cli runtime install ffmpeg")
}

func resolveLocalCachedOrCloudResource(value string, cfg *config) (string, error) {
	if value == "" {
		return "", fmt.Errorf("empty resource")
	}
	if _, err := os.Stat(value); err == nil {
		return filepath.Abs(value)
	}
	if cached, err := clientruntime.CurrentResource(value); err == nil {
		return cached.Path, nil
	}
	if cfg == nil {
		return "", fmt.Errorf("not a local file or cached resource: %s. Run 'luma-cli auth login <card_key>' to cache cloud resources", value)
	}
	cached, err := clientruntime.CacheResource(cfg.CardKey, value)
	if err != nil {
		return "", fmt.Errorf("not a local file or cloud resource: %s", value)
	}
	return cached.Path, nil
}

func recordProjectArtifact(artifactType, path, step string) {
	proj, _ := project.GetActiveProject()
	if proj == nil {
		return
	}
	_ = proj.AddArtifact(project.Artifact{Type: artifactType, Path: path, Step: step})
}

func renderCoverImage(imagePath, outputPath, title, subtitle, fontPath string) error {
	input, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer input.Close()
	src, _, err := image.Decode(input)
	if err != nil {
		return err
	}
	bounds := src.Bounds()
	canvas := image.NewRGBA(bounds)
	imagedraw.Draw(canvas, bounds, src, bounds.Min, imagedraw.Src)

	fontData, err := os.ReadFile(fontPath)
	if err != nil {
		return fmt.Errorf("read font: %w", err)
	}
	parsedFont, err := opentype.Parse(fontData)
	if err != nil {
		return fmt.Errorf("parse font: %w", err)
	}
	titleFace, err := opentype.NewFace(parsedFont, &opentype.FaceOptions{Size: fitFontSize(parsedFont, title, float64(bounds.Dx()-120), 72), DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return err
	}
	defer titleFace.Close()
	subtitleFace, err := opentype.NewFace(parsedFont, &opentype.FaceOptions{Size: fitFontSize(parsedFont, subtitle, float64(bounds.Dx()-120), 38), DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return err
	}
	defer subtitleFace.Close()

	y := bounds.Min.Y + int(float64(bounds.Dy())*0.58)
	if title != "" {
		y = drawTextBlock(canvas, title, titleFace, 60, y, color.RGBA{255, 255, 255, 255})
	}
	if subtitle != "" {
		drawTextBlock(canvas, subtitle, subtitleFace, 60, y+28, color.RGBA{255, 255, 255, 255})
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	switch strings.ToLower(filepath.Ext(outputPath)) {
	case ".png":
		return png.Encode(outputFile, canvas)
	default:
		return jpeg.Encode(outputFile, canvas, &jpeg.Options{Quality: 92})
	}
}

func fitFontSize(parsedFont *opentype.Font, text string, maxWidth, start float64) float64 {
	if strings.TrimSpace(text) == "" {
		return start
	}
	for size := start; size >= 24; size -= 2 {
		face, err := opentype.NewFace(parsedFont, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
		if err != nil {
			return start
		}
		width := font.MeasureString(face, text).Round()
		_ = face.Close()
		if float64(width) <= maxWidth {
			return size
		}
	}
	return 24
}

func drawTextBlock(dst *image.RGBA, text string, face font.Face, x, baselineY int, fill color.Color) int {
	metrics := face.Metrics()
	height := (metrics.Ascent + metrics.Descent).Round()
	width := font.MeasureString(face, text).Round()
	paddingX := 24
	paddingY := 16
	rect := image.Rect(x-28, baselineY-metrics.Ascent.Round()-paddingY, x+width+paddingX, baselineY+metrics.Descent.Round()+paddingY)
	imagedraw.Draw(dst, rect, &image.Uniform{color.RGBA{0, 0, 0, 150}}, image.Point{}, imagedraw.Over)
	drawer := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(fill),
		Face: face,
		Dot:  fixed.P(x, baselineY),
	}
	drawer.DrawString(text)
	return baselineY + height + paddingY
}

func writeSimpleResult(data map[string]any) {
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: data})
		return
	}
	if p, ok := data["output_path"].(string); ok {
		fmt.Printf("Output: %s\n", p)
	}
}
