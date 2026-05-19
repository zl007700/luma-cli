package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/clientruntime"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
)

type pipMediaInfo struct {
	Width    int
	Height   int
	Duration float64
}

type pipRenderSpec struct {
	MaterialPath     string
	MediaType        string
	Start            float64
	End              float64
	Duration         float64
	MaterialDuration float64
	Replace          bool
}

func cmdPIP(args []string) {
	if len(args) < 1 || args[0] != "render" {
		fmt.Println("usage: luma-cli pip render <video> --plan pip_plan.json [--output pip_video.mp4] [--landscape-height-ratio 0.32]")
		return
	}
	parsed := cmdutil.Parse(args[1:])
	videoPath := parsed.Pos(0)
	planPath := parsed.String("plan", "")
	if videoPath == "" || planPath == "" {
		fmt.Println("usage: luma-cli pip render <video> --plan pip_plan.json [--output pip_video.mp4]")
		return
	}
	outputPath := parsed.String("output", "pip_video.mp4")
	landscapeRatio, err := parsed.Float("landscape-height-ratio", 0.32)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	absOut, err := absoluteOutputPath(outputPath)
	if err != nil {
		fmt.Printf("Error: bad output path: %v\n", err)
		return
	}
	if err := renderPIPVideo(videoPath, planPath, absOut, landscapeRatio); err != nil {
		fmt.Printf("Error: pip render failed: %v\n", err)
		return
	}
	recordProjectArtifact("pip", absOut, "pip.render")
	writeSimpleResult(map[string]any{"output_path": absOut, "plan_path": planPath})
}

func renderPIPVideo(sourceVideo, planPath, outputPath string, landscapeRatio float64) error {
	planData, err := os.ReadFile(planPath)
	if err != nil {
		return err
	}
	var plan map[string]any
	if err := json.Unmarshal(planData, &plan); err != nil {
		return err
	}
	ffmpeg, err := installedFFmpegPath()
	if err != nil {
		return err
	}
	ffprobe, err := installedFFprobePath(ffmpeg)
	if err != nil {
		return err
	}
	sourceInfo, err := probeMedia(ffprobe, sourceVideo)
	if err != nil {
		return err
	}
	specs, err := buildPIPRenderSpecs(plan, ffprobe, sourceInfo.Duration)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	if len(specs) == 0 {
		return copyFile(sourceVideo, outputPath)
	}
	cmd := []string{ffmpeg, "-y", "-i", sourceVideo}
	filterParts := []string{"[0:v]setsar=1[v0]"}
	currentLabel := "v0"
	for index, spec := range specs {
		inputIndex := index + 1
		if spec.MediaType == "image" {
			cmd = append(cmd, "-loop", "1", "-i", spec.MaterialPath)
		} else {
			cmd = append(cmd, "-stream_loop", "-1", "-i", spec.MaterialPath)
		}
		prepared := fmt.Sprintf("prep%d", inputIndex)
		filterParts = append(filterParts, buildPIPPrepareFilter(inputIndex, prepared, spec, sourceInfo.Width, sourceInfo.Height, landscapeRatio))
		outputLabel := fmt.Sprintf("v%d", inputIndex)
		if spec.Replace {
			filterParts = append(filterParts, fmt.Sprintf("[%s][%s]overlay=0:0:enable='between(t,%.6f,%.6f)'[%s]", currentLabel, prepared, spec.Start, spec.End, outputLabel))
			currentLabel = outputLabel
			continue
		}
		bgSrc := fmt.Sprintf("bgsrc%d", inputIndex)
		fgSrc := fmt.Sprintf("fgsrc%d", inputIndex)
		blur := fmt.Sprintf("blur%d", inputIndex)
		mid := fmt.Sprintf("mid%d", inputIndex)
		filterParts = append(filterParts, fmt.Sprintf("[%s]split=2[%s][%s]", prepared, bgSrc, fgSrc))
		filterParts = append(filterParts, fmt.Sprintf("[%s]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,gblur=sigma=30,eq=brightness=-0.03:saturation=0.82,setsar=1[%s]", bgSrc, sourceInfo.Width, sourceInfo.Height, sourceInfo.Width, sourceInfo.Height, blur))
		filterParts = append(filterParts, fmt.Sprintf("[%s][%s]overlay=0:0:enable='between(t,%.6f,%.6f)'[%s];[%s][%s]overlay=(W-w)/2:(H-h)/2:enable='between(t,%.6f,%.6f)'[%s]", currentLabel, blur, spec.Start, spec.End, mid, mid, fgSrc, spec.Start, spec.End, outputLabel))
		currentLabel = outputLabel
	}
	cmd = append(cmd,
		"-filter_complex", strings.Join(filterParts, ";"),
		"-map", "["+currentLabel+"]",
		"-map", "0:a?",
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "copy",
		"-movflags", "+faststart",
		outputPath,
	)
	run := exec.Command(cmd[0], cmd[1:]...)
	if data, err := run.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\n%s", err, string(data))
	}
	return nil
}

