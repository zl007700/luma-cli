package commands

import (
	"github.com/luma-cli/lumer-cli/internal/output"
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

func cmdCover(args []string) error {
	if len(args) < 1 {
		printCoverUsage()
		return nil
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
	return nil
}

func cmdCoverFrame(raw []string) error {
	parsed := cmdutil.Parse(raw)
	videoPath := parsed.Pos(0)
	if videoPath == "" {
		fmt.Println("usage: luma-cli cover frame <video> [--time 1.0] [--output cover_frame.png]")
		return nil
	}
	outputPath := parsed.String("output", "step6_cover_frame.png")
	seek := parsed.String("time", "1.0")
	absOut, err := ensureOutputDir(outputPath)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}
	ffmpeg, err := installedFFmpegPath()
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}
	cmd := exec.Command(ffmpeg, "-y", "-ss", seek, "-i", videoPath, "-frames:v", "1", absOut)
	if data, err := cmd.CombinedOutput(); err != nil {
		return output.ErrSystem(fmt.Sprintf("ffmpeg frame extract failed: %v\n%s\n", err, string(data)))
	}
	// Rename output to include content hash for traceability.
	if hashed, err := hashSuffixFile(absOut); err == nil {
		absOut = hashed
	}

	recordProjectArtifact("cover_frame", absOut, "cover.frame")
	writeSimpleResult(map[string]any{"output_path": absOut})
	return nil
}

func cmdCoverGenerate(raw []string) error {
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
		return nil
	}
	if _, err := os.Stat(sourcePath); err != nil {
		return output.ErrValidation(fmt.Sprintf("source file not found: %s\n", sourcePath))
	}
	cfg := loadConfig()
	if cfg == nil {
		return output.ErrAuth("not logged in. Run: luma-cli auth login <card_key>")
	}
	defaults := loadClientDefaults(cfg)
	count, err := parsed.Int("count", 6)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}
	if count <= 0 {
		count = 6
	}
	if count > 20 {
		count = 20
	}
	frameSecond, err := parsed.Float("frame-second", 1.0)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}
	timeoutSec, err := parsed.Int("timeout", 600)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}
	outputDir := parsed.String("output-dir", "covers")
	absOutputDir, err := ensureOutputDir(outputDir)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}

	fmt.Println("Uploading cover source...")
	sourceKey, err := cloud.UploadFile(sourcePath, cfg.CardKey, "cover_input")
	if err != nil {
		return output.ErrNetwork(fmt.Sprintf("upload source failed: %v\n", err))
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
	if ref := parsed.String("title-font", fontRef); ref != "" {
		if parsed.Has("font") || parsed.Has("title-font") {
			objKey, resID := resolveCloudResourceRef("font", ref, cfg.CardKey)
			if objKey != "" {
				input["title_font_object_key"] = objKey
			} else {
				input["title_font_resource_id"] = resID
			}
		} else {
			input["title_font_resource_id"] = ref
		}
	}
	if ref := parsed.String("subtitle-font", fontRef); ref != "" {
		if parsed.Has("font") || parsed.Has("subtitle-font") {
			objKey, resID := resolveCloudResourceRef("font", ref, cfg.CardKey)
			if objKey != "" {
				input["subtitle_font_object_key"] = objKey
			} else {
				input["subtitle_font_resource_id"] = resID
			}
		} else {
			input["subtitle_font_resource_id"] = ref
		}
	}
	if parsed.Has("template") {
		if ref := parsed.String("template", ""); ref != "" {
			objKey, resID := resolveCloudResourceRef("cover_templates", ref, cfg.CardKey)
			if objKey != "" {
				input["template_object_keys"] = []string{objKey}
			} else {
				input["template_resource_ids"] = []string{resID}
			}
		}
	} else {
		if ids := defaultCoverTemplateResourceIDs(cfg, defaults, count); len(ids) > 0 {
			input["template_resource_ids"] = ids
		} else if ref := strings.TrimSpace(defaults.Cover.Template); ref != "" {
			input["template_resource_ids"] = []string{ref}
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
		return output.ErrNetwork(fmt.Sprintf("submit cover task failed: %v\n", err))
	}
	taskID, _ := taskResult["task_id"].(string)
	if taskID == "" {
		return output.ErrNetwork("no task_id returned")
	}
	fmt.Printf("  Task ID: %s\n", taskID)

	status, stillRunning := cloud.WaitTaskComplete(taskID, cfg.CardKey, timeoutSec)
	if stillRunning {
		return output.ErrNetwork("cover task timed out")
	}
	if msg := atom.TaskFailure(status); msg != "" {
		return output.ErrNetwork(fmt.Sprintf("cover task failed: %s\n", msg))
	}
	if statusText := strings.ToLower(fmt.Sprint(status["status"])); statusText != "" && statusText != "completed" {
		return output.ErrNetwork(fmt.Sprintf("cover task failed: %v\n", status))
	}

	manifestPath, downloaded, err := downloadCoverOutputs(status, absOutputDir)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}
	recordProjectArtifact("cover", absOutputDir, "cover.generate")
	writeSimpleResult(map[string]any{
		"task_id":       taskID,
		"output_dir":    absOutputDir,
		"manifest_path": manifestPath,
		"downloaded":    downloaded,
	})
	if runtimeOpts.JSON {
		return nil
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
	return nil
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

// resolveCloudResourceRef resolves a resource reference that may be an
// asset key, object key, or resource ID. Returns the object_key and
// resource_id fields to set in a task input payload.
func resolveCloudResourceRef(group, ref, cardKey string) (objKey, resourceID string) {
	if key, err := atom.ResolveAssetKey(group, ref, cardKey); err == nil {
		return key, ""
	}
	if isObjectKeyRef(ref) {
		return atom.NormalizeResourceKey(ref, cardKey), ""
	}
	return "", ref
}

func downloadCoverOutputs(status map[string]any, absOutputDir string) (manifestPath string, downloaded []string, _ error) {
	output, _ := status["output"].(map[string]any)
	result, _ := output["result"].(map[string]any)
	manifestPath = filepath.Join(absOutputDir, "cover_manifest.json")
	if len(result) > 0 {
		data, _ := json.MarshalIndent(result, "", "  ")
		_ = os.WriteFile(manifestPath, data, 0644)
	} else if manifestURL := atom.ResultURL(status); manifestURL != "" {
		if err := atom.DownloadFile(manifestURL, manifestPath); err != nil {
			return manifestPath, nil, fmt.Errorf("download manifest failed: %w", err)
		}
	}
	covers, _ := result["covers"].([]any)
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
	return manifestPath, downloaded, nil
}

func printCoverUsage() {
	fmt.Println("luma-cli cover <subcommand>")
	fmt.Println("  frame <video> [--time 1.0] [--output step6_cover_frame.png]")
	fmt.Println("  render [image] --title <text> [--subtitle <text>] [--font <path_or_resource_id>] [--template <resource_id>] [--output step6_cover.jpg]")
	fmt.Println("  generate <video_or_image> --title <text> [--subtitle <text>] [--count 6] [--output-dir covers]")
}
