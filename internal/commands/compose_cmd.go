package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/clientruntime"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
	"github.com/luma-cli/lumer-cli/project"
)

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
	bgmPath, err := resolveLocalOrCachedResource(bgmValue)
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
	if font != "" {
		if resolved, err := resolveLocalOrCachedResource(font); err == nil {
			font = resolved
		}
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
	filter := buildCoverFilter(title, subtitle, font)
	args := []string{"-y", "-i", imagePath}
	if filter != "" {
		args = append(args, "-vf", filter)
	}
	args = append(args, "-frames:v", "1", "-q:v", "2", absOut)
	cmd := exec.Command(ffmpeg, args...)
	if data, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Error: ffmpeg cover render failed: %v\n%s\n", err, string(data))
		return
	}
	metaPath := strings.TrimSuffix(absOut, filepath.Ext(absOut)) + ".json"
	meta := map[string]any{"title": title, "subtitle": subtitle, "image_path": imagePath, "output_path": absOut}
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

func resolveLocalOrCachedResource(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("empty resource")
	}
	if _, err := os.Stat(value); err == nil {
		return filepath.Abs(value)
	}
	cached, err := clientruntime.CurrentResource(value)
	if err != nil {
		return "", fmt.Errorf("not a local file or cached resource: %s", value)
	}
	return cached.Path, nil
}

func buildCoverFilter(title, subtitle, font string) string {
	var filters []string
	fontOpt := ""
	if font != "" {
		fontOpt = ":fontfile='" + escapeDrawtext(font) + "'"
	}
	if title != "" {
		filters = append(filters, fmt.Sprintf("drawtext=text='%s'%s:fontcolor=white:fontsize=72:box=1:boxcolor=black@0.55:boxborderw=24:x=60:y=h*0.58", escapeDrawtext(title), fontOpt))
	}
	if subtitle != "" {
		filters = append(filters, fmt.Sprintf("drawtext=text='%s'%s:fontcolor=white:fontsize=38:box=1:boxcolor=black@0.45:boxborderw=16:x=60:y=h*0.58+110", escapeDrawtext(subtitle), fontOpt))
	}
	return strings.Join(filters, ",")
}

func escapeDrawtext(value string) string {
	value = strings.ReplaceAll(value, `\`, `/`)
	value = strings.ReplaceAll(value, `'`, `\'`)
	value = strings.ReplaceAll(value, `:`, `\:`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return value
}

func recordProjectArtifact(artifactType, path, step string) {
	proj, _ := project.GetActiveProject()
	if proj == nil {
		return
	}
	_ = proj.AddArtifact(project.Artifact{Type: artifactType, Path: path, Step: step})
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
