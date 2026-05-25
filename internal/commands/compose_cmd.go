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

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/atom"
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
	// Rename output to include content hash for traceability.
	if hashed, err := hashSuffixFile(absOut); err == nil {
		absOut = hashed
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
	case "generate":
		cmdCoverGenerate(args[1:])
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
	// Rename output to include content hash for traceability.
	if hashed, err := hashSuffixFile(absOut); err == nil {
		absOut = hashed
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
	// Rename output to include content hash for traceability.
	if hashed, err := hashSuffixFile(absOut); err == nil {
		absOut = hashed
	}

	metaPath := strings.TrimSuffix(absOut, filepath.Ext(absOut)) + ".json"
	meta := map[string]any{"title": title, "subtitle": subtitle, "image_path": imagePath, "title_font_path": titleFont, "subtitle_font_path": subtitleFont, "output_path": absOut}
	if data, err := json.MarshalIndent(meta, "", "  "); err == nil {
		_ = os.WriteFile(metaPath, data, 0644)
	}
	recordProjectArtifact("cover", absOut, "cover.render")
	writeSimpleResult(map[string]any{"output_path": absOut, "meta_path": metaPath})
}

func cmdCoverGenerate(raw []string) {
	parsed := cmdutil.Parse(raw)
	sourcePath := parsed.String("video", "")
	if sourcePath == "" {
		sourcePath = parsed.String("image", "")
	}
	if sourcePath == "" {
		sourcePath = parsed.String("frame", "")
	}
	if sourcePath == "" {
		sourcePath = parsed.Pos(0)
	}
	title := strings.TrimSpace(parsed.String("title", ""))
	subtitle := strings.TrimSpace(parsed.String("subtitle", ""))
	if sourcePath == "" || title == "" {
		fmt.Println("usage: luma-cli cover generate <video_or_image> --title <text> [--subtitle <text>] [--count 6] [--output-dir covers]")
		return
	}
	if _, err := os.Stat(sourcePath); err != nil {
		fmt.Printf("Error: source file not found: %s\n", sourcePath)
		return
	}
	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return
	}
	defaults := loadClientDefaults(cfg)
	count, err := parsed.Int("count", 6)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if count <= 0 {
		count = 6
	}
	if count > 20 {
		count = 20
	}
	frameSecond, err := parsed.Float("frame-second", 1.0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	timeoutSec, err := parsed.Int("timeout", 600)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	outputDir := parsed.String("output-dir", "covers")
	absOutputDir, err := absoluteOutputPath(outputDir)
	if err != nil {
		fmt.Printf("Error: bad output dir: %v\n", err)
		return
	}
	if err := os.MkdirAll(absOutputDir, 0755); err != nil {
		fmt.Printf("Error: create output dir failed: %v\n", err)
		return
	}

	fmt.Println("Uploading cover source...")
	sourceKey, err := cloud.UploadFile(sourcePath, cfg.CardKey, "cover_input")
	if err != nil {
		fmt.Printf("Error: upload source failed: %v\n", err)
		return
	}
	sourceKey = atom.NormalizeResourceKey(sourceKey, cfg.CardKey)

	input := map[string]any{
		"mode":         parsed.String("mode", "template"),
		"title":        title,
		"subtitle":     subtitle,
		"count":        count,
		"frame_second": frameSecond,
	}
	ext := strings.ToLower(filepath.Ext(sourcePath))
	if parsed.String("frame", "") != "" || ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
		if parsed.String("video", "") != "" {
			input["source_video_object_key"] = sourceKey
		} else {
			input["source_image_object_key"] = sourceKey
		}
	} else {
		input["source_video_object_key"] = sourceKey
	}

	fontRef := parsed.String("font", strings.TrimSpace(defaults.Cover.Font))
	titleFontRef := parsed.String("title-font", fontRef)
	subtitleFontRef := parsed.String("subtitle-font", fontRef)
	if titleFontRef != "" {
		if parsed.Has("font") || parsed.Has("title-font") {
			if key, err := atom.ResolveAssetKey("font", titleFontRef, cfg.CardKey); err == nil {
				input["title_font_object_key"] = key
			} else if isObjectKeyRef(titleFontRef) {
				input["title_font_object_key"] = atom.NormalizeResourceKey(titleFontRef, cfg.CardKey)
			} else {
				input["title_font_resource_id"] = titleFontRef
			}
		} else {
			input["title_font_resource_id"] = titleFontRef
		}
	}
	if subtitleFontRef != "" {
		if parsed.Has("font") || parsed.Has("subtitle-font") {
			if key, err := atom.ResolveAssetKey("font", subtitleFontRef, cfg.CardKey); err == nil {
				input["subtitle_font_object_key"] = key
			} else if isObjectKeyRef(subtitleFontRef) {
				input["subtitle_font_object_key"] = atom.NormalizeResourceKey(subtitleFontRef, cfg.CardKey)
			} else {
				input["subtitle_font_resource_id"] = subtitleFontRef
			}
		} else {
			input["subtitle_font_resource_id"] = subtitleFontRef
		}
	}
	if parsed.Has("template") {
		templateRef := parsed.String("template", "")
		if templateRef != "" {
			if key, err := atom.ResolveAssetKey("cover_templates", templateRef, cfg.CardKey); err == nil {
				input["template_object_keys"] = []string{key}
			} else if isObjectKeyRef(templateRef) {
				input["template_object_keys"] = []string{atom.NormalizeResourceKey(templateRef, cfg.CardKey)}
			} else {
				input["template_resource_ids"] = []string{templateRef}
			}
		}
	} else {
		templateIDs := defaultCoverTemplateResourceIDs(cfg, defaults, count)
		if len(templateIDs) > 0 {
			input["template_resource_ids"] = templateIDs
		} else {
			templateRef := strings.TrimSpace(defaults.Cover.Template)
			if templateRef != "" {
				input["template_resource_ids"] = []string{templateRef}
			}
		}
	}

	fmt.Println("Submitting cover task...")
	fmt.Printf("  Source: %s\n", sourceKey)
	fmt.Printf("  Count: %d\n", count)
	if input["title_font_resource_id"] != nil {
		fmt.Printf("  Font resource: %s\n", input["title_font_resource_id"])
	}
	if input["template_resource_ids"] != nil {
		fmt.Printf("  Template resources: %v\n", input["template_resource_ids"])
	}
	taskResult, err := cloud.SubmitTask("cover", "cover_output", input, cfg.CardKey)
	if err != nil {
		fmt.Printf("Error: submit cover task failed: %v\n", err)
		return
	}
	taskID, _ := taskResult["task_id"].(string)
	if taskID == "" {
		fmt.Println("Error: no task_id returned")
		return
	}
	fmt.Printf("  Task ID: %s\n", taskID)

	status, stillRunning := cloud.WaitTaskComplete(taskID, cfg.CardKey, timeoutSec)
	if stillRunning {
		fmt.Println("Error: cover task timed out")
		return
	}
	if msg := atom.TaskFailure(status); msg != "" {
		fmt.Printf("Error: cover task failed: %s\n", msg)
		return
	}
	if statusText := strings.ToLower(fmt.Sprint(status["status"])); statusText != "" && statusText != "completed" {
		fmt.Printf("Error: cover task failed: %v\n", status)
		return
	}

	output, _ := status["output"].(map[string]any)
	result, _ := output["result"].(map[string]any)
	manifestPath := filepath.Join(absOutputDir, "cover_manifest.json")
	if len(result) > 0 {
		data, _ := json.MarshalIndent(result, "", "  ")
		_ = os.WriteFile(manifestPath, data, 0644)
	} else if manifestURL := atom.ResultURL(status); manifestURL != "" {
		if err := atom.DownloadFile(manifestURL, manifestPath); err != nil {
			fmt.Printf("Error: download manifest failed: %v\n", err)
			return
		}
	}
	covers, _ := result["covers"].([]any)
	downloaded := []string{}
	for idx, item := range covers {
		cover, _ := item.(map[string]any)
		url, _ := cover["image_url"].(string)
		if strings.TrimSpace(url) == "" {
			continue
		}
		target := filepath.Join(absOutputDir, fmt.Sprintf("cover_%02d.jpg", idx+1))
		if err := atom.DownloadFile(url, target); err != nil {
			fmt.Printf("  Warning: download cover %d failed: %v\n", idx+1, err)
			continue
		}
		downloaded = append(downloaded, target)
	}
	recordProjectArtifact("cover", absOutputDir, "cover.generate")
	writeSimpleResult(map[string]any{
		"task_id":       taskID,
		"output_dir":    absOutputDir,
		"manifest_path": manifestPath,
		"downloaded":    downloaded,
	})
	if runtimeOpts.JSON {
		return
	}
	fmt.Println("Done!")
	if _, err := os.Stat(manifestPath); err == nil {
		fmt.Printf("  Manifest: %s\n", manifestPath)
	}
	for _, path := range downloaded {
		fmt.Printf("  Cover: %s\n", path)
	}
	if len(downloaded) == 0 {
		fmt.Println("  Cover images were generated in cloud, but this backend response did not include signed image_url fields.")
	}
}

