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
	"strconv"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/clientruntime"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

func cmdBGM(args []string) {
	if len(args) < 1 || args[0] != "mix" {
		fmt.Println("usage: luma-cli bgm mix <video> [--bgm <file_or_resource_id>] [--output <mp4>] [--voice-volume 1.0] [--bgm-volume 0.25]")
		return
	}
	parsed := cmdutil.Parse(args[1:])
	videoPath := parsed.Pos(0)
	bgmValue := parsed.String("bgm", "")
	if videoPath == "" {
		fmt.Println("usage: luma-cli bgm mix <video> [--bgm <file_or_resource_id>] [--output <mp4>]")
		return
	}
	cfg := loadConfig()
	defaults := loadClientDefaults(cfg)
	if bgmValue == "" {
		bgmValue = defaults.BGM.Default
	}
	if bgmValue == "" {
		fmt.Println("Error: no BGM specified and no default BGM configured")
		return
	}
	outputPath := parsed.String("output", "step6_bgm.mp4")
	voiceVolume := parsed.String("voice-volume", formatVolume(defaults.BGM.VoiceVolume, "1.0"))
	bgmVolume := parsed.String("bgm-volume", formatVolume(defaults.BGM.BGMVolume, "0.25"))
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
	outputPath := parsed.String("output", "step6_cover_frame.png")
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
	cfg := loadConfig()
	defaults := loadClientDefaults(cfg)
	imagePath := parsed.String("image", "")
	if template := parsed.String("template", ""); template != "" {
		imagePath = template
	}
	if imagePath == "" {
		imagePath = parsed.Pos(0)
	}
	if imagePath == "" {
		imagePath = defaults.Cover.Template
	}
	outputPath := parsed.String("output", "step6_cover.jpg")
	title := parsed.String("title", "")
	subtitle := parsed.String("subtitle", "")
	font := parsed.String("font", "")
	titleFont := parsed.String("title-font", "")
	subtitleFont := parsed.String("subtitle-font", "")
	if font == "" {
		font = defaults.Cover.Font
	}
	if titleFont == "" {
		titleFont = font
	}
	if subtitleFont == "" {
		subtitleFont = font
	}
	if imagePath == "" {
		fmt.Println("usage: luma-cli cover render <image> --title <text> [--subtitle <text>] [--output title_cover.jpg]")
		return
	}
	if resolved, err := resolveLocalCachedOrCloudResource(imagePath, cfg); err == nil {
		imagePath = resolved
	} else if parsed.String("image", "") != "" || parsed.Pos(0) != "" {
		fmt.Printf("Error: resolve cover image failed: %v\n", err)
		return
	} else {
		fmt.Printf("Error: resolve default cover template failed: %v\n", err)
		return
	}
	if resolved, err := resolveLocalCachedOrCloudResource(titleFont, cfg); err == nil {
		titleFont = resolved
	} else if parsed.String("font", "") != "" || parsed.String("title-font", "") != "" {
		fmt.Printf("Error: resolve title font failed: %v\n", err)
		return
	}
	if resolved, err := resolveLocalCachedOrCloudResource(subtitleFont, cfg); err == nil {
		subtitleFont = resolved
	} else if parsed.String("font", "") != "" || parsed.String("subtitle-font", "") != "" {
		fmt.Printf("Error: resolve subtitle font failed: %v\n", err)
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
	titleSize, err := parsed.Float("title-size", float64(defaults.Cover.TitleSize))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	subtitleSize, err := parsed.Float("subtitle-size", float64(defaults.Cover.SubtitleSize))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	topMargin, err := parsed.Float("top-margin", -1)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	verticalOffset, err := parsed.Float("vertical-offset", 0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	opts := coverRenderOptions{
		Title:          title,
		Subtitle:       subtitle,
		TitleFont:      titleFont,
		SubtitleFont:   subtitleFont,
		TitleSize:      titleSize,
		SubtitleSize:   subtitleSize,
		TitleColor:     parsed.String("title-color", "#FFFFFF"),
		SubtitleColor:  parsed.String("subtitle-color", "#FFFFFF"),
		Align:          parsed.String("align", "left"),
		TopMargin:      topMargin,
		VerticalOffset: verticalOffset,
	}
	if err := renderCoverImageWithOptions(imagePath, absOut, opts); err != nil {
		fmt.Printf("Error: cover render failed: %v\n", err)
		return
	}
	metaPath := strings.TrimSuffix(absOut, filepath.Ext(absOut)) + ".json"
	meta := map[string]any{"title": title, "subtitle": subtitle, "image_path": imagePath, "title_font_path": titleFont, "subtitle_font_path": subtitleFont, "output_path": absOut}
	if data, err := json.MarshalIndent(meta, "", "  "); err == nil {
		_ = os.WriteFile(metaPath, data, 0644)
	}
	recordProjectArtifact("cover", absOut, "cover.render")
	writeSimpleResult(map[string]any{"output_path": absOut, "meta_path": metaPath})
}

func printCoverUsage() {
	fmt.Println("luma-cli cover <subcommand>")
	fmt.Println("  frame <video> [--time 1.0] [--output step6_cover_frame.png]")
	fmt.Println("  render [image] --title <text> [--subtitle <text>] [--font <path_or_resource_id>] [--template <resource_id>] [--output step6_cover.jpg]")
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

type coverRenderOptions struct {
	Title          string
	Subtitle       string
	TitleFont      string
	SubtitleFont   string
	TitleSize      float64
	SubtitleSize   float64
	TitleColor     string
	SubtitleColor  string
	Align          string
	TopMargin      float64
	VerticalOffset float64
}

func renderCoverImage(imagePath, outputPath, title, subtitle, fontPath string) error {
	return renderCoverImageWithOptions(imagePath, outputPath, coverRenderOptions{
		Title: title, Subtitle: subtitle, TitleFont: fontPath, SubtitleFont: fontPath,
		TitleSize: 72, SubtitleSize: 38, TitleColor: "#FFFFFF", SubtitleColor: "#FFFFFF", Align: "left", TopMargin: -1,
	})
}

func renderCoverImageWithOptions(imagePath, outputPath string, opts coverRenderOptions) error {
	src, err := loadCoverCanvas(imagePath)
	if err != nil {
		return err
	}
	bounds := src.Bounds()
	canvas := image.NewRGBA(bounds)
	imagedraw.Draw(canvas, bounds, src, bounds.Min, imagedraw.Src)

	titleFontData, err := os.ReadFile(opts.TitleFont)
	if err != nil {
		return fmt.Errorf("read title font: %w", err)
	}
	titleParsedFont, err := opentype.Parse(titleFontData)
	if err != nil {
		return fmt.Errorf("parse title font: %w", err)
	}
	subtitleParsedFont := titleParsedFont
	if opts.SubtitleFont != "" && opts.SubtitleFont != opts.TitleFont {
		subtitleFontData, err := os.ReadFile(opts.SubtitleFont)
		if err != nil {
			return fmt.Errorf("read subtitle font: %w", err)
		}
		subtitleParsedFont, err = opentype.Parse(subtitleFontData)
		if err != nil {
			return fmt.Errorf("parse subtitle font: %w", err)
		}
	}
	if opts.TitleSize <= 0 {
		opts.TitleSize = fitFontSize(titleParsedFont, opts.Title, float64(bounds.Dx()-120), 72)
	}
	if opts.SubtitleSize <= 0 {
		opts.SubtitleSize = fitFontSize(subtitleParsedFont, opts.Subtitle, float64(bounds.Dx()-120), 38)
	}
	titleFace, err := opentype.NewFace(titleParsedFont, &opentype.FaceOptions{Size: opts.TitleSize, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return err
	}
	defer titleFace.Close()
	subtitleFace, err := opentype.NewFace(subtitleParsedFont, &opentype.FaceOptions{Size: opts.SubtitleSize, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return err
	}
	defer subtitleFace.Close()

	y := bounds.Min.Y + int(float64(bounds.Dy())*0.58)
	if opts.TopMargin >= 0 {
		y = bounds.Min.Y + int(opts.TopMargin)
	}
	y += int(opts.VerticalOffset)
	if opts.Title != "" {
		y = drawTextBlockAligned(canvas, opts.Title, titleFace, opts.Align, y, parseHexColor(opts.TitleColor, color.RGBA{255, 255, 255, 255}))
	}
	if opts.Subtitle != "" {
		drawTextBlockAligned(canvas, opts.Subtitle, subtitleFace, opts.Align, y+28, parseHexColor(opts.SubtitleColor, color.RGBA{255, 255, 255, 255}))
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

func loadCoverCanvas(path string) (image.Image, error) {
	if strings.EqualFold(filepath.Ext(path), ".json") {
		return coverCanvasFromTemplate(path)
	}
	input, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer input.Close()
	src, _, err := image.Decode(input)
	return src, err
}

func coverCanvasFromTemplate(path string) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	preview, _ := payload["preview"].(map[string]any)
	from := parseHexColor(strAny(preview["bg_from"]), color.RGBA{28, 22, 46, 255})
	to := parseHexColor(strAny(preview["bg_to"]), color.RGBA{8, 11, 24, 255})
	img := image.NewRGBA(image.Rect(0, 0, 1080, 1920))
	for y := 0; y < 1920; y++ {
		t := float64(y) / 1919.0
		c := lerpColor(from, to, t)
		for x := 0; x < 1080; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img, nil
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

func drawTextBlockAligned(dst *image.RGBA, text string, face font.Face, align string, baselineY int, fill color.Color) int {
	width := font.MeasureString(face, text).Round()
	margin := 60
	x := margin
	switch strings.ToLower(strings.TrimSpace(align)) {
	case "center":
		x = (dst.Bounds().Dx() - width) / 2
	case "right":
		x = dst.Bounds().Dx() - width - margin
	}
	return drawTextBlock(dst, text, face, x, baselineY, fill)
}

func parseHexColor(value string, fallback color.RGBA) color.RGBA {
	raw := strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(raw) != 6 {
		return fallback
	}
	r, err := strconv.ParseUint(raw[0:2], 16, 8)
	if err != nil {
		return fallback
	}
	g, err := strconv.ParseUint(raw[2:4], 16, 8)
	if err != nil {
		return fallback
	}
	b, err := strconv.ParseUint(raw[4:6], 16, 8)
	if err != nil {
		return fallback
	}
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
		A: 255,
	}
}

func writeSimpleResult(data map[string]any) {
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: data})
		return
	}
	if p, ok := data["output_path"].(string); ok {
		fmt.Printf("Output: %s\n", p)
	}
	for _, key := range []string{"count", "scene_count", "matched_count", "mode", "task_id", "csv_path"} {
		if value, ok := data[key]; ok && strAny(value) != "" {
			fmt.Printf("%s: %v\n", key, value)
		}
	}
}
