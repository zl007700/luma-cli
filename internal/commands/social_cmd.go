package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

func cmdSocial(args []string) {
	if len(args) < 1 {
		printSocialUsage()
		return
	}
	if args[0] == "download" {
		cmdSocialDownload(args[1:])
		return
	}
	cmdSocialDownload(args)
}

func cmdSocialDownload(args []string) {
	parsed := cmdutil.Parse(args)
	if len(parsed.Positionals) < 1 {
		printSocialUsage()
		return
	}

	shareLink := parsed.Pos(0)
	if !strings.Contains(shareLink, "douyin.com") && !strings.Contains(shareLink, "v.douyin.com") {
		writeSocialError("invalid_douyin_link", fmt.Sprintf("input does not look like a Douyin share link: %s", shareLink))
		return
	}

	outputPath := parsed.String("output", "")
	cfg := loadConfig()
	cardKey := ""
	if cfg != nil {
		cardKey = cfg.CardKey
	}

	if !runtimeOpts.JSON {
		fmt.Println("Parsing share link...")
	}
	result, err := atom.DownloadSocialVideo(shareLink, outputPath, cardKey)
	if err != nil {
		writeSocialError("social_download_failed", err.Error())
		return
	}

	data := map[string]any{
		"video_path": result.VideoPath,
		"title":      result.Title,
		"video_id":   result.VideoID,
		"path":       result.Path,
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: data})
		return
	}

	fmt.Printf("Done! Saved to: %s\n", result.VideoPath)
	fmt.Printf("Path: %s\n", result.Path)
	if result.Title != "" {
		fmt.Printf("Title: %s\n", result.Title)
	}

	writeProjectDouyinDownloadResult(result.VideoPath, result.Title)
}

func printSocialUsage() {
	fmt.Println("usage: luma-cli social download <share_link> [--output <path>]")
	fmt.Println("")
	fmt.Println("  <share_link>  Douyin video share link or direct /video/{id} URL")
	fmt.Println("  --output      Output video file path (default: derived from video title)")
}

func writeSocialError(code, message string) {
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: false, Code: code, Error: message})
		return
	}
	fmt.Printf("Error: %s\n", message)
}

func writeProjectDouyinDownloadResult(videoPath, title string) {
	proj := resolveProjectByName("")
	if proj == nil {
		return
	}

	content := map[string]any{
		"success":     true,
		"message":     "video downloaded",
		"video_id":    "",
		"video_path":  videoPath,
		"video_name":  title,
		"meta_path":   "",
		"text":        "",
		"asr_success": false,
	}
	payload, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		fmt.Printf("Warning: marshal download_video.json failed: %v\n", err)
		return
	}

	jsonPath := filepath.Join(proj.Path, "download_video.json")
	if err := os.WriteFile(jsonPath, payload, 0644); err != nil {
		fmt.Printf("Warning: write %s failed: %v\n", jsonPath, err)
	}
}