func isObjectKeyRef(value string) bool {
	value = strings.TrimSpace(value)
	return strings.Contains(value, "/") || strings.HasPrefix(value, "prod/") || strings.HasPrefix(value, "resource/")
}

func defaultCoverTemplateResourceIDs(cfg *config, defaults *cloud.ClientDefaults, count int) []string {
	if cfg == nil {
		return nil
	}
	limit := count
	if limit <= 0 {
		limit = 6
	}
	items, err := cloud.ListClientResources("cover_template", "", cfg.CardKey)
	if err != nil || len(items) == 0 {
		items, err = cloud.ListClientResources("template", "cover", cfg.CardKey)
		if err != nil || len(items) == 0 {
			return nil
		}
	}
	ids := []string{}
	seen := map[string]bool{}
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if defaults != nil {
		add(defaults.Cover.Template)
	}
	for _, item := range items {
		add(item.ID)
		if len(ids) >= limit {
			break
		}
	}
	return ids
}

func downloadCoverCandidates(status map[string]any, outputDir string) int {
	result, ok := nestedMap(status, "output", "result")
	if !ok {
		return 0
	}
	rawCovers, ok := result["covers"].([]any)
	if !ok {
		return 0
	}
	downloaded := 0
	for i, raw := range rawCovers {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		url := strAny(item["image_url"])
		if url == "" {
			url = strAny(item["url"])
		}
		if url == "" {
			continue
		}
		outPath := filepath.Join(outputDir, fmt.Sprintf("cover_%02d.jpg", i+1))
		if err := atom.DownloadFile(url, outPath); err == nil {
			downloaded++
		}
	}
	return downloaded
}

func nestedMap(root map[string]any, path ...string) (map[string]any, bool) {
	current := root
	for _, key := range path {
		next, ok := current[key].(map[string]any)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func printCoverUsage() {
	fmt.Println("luma-cli cover <subcommand>")
	fmt.Println("  frame <video> [--time 1.0] [--output step6_cover_frame.png]")
	fmt.Println("  render [image] --title <text> [--subtitle <text>] [--font <path_or_resource_id>] [--template <resource_id>] [--output step6_cover.jpg]")
	fmt.Println("  generate <video_or_image> --title <text> [--subtitle <text>] [--count 6] [--output-dir covers]")
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
	for _, key := range []string{"count", "scene_count", "matched_count", "mode", "task_id", "csv_path", "group_path"} {
		if value, ok := data[key]; ok && strAny(value) != "" {
			fmt.Printf("%s: %v\n", key, value)
		}
	}
}
