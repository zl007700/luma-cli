package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
)

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

func printCoverUsage() {
	fmt.Println("luma-cli cover <subcommand>")
	fmt.Println("  frame <video> [--time 1.0] [--output step6_cover_frame.png]")
	fmt.Println("  render [image] --title <text> [--subtitle <text>] [--font <path_or_resource_id>] [--template <resource_id>] [--output step6_cover.jpg]")
	fmt.Println("  generate <video_or_image> --title <text> [--subtitle <text>] [--count 6] [--output-dir covers]")
}
