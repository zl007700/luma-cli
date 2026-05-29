package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

func cmdImage(args []string) error {
	if len(args) < 1 || args[0] != "generate" {
		printImageUsage()
		return nil
	}
	return cmdImageGenerate(args[1:])
}

func cmdVideo(args []string) error {
	if len(args) < 1 || args[0] != "generate" {
		printVideoUsage()
		return nil
	}
	return cmdVideoGenerate(args[1:])
}

func cmdImageGenerate(raw []string) error {
	parsed := cmdutil.Parse(raw)
	prompt := strings.TrimSpace(parsed.String("prompt", ""))
	if prompt == "" {
		prompt = strings.TrimSpace(parsed.Pos(0))
	}
	if prompt == "" {
		printImageUsage()
		return nil
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	count, err := parsed.Int("count", 1)
	if err != nil {
		return output.ErrValidation("%v\n", err)
	}
	if count <= 0 {
		count = 1
	}
	if count > 4 {
		count = 4
	}
	timeoutSec, err := parsed.Int("timeout", 900)
	if err != nil {
		return output.ErrValidation("%v\n", err)
	}
	outputDir := parsed.String("output-dir", "generated_images")
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return output.ErrSystem("bad output dir: %v\n", err)
	}
	if err := os.MkdirAll(absOutputDir, 0755); err != nil {
		return output.ErrSystem("create output dir failed: %v\n", err)
	}

	if !runtimeOpts.JSON {
		fmt.Println("Submitting image generation task...")
	}
	input := map[string]any{
		"prompt":       prompt,
		"count":        count,
		"aspect_ratio": parsed.String("aspect-ratio", "9:16"),
	}
	status, taskID, err := submitAndWaitMediaTask("image_generation", "image_generation_output", input, cfg.CardKey, timeoutSec)
	if err != nil {
		return err
	}
	manifestPath, downloaded, err := downloadGeneratedArtifacts(status, absOutputDir, "image", ".png")
	if err != nil {
		return output.ErrSystem("%v\n", err)
	}
	recordProjectArtifact("image_generation", absOutputDir, "image.generate")
	writeSimpleResult(map[string]any{
		"task_id":       taskID,
		"output_dir":    absOutputDir,
		"manifest_path": manifestPath,
		"downloaded":    downloaded,
		"count":         len(downloaded),
	})
	return nil
}