func buildPIPRenderSpecs(plan map[string]any, ffprobe string, sourceDuration float64) ([]pipRenderSpec, error) {
	scenes := listMap(plan["scene_units"])
	sceneMap := map[string]map[string]any{}
	for _, scene := range scenes {
		if id := strAny(scene["scene_id"]); id != "" {
			sceneMap[id] = scene
		}
	}
	materials := append(listMap(plan["material_candidates"]), listMap(plan["materials"])...)
	materialMap := map[string]map[string]any{}
	for _, material := range materials {
		id := firstString(material, "material_id", "id", "resource_id")
		if id != "" {
			materialMap[id] = material
		}
	}
	cfg := loadConfig()
	var specs []pipRenderSpec
	for _, insert := range listMap(plan["inserts"]) {
		start, end := pipInsertRange(insert, sceneMap, sourceDuration)
		if end <= start {
			continue
		}
		materialID := firstString(insert, "material_id", "id", "resource_id")
		material := materialMap[materialID]
		materialPathValue := firstString(material, "path", "file_path", "local_path", "material_path")
		if materialPathValue == "" {
			materialPathValue = firstString(insert, "path", "file_path", "local_path", "material_path")
		}
		if materialPathValue == "" {
			materialPathValue = materialID
		}
		materialPath, err := resolveLocalCachedOrCloudResource(materialPathValue, cfg)
		if err != nil {
			return nil, err
		}
		info, err := probeMedia(ffprobe, materialPath)
		if err != nil {
			return nil, err
		}
		mediaType := strings.ToLower(firstString(material, "media_type", "type"))
		if mediaType == "" {
			mediaType = inferPIPMediaType(materialPath)
		}
		specs = append(specs, pipRenderSpec{
			MaterialPath:     materialPath,
			MediaType:        mediaType,
			Start:            start,
			End:              end,
			Duration:         end - start,
			MaterialDuration: maxFloat(info.Duration, 0.01),
			Replace:          info.Height >= info.Width,
		})
	}
	return specs, nil
}

func pipInsertRange(insert map[string]any, scenes map[string]map[string]any, sourceDuration float64) (float64, float64) {
	if start, ok := numberAny(insert["start"]); ok {
		end, _ := numberAny(insert["end"])
		return maxFloat(start, 0), minFloat(end, sourceDuration)
	}
	startScene := scenes[firstString(insert, "start_scene_id", "start_sent_id", "scene_id")]
	endSceneID := firstString(insert, "end_scene_id", "end_sent_id")
	if endSceneID == "" {
		endSceneID = firstString(insert, "start_scene_id", "start_sent_id", "scene_id")
	}
	endScene := scenes[endSceneID]
	start, _ := numberAny(startScene["start"])
	end, _ := numberAny(endScene["end"])
	return maxFloat(start, 0), minFloat(end, sourceDuration)
}

func buildPIPPrepareFilter(inputIndex int, outputLabel string, spec pipRenderSpec, sourceWidth, sourceHeight int, landscapeRatio float64) string {
	_ = landscapeRatio
	scale := fmt.Sprintf("scale=%d:-2,setsar=1", sourceWidth)
	if spec.Replace {
		scale = fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,setsar=1", sourceWidth, sourceHeight, sourceWidth, sourceHeight)
	}
	return fmt.Sprintf("[%d:v]setpts=PTS-STARTPTS+%.6f/TB,%s,trim=duration=%.6f[%s]", inputIndex, spec.Start, scale, spec.Duration, outputLabel)
}

func installedFFprobePath(ffmpeg string) (string, error) {
	name := "ffprobe"
	if strings.HasSuffix(strings.ToLower(ffmpeg), ".exe") {
		name = "ffprobe.exe"
	}
	candidate := filepath.Join(filepath.Dir(ffmpeg), name)
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	runtimeInfo, err := clientruntime.CurrentRuntime("ffmpeg")
	if err == nil && runtimeInfo.ExecutablePath != "" {
		candidate = filepath.Join(filepath.Dir(runtimeInfo.ExecutablePath), name)
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("ffprobe not found. Run: luma-cli runtime install ffmpeg")
}

func probeMedia(ffprobe, path string) (pipMediaInfo, error) {
	cmd := exec.Command(ffprobe, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
	data, err := cmd.Output()
	if err != nil {
		return pipMediaInfo{}, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return pipMediaInfo{}, err
	}
	info := pipMediaInfo{}
	if format, ok := payload["format"].(map[string]any); ok {
		info.Duration, _ = numberAny(format["duration"])
	}
	for _, stream := range listMap(payload["streams"]) {
		if strAny(stream["codec_type"]) == "video" {
			info.Width, _ = intAny(stream["width"])
			info.Height, _ = intAny(stream["height"])
			break
		}
	}
	if info.Width <= 0 {
		info.Width = 1080
	}
	if info.Height <= 0 {
		info.Height = 1920
	}
	return info, nil
}

func inferPIPMediaType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".bmp":
		return "image"
	default:
		return "video"
	}
}

func listMap(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strAny(m[key]); value != "" {
			return value
		}
	}
	return ""
}

func strAny(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func numberAny(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		n, err := v.Float64()
		return n, err == nil
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func intAny(value any) (int, bool) {
	n, ok := numberAny(value)
	return int(n), ok
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if b <= 0 || a < b {
		return a
	}
	return b
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
