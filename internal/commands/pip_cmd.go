package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/clientruntime"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
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
	if len(args) < 1 {
		printPIPUsage()
		return
	}
	switch args[0] {
	case "scan":
		cmdPIPScan(args[1:])
	case "scene":
		cmdPIPScene(args[1:])
	case "match":
		cmdPIPMatch(args[1:])
	case "plan":
		cmdPIPPlan(args[1:])
	case "render":
		cmdPIPRender(args[1:])
	default:
		printPIPUsage()
	}
}

func cmdPIPScene(raw []string) {
	args := cmdutil.Parse(raw)
	segmentsPath := strings.TrimSpace(args.String("segments", ""))
	if segmentsPath == "" {
		segmentsPath = strings.TrimSpace(args.Pos(0))
	}
	if segmentsPath == "" {
		fmt.Println("usage: luma-cli pip scene --segments segments.json [--output step4_scene_units.json]")
		return
	}
	outputPath := args.String("output", "step4_scene_units.json")
	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return
	}
	segments, err := loadSegmentsForPIP(segmentsPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	sceneUnits, err := cloudSceneUnits(segments, cfg.CardKey)
	if err != nil {
		fmt.Printf("Error: scene plan failed: %v\n", err)
		return
	}
	absOut, err := absoluteOutputPath(outputPath)
	if err != nil {
		fmt.Printf("Error: bad output path: %v\n", err)
		return
	}
	if err := writeJSONFile(absOut, map[string]any{"scene_units": sceneUnits, "segments": segments}); err != nil {
		fmt.Printf("Error: write output failed: %v\n", err)
		return
	}
	recordProjectArtifact("scene_units", absOut, "pip.scene")
	writeSimpleResult(map[string]any{"output_path": absOut, "scene_count": len(sceneUnits)})
}

func cmdPIPMatch(raw []string) {
	args := cmdutil.Parse(raw)
	scenesPath := strings.TrimSpace(args.String("scenes", ""))
	if scenesPath == "" {
		scenesPath = strings.TrimSpace(args.String("scene-units", ""))
	}
	materialsPath := strings.TrimSpace(args.String("materials", ""))
	if scenesPath == "" || materialsPath == "" {
		fmt.Println("usage: luma-cli pip match --scenes step4_scene_units.json --materials materials.json [--mode auto|cloud|local] [--output step4_material_matches.json]")
		return
	}
	outputPath := args.String("output", "step4_material_matches.json")
	mode := strings.ToLower(strings.TrimSpace(args.String("mode", "auto")))
	maxInserts, err := args.Int("max-inserts", 8)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	sceneUnits, err := loadSceneUnitsForPIP(scenesPath)
	if err != nil {
		fmt.Printf("Error: read scenes failed: %v\n", err)
		return
	}
	materials, err := loadMaterialMapsForPIP(materialsPath)
	if err != nil {
		fmt.Printf("Error: read materials failed: %v\n", err)
		return
	}
	inserts, modeUsed, err := matchPIPMaterials(sceneUnits, materials, mode, maxInserts)
	if err != nil {
		fmt.Printf("Error: material match failed: %v\n", err)
		return
	}
	assignments := buildPIPAssignments(inserts)
	absOut, err := absoluteOutputPath(outputPath)
	if err != nil {
		fmt.Printf("Error: bad output path: %v\n", err)
		return
	}
	if err := writeJSONFile(absOut, map[string]any{
		"scene_units": sceneUnits,
		"materials":   materials,
		"assignments": assignments,
		"inserts":     inserts,
		"mode":        modeUsed,
	}); err != nil {
		fmt.Printf("Error: write output failed: %v\n", err)
		return
	}
	recordProjectArtifact("material_matches", absOut, "pip.match")
	writeSimpleResult(map[string]any{"output_path": absOut, "matched_count": len(inserts), "mode": modeUsed})
}