func cmdVideoGenerate(raw []string) error {
	parsed := cmdutil.Parse(raw)
	prompt := strings.TrimSpace(parsed.String("prompt", ""))
	if prompt == "" {
		prompt = strings.TrimSpace(parsed.Pos(0))
	}
	imagePath := strings.TrimSpace(parsed.String("image", ""))
	imageKey := strings.TrimSpace(parsed.String("image-key", ""))
	if prompt == "" || (imagePath == "" && imageKey == "") {
		printVideoUsage()
		return nil
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	if imageKey == "" {
		if _, err := os.Stat(imagePath); err != nil {
			return output.ErrValidation("image file not found: %s\n", imagePath)
		}
		if !runtimeOpts.JSON {
			fmt.Println("Uploading first frame...")
		}
		uploaded, err := cloud.UploadFile(imagePath, cfg.CardKey, "image_generation_input")
		if err != nil {
			return output.ErrNetwork("upload image failed: %v\n", err)
		}
		imageKey = atom.NormalizeResourceKey(uploaded, cfg.CardKey)
	} else {
		imageKey = atom.NormalizeResourceKey(imageKey, cfg.CardKey)
	}
	duration, err := parsed.Int("duration", 4)
	if err != nil {
		return output.ErrValidation("%v\n", err)
	}
	if duration <= 0 {
		duration = 4
	}
	timeoutSec, err := parsed.Int("timeout", 1200)
	if err != nil {
		return output.ErrValidation("%v\n", err)
	}
	outputPath := parsed.String("output", "generated_video.mp4")
	absOutput, err := ensureOutputDir(outputPath)
	if err != nil {
		return output.ErrSystem("%v\n", err)
	}

	if !runtimeOpts.JSON {
		fmt.Println("Submitting video generation task...")
	}
	input := map[string]any{
		"prompt":           prompt,
		"image_object_key": imageKey,
		"duration_seconds": duration,
		"aspect_ratio":     parsed.String("aspect-ratio", "9:16"),
	}
	status, taskID, err := submitAndWaitMediaTask("video_generation", "video_generation_output", input, cfg.CardKey, timeoutSec)
	if err != nil {
		return err
	}
	manifestPath, downloaded, err := downloadGeneratedArtifacts(status, filepath.Dir(absOutput), "video", ".mp4")
	if err != nil {
		return output.ErrSystem("%v\n", err)
	}
	if len(downloaded) > 0 && downloaded[0] != absOutput {
		_ = os.Remove(absOutput)
		if err := os.Rename(downloaded[0], absOutput); err != nil {
			return output.ErrSystem("move generated video failed: %v\n", err)
		}
		downloaded[0] = absOutput
	}
	recordProjectArtifact("video_generation", absOutput, "video.generate")
	writeSimpleResult(map[string]any{
		"task_id":       taskID,
		"output_path":   absOutput,
		"manifest_path": manifestPath,
		"downloaded":    downloaded,
	})
	return nil
}

func submitAndWaitMediaTask(taskType, taskName string, input map[string]any, cardKey string, timeoutSec int) (map[string]any, string, error) {
	taskResult, err := cloud.SubmitTask(taskType, taskName, input, cardKey)
	if err != nil {
		return nil, "", output.ErrNetwork("submit %s task failed: %v\n", taskType, err)
	}
	taskID, _ := taskResult["task_id"].(string)
	if taskID == "" {
		return nil, "", output.ErrNetwork("no task_id returned")
	}
	if !runtimeOpts.JSON {
		fmt.Printf("  Task ID: %s\n", taskID)
	}
	status, stillRunning := cloud.WaitTaskComplete(taskID, cardKey, timeoutSec)
	if stillRunning {
		return status, taskID, output.ErrNetwork("%s task timed out", taskType)
	}
	if msg := atom.TaskFailure(status); msg != "" {
		return status, taskID, output.ErrNetwork("%s task failed: %s\n", taskType, msg)
	}
	if statusText := strings.ToLower(fmt.Sprint(status["status"])); statusText != "" && statusText != "completed" {
		return status, taskID, output.ErrNetwork("%s task failed: %v\n", taskType, status)
	}
	return status, taskID, nil
}

func downloadGeneratedArtifacts(status map[string]any, absOutputDir, prefix, ext string) (string, []string, error) {
	outputPayload, _ := status["output"].(map[string]any)
	result, _ := outputPayload["result"].(map[string]any)
	manifestPath := filepath.Join(absOutputDir, prefix+"_manifest.json")
	if len(result) > 0 {
		data, _ := json.MarshalIndent(result, "", "  ")
		_ = os.WriteFile(manifestPath, data, 0644)
	} else if manifestURL := atom.ResultURL(status); manifestURL != "" {
		if err := atom.DownloadFile(manifestURL, manifestPath); err != nil {
			return manifestPath, nil, fmt.Errorf("download manifest failed: %w", err)
		}
	}
	artifacts, _ := result["artifacts"].([]any)
	downloaded := []string{}
	for idx, item := range artifacts {
		artifact, _ := item.(map[string]any)
		url, _ := artifact["download_url"].(string)
		if strings.TrimSpace(url) == "" {
			continue
		}
		target := filepath.Join(absOutputDir, fmt.Sprintf("%s_%02d%s", prefix, idx+1, ext))
		if err := atom.DownloadFile(url, target); err != nil {
			fmt.Printf("  Warning: download artifact %d failed: %v\n", idx+1, err)
			continue
		}
		downloaded = append(downloaded, target)
	}
	return manifestPath, downloaded, nil
}

func printImageUsage() {
	fmt.Println("luma-cli image generate <prompt> [--count 1] [--aspect-ratio 9:16] [--output-dir generated_images]")
}

func printVideoUsage() {
	fmt.Println("luma-cli video generate <prompt> --image <first_frame.png> [--duration 4] [--aspect-ratio 9:16] [--output generated_video.mp4]")
}