func cmdPIPScan(raw []string) {
	args := cmdutil.Parse(raw)
	inputPath := args.Pos(0)
	if inputPath == "" {
		inputPath = args.String("input", "")
	}
	if inputPath == "" {
		fmt.Println("usage: luma-cli pip scan <material_dir> [--output materials.json]")
		return
	}
	outputPath := args.String("output", "materials.json")
	materials, err := describeMaterials(inputPath)
	if err != nil {
		fmt.Printf("Error: pip scan failed: %v\n", err)
		return
	}
	absOut, err := absoluteOutputPath(outputPath)
	if err != nil {
		fmt.Printf("Error: bad output path: %v\n", err)
		return
	}
	if err := writeJSONFile(absOut, map[string]any{"materials": materials}); err != nil {
		fmt.Printf("Error: write output failed: %v\n", err)
		return
	}
	recordProjectArtifact("materials", absOut, "pip.scan")
	writeSimpleResult(map[string]any{"output_path": absOut, "count": len(materials)})
}

func cmdPIPPlan(raw []string) {
	args := cmdutil.Parse(raw)
	segmentsPath := strings.TrimSpace(args.String("segments", ""))
	materialsPath := strings.TrimSpace(args.String("materials", ""))
	if segmentsPath == "" || materialsPath == "" {
		fmt.Println("usage: luma-cli pip plan --segments segments.json --materials materials.json [--output step5_picture_in_picture_plan.json]")
		return
	}
	outputPath := args.String("output", "step5_picture_in_picture_plan.json")
	landscapeRatio, err := args.Float("landscape-height-ratio", 0.32)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	matchMode := strings.ToLower(strings.TrimSpace(args.String("match-mode", "auto")))
	maxInserts, err := args.Int("max-inserts", 8)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return
	}
	segments, err := loadSegmentsForPIP(segmentsPath)
	if err != nil {
		fmt.Printf("Error: read segments failed: %v\n", err)
		return
	}
	materials, err := loadMaterialsForPIP(materialsPath)
	if err != nil {
		fmt.Printf("Error: read materials failed: %v\n", err)
		return
	}
	if len(segments) == 0 || len(materials) == 0 {
		fmt.Println("Error: segments and materials cannot be empty")
		return
	}
	sceneUnits, err := cloudSceneUnits(segments, cfg.CardKey)
	if err != nil {
		fmt.Printf("Error: scene plan failed: %v\n", err)
		return
	}
	materialMaps := listMapFromAny(materials)
	inserts, modeUsed, err := matchPIPMaterials(listMapFromAny(sceneUnits), materialMaps, matchMode, maxInserts)
	if err != nil {
		fmt.Printf("Error: material match failed: %v\n", err)
		return
	}
	assignments := buildPIPAssignments(inserts)
	plan := map[string]any{
		"enabled":                len(inserts) > 0,
		"landscape_height_ratio": landscapeRatio,
		"match_looseness":        args.String("match-looseness", "normal"),
		"match_mode":             modeUsed,
		"scene_units":            sceneUnits,
		"material_candidates":    materials,
		"materials":              materials,
		"assignments":            assignments,
		"inserts":                inserts,
		"planning_mode":          "agent",
		"render_rules": map[string]any{
			"portrait":  "replace",
			"landscape": "center_with_blurred_background",
			"empty":     "keep_original",
		},
		"status": "planned",
	}
	if len(inserts) == 0 {
		plan["status"] = "empty"
	}
	absOut, err := absoluteOutputPath(outputPath)
	if err != nil {
		fmt.Printf("Error: bad output path: %v\n", err)
		return
	}
	if err := writeJSONFile(absOut, plan); err != nil {
		fmt.Printf("Error: write output failed: %v\n", err)
		return
	}
	recordProjectArtifact("pip_plan", absOut, "pip.plan")
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"output_path": absOut, "scene_count": len(sceneUnits), "matched_count": len(inserts)}})
		return
	}
	fmt.Printf("Plan saved to: %s\n", absOut)
	fmt.Printf("Scenes: %d, matched inserts: %d\n", len(sceneUnits), len(inserts))
}

func cmdPIPRender(raw []string) {
	parsed := cmdutil.Parse(raw)
	videoPath := parsed.Pos(0)
	planPath := parsed.String("plan", "")
	if videoPath == "" || planPath == "" {
		fmt.Println("usage: luma-cli pip render <video> --plan step5_picture_in_picture_plan.json [--output step5_picture_in_picture.mp4]")
		return
	}
	outputPath := parsed.String("output", "step5_picture_in_picture.mp4")
	landscapeRatio := 0.0
	var err error
	if parsed.Has("landscape-height-ratio") {
		landscapeRatio, err = parsed.Float("landscape-height-ratio", 0.32)
	}
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

func printPIPUsage() {
	fmt.Println("luma-cli pip <subcommand>")
	fmt.Println("  scan <material_dir> [--output materials.json]")
	fmt.Println("  scene --segments segments.json [--output step4_scene_units.json]")
	fmt.Println("  match --scenes step4_scene_units.json --materials materials.json [--mode auto|cloud|local] [--output step4_material_matches.json]")
	fmt.Println("  plan --segments segments.json --materials materials.json [--match-mode auto|cloud|local] [--output step5_picture_in_picture_plan.json]")
	fmt.Println("  render <video> --plan step5_picture_in_picture_plan.json [--output step5_picture_in_picture.mp4] [--landscape-height-ratio 0.32]")
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
	if landscapeRatio <= 0 {
		if value, ok := numberAny(plan["landscape_height_ratio"]); ok && value > 0 {
			landscapeRatio = value
		} else {
			landscapeRatio = 0.32
		}
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
	inserts := listMap(plan["inserts"])
	if len(inserts) == 0 {
		inserts = listMap(plan["pip_plan"])
	}
	for _, insert := range inserts {
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
	sort.SliceStable(specs, func(i, j int) bool {
		return specs[i].Start < specs[j].Start
	})
	return specs, nil
}

func pipInsertRange(insert map[string]any, scenes map[string]map[string]any, sourceDuration float64) (float64, float64) {
	if start, ok := numberAny(insert["start"]); ok {
		end, _ := numberAny(insert["end"])
		if end <= start {
			_, end = pipInsertSceneRange(insert, scenes)
		}
		return maxFloat(start, 0), minFloat(end, sourceDuration)
	}
	start, end := pipInsertSceneRange(insert, scenes)
	return start, minFloat(end, sourceDuration)
}

func pipInsertSceneRange(insert map[string]any, scenes map[string]map[string]any) (float64, float64) {
	startScene := scenes[firstString(insert, "start_scene_id", "start_sent_id", "scene_id")]
	endSceneID := firstString(insert, "end_scene_id", "end_sent_id")
	if endSceneID == "" {
		endSceneID = firstString(insert, "start_scene_id", "start_sent_id", "scene_id")
	}
	endScene := scenes[endSceneID]
	start, _ := numberAny(startScene["start"])
	end, _ := numberAny(endScene["end"])
	return maxFloat(start, 0), end
}

func buildPIPPrepareFilter(inputIndex int, outputLabel string, spec pipRenderSpec, sourceWidth, sourceHeight int, landscapeRatio float64) string {
	_ = landscapeRatio
	scale := fmt.Sprintf("scale=%d:-2,setsar=1", sourceWidth)
	if spec.Replace {
		scale = fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,setsar=1", sourceWidth, sourceHeight, sourceWidth, sourceHeight)
	}
	return fmt.Sprintf("[%d:v]setpts=PTS-STARTPTS+%.6f/TB,%s,trim=duration=%.6f[%s]", inputIndex, spec.Start, scale, spec.Duration, outputLabel)
}

func loadSegmentsForPIP(path string) ([]any, error) {
	payload, err := readJSONObject(path)
	if err != nil {
		return nil, err
	}
	segments := anyListFromPayload(payload, "segments")
	if len(segments) == 0 {
		return nil, fmt.Errorf("segments cannot be empty")
	}
	return segments, nil
}

func loadMaterialsForPIP(path string) ([]any, error) {
	payload, err := readJSONObject(path)
	if err != nil {
		return nil, err
	}
	materials := anyListFromPayload(payload, "materials")
	if len(materials) == 0 {
		return nil, fmt.Errorf("materials cannot be empty")
	}
	return materials, nil
}

func loadMaterialMapsForPIP(path string) ([]map[string]any, error) {
	materials, err := loadMaterialsForPIP(path)
	if err != nil {
		return nil, err
	}
	return listMapFromAny(materials), nil
}

func loadSceneUnitsForPIP(path string) ([]map[string]any, error) {
	payload, err := readJSONObject(path)
	if err != nil {
		return nil, err
	}
	sceneUnits := anyListFromPayload(payload, "scene_units")
	if len(sceneUnits) == 0 {
		sceneUnits = anyListFromPayload(payload, "scenes")
	}
	if len(sceneUnits) == 0 {
		return nil, fmt.Errorf("scene_units cannot be empty")
	}
	return listMapFromAny(sceneUnits), nil
}

func cloudSceneUnits(segments []any, cardKey string) ([]any, error) {
	sceneResp, err := cloud.RunAgentAbility("/v1/agent/storyboard/scene", map[string]any{"segments": segments}, nil, cardKey)
	if err != nil {
		return nil, err
	}
	sceneUnits := anyListFromPayload(sceneResp.Result, "scene_units")
	if len(sceneUnits) == 0 {
		return nil, fmt.Errorf("backend returned no scene_units")
	}
	return sceneUnits, nil
}

func matchPIPMaterials(sceneUnits, materials []map[string]any, mode string, maxInserts int) ([]map[string]any, string, error) {
	if maxInserts <= 0 {
		maxInserts = 8
	}
	if mode == "" {
		mode = "auto"
	}
	if mode != "local" {
		cfg := loadConfig()
		if cfg != nil {
			matchResp, err := cloud.RunAgentAbility("/v1/agent/material/match", map[string]any{"scene_units": sceneUnits, "materials": materials, "max_inserts": maxInserts}, nil, cfg.CardKey)
			if err == nil {
				rawInserts := anyListFromPayload(matchResp.Result, "inserts")
				inserts := normalizePIPInserts(rawInserts, sceneUnits, materials)
				if len(inserts) > maxInserts {
					inserts = inserts[:maxInserts]
				}
				return inserts, "cloud", nil
			}
			if mode == "cloud" {
				return nil, "cloud", err
			}
		} else if mode == "cloud" {
			return nil, "cloud", fmt.Errorf("not logged in. Run: luma-cli auth login <card_key>")
		}
	}
	return localPIPMaterialMatches(sceneUnits, materials, maxInserts), "local", nil
}

func localPIPMaterialMatches(sceneUnits, materials []map[string]any, maxInserts int) []map[string]any {
	descriptors := materialDescriptorsFromMaps(materials)
	inserts := []map[string]any{}
	used := map[string]bool{}
	for _, scene := range sceneUnits {
		query := sceneSearchText(scene)
		if query == "" {
			continue
		}
		matches := searchMaterials(descriptors, query, len(descriptors))
		if len(matches) == 0 {
			continue
		}
		var chosen materialSearchMatch
		found := false
		for _, match := range matches {
			id := match.Material.MaterialID
			if id == "" {
				id = match.Material.Path
			}
			if !used[id] {
				chosen = match
				used[id] = true
				found = true
				break
			}
		}
		if !found {
			continue
		}
		sceneID := firstString(scene, "scene_id", "id", "segment_id", "sent_id")
		insert := map[string]any{
			"scene_id":        sceneID,
			"start_scene_id":  sceneID,
			"end_scene_id":    sceneID,
			"material_id":     chosen.Material.MaterialID,
			"material_path":   chosen.Material.Path,
			"media_type":      chosen.Material.MediaType,
			"start":           scene["start"],
			"end":             scene["end"],
			"score":           chosen.Score,
			"matched_fields":  chosen.MatchedFields,
			"reason":          "local keyword match",
			"planning_source": "local",
		}
		inserts = append(inserts, insert)
		if len(inserts) >= maxInserts {
			break
		}
	}
	return normalizePIPInserts(anySliceFromMaps(inserts), sceneUnits, materials)
}

func materialDescriptorsFromMaps(items []map[string]any) []materialDescriptor {
	out := make([]materialDescriptor, 0, len(items))
	for _, item := range items {
		descriptor := materialDescriptor{
			MaterialID:    firstString(item, "material_id", "id", "resource_id"),
			MediaType:     firstNonEmpty(firstString(item, "media_type", "type"), "video"),
			Path:          firstString(item, "path", "file_path", "local_path", "material_path"),
			Title:         firstString(item, "title", "name", "file_name"),
			Summary:       firstString(item, "summary", "description", "content_summary", "visual_summary"),
			Description:   firstString(item, "description", "summary", "content_summary", "visual_summary"),
			Tags:          stringListFromKeys(item, "tags", "keywords", "labels"),
			VisualFocus:   firstString(item, "visual_focus", "main_object", "subject", "scene"),
			SellingPoints: stringListFromKeys(item, "selling_points", "highlights", "features"),
		}
		if descriptor.MaterialID == "" && descriptor.Path != "" {
			descriptor.MaterialID = materialID(descriptor.Path)
		}
		if descriptor.Title == "" {
			descriptor.Title = strings.TrimSuffix(filepath.Base(descriptor.Path), filepath.Ext(descriptor.Path))
		}
		out = append(out, descriptor)
	}
	return out
}

func sceneSearchText(scene map[string]any) string {
	parts := []string{
		firstString(scene, "title", "topic", "scene_title"),
		firstString(scene, "summary", "description", "content", "text", "script"),
		firstString(scene, "visual_focus", "main_object", "subject"),
		strings.Join(stringListFromKeys(scene, "keywords", "tags", "labels"), " "),
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func anySliceFromMaps(items []map[string]any) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}

func normalizePIPInserts(raw []any, scenes []map[string]any, materials []map[string]any) []map[string]any {
	sceneMap := map[string]map[string]any{}
	for _, scene := range scenes {
		id := firstString(scene, "scene_id", "sent_id", "segment_id", "id")
		if id != "" {
			sceneMap[id] = scene
		}
	}
	materialIDs := map[string]bool{}
	for _, material := range materials {
		id := firstString(material, "material_id", "id", "resource_id")
		if id != "" {
			materialIDs[id] = true
		}
	}
	inserts := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		insert, ok := item.(map[string]any)
		if !ok {
			continue
		}
		normalized := cloneMap(insert)
		materialID := firstString(normalized, "material_id", "id", "resource_id")
		if materialID == "" {
			continue
		}
		normalized["material_id"] = materialID
		if !materialIDs[materialID] {
			normalized["material_known"] = false
		}
		sceneID := firstString(normalized, "scene_id", "start_scene_id", "start_sent_id")
		if sceneID != "" {
			if normalized["start_scene_id"] == nil {
				normalized["start_scene_id"] = sceneID
			}
			if normalized["end_scene_id"] == nil {
				normalized["end_scene_id"] = firstNonEmpty(firstString(normalized, "end_scene_id", "end_sent_id"), sceneID)
			}
			if scene, ok := sceneMap[sceneID]; ok {
				if _, ok := normalized["start"]; !ok {
					normalized["start"] = scene["start"]
				}
				if _, ok := normalized["end"]; !ok {
					normalized["end"] = scene["end"]
				}
				if _, ok := normalized["start_segment_id"]; !ok {
					normalized["start_segment_id"] = firstString(scene, "start_segment_id", "segment_id", "sent_id")
				}
				if _, ok := normalized["end_segment_id"]; !ok {
					normalized["end_segment_id"] = firstString(scene, "end_segment_id", "segment_id", "sent_id")
				}
			}
		}
		inserts = append(inserts, normalized)
	}
	sort.SliceStable(inserts, func(i, j int) bool {
		return insertSortValue(inserts[i]) < insertSortValue(inserts[j])
	})
	filtered := make([]map[string]any, 0, len(inserts))
	lastEnd := -1.0
	for _, insert := range inserts {
		start, _ := numberAny(insert["start"])
		end, _ := numberAny(insert["end"])
		if end > start && start < lastEnd {
			continue
		}
		if end > lastEnd {
			lastEnd = end
		}
		filtered = append(filtered, insert)
	}
	return filtered
}

func buildPIPAssignments(inserts []map[string]any) []map[string]any {
	assignments := make([]map[string]any, 0, len(inserts))
	for _, insert := range inserts {
		assignments = append(assignments, map[string]any{
			"scene_id":    firstString(insert, "scene_id", "start_scene_id"),
			"material_id": firstString(insert, "material_id"),
			"start":       insert["start"],
			"end":         insert["end"],
			"reason":      firstString(insert, "reason", "match_reason"),
		})
	}
	return assignments
}

func insertSortValue(insert map[string]any) float64 {
	if start, ok := numberAny(insert["start"]); ok {
		return start
	}
	if id := firstString(insert, "start_segment_id", "segment_id", "sent_id"); id != "" {
		if n, err := strconv.ParseFloat(strings.TrimLeft(id, "s_"), 64); err == nil {
			return n
		}
	}
	return 0
}

func listMapFromAny(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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

func readJSONObject(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func anyListFromPayload(payload map[string]any, key string) []any {
	value := payload[key]
	if items, ok := value.([]any); ok {
		return items
	}
	if result, ok := payload["result"].(map[string]any); ok {
		if items, ok := result[key].([]any); ok {
			return items
		}
	}
	return nil
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
