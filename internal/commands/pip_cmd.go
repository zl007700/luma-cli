package commands

import (
	"fmt"
	"os"
	"strings"

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

func cmdPIP(args []string) error {
	if len(args) < 1 {
		printPIPUsage()
		return nil
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
	return nil
}

func cmdPIPScene(raw []string) error {
	args := cmdutil.Parse(raw)
	segmentsPath := strings.TrimSpace(args.String("segments", ""))
	if segmentsPath == "" {
		segmentsPath = strings.TrimSpace(args.Pos(0))
	}
	if segmentsPath == "" {
		fmt.Println("usage: luma-cli pip scene --segments segments.json [--output step4_scene_units.json]")
		return nil
	}
	outputPath := args.String("output", "step4_scene_units.json")
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	segments, err := loadSegmentsForPIP(segmentsPath)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}
	sceneUnits, err := cloudSceneUnits(segments, cfg.CardKey)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("scene plan failed: %v\n", err))
	}
	absOut, err := absoluteOutputPath(outputPath)
	if err != nil {
		return output.ErrValidation(fmt.Sprintf("bad output path: %v\n", err))
	}
	if err := writeJSONFile(absOut, map[string]any{"scene_units": sceneUnits, "segments": segments}); err != nil {
		return output.ErrSystem(fmt.Sprintf("write output failed: %v\n", err))
	}
	recordProjectArtifact("scene_units", absOut, "pip.scene")
	writeSimpleResult(map[string]any{"output_path": absOut, "scene_count": len(sceneUnits)})
	return nil
}

func cmdPIPMatch(raw []string) error {
	args := cmdutil.Parse(raw)
	scenesPath := strings.TrimSpace(args.String("scenes", ""))
	if scenesPath == "" {
		scenesPath = strings.TrimSpace(args.String("scene-units", ""))
	}
	materialsPath := strings.TrimSpace(args.String("materials", ""))
	if scenesPath == "" || materialsPath == "" {
		fmt.Println("usage: luma-cli pip match --scenes step4_scene_units.json --materials materials.json [--mode auto|cloud|local] [--output step4_material_matches.json]")
		return nil
	}
	outputPath := args.String("output", "step4_material_matches.json")
	mode := strings.ToLower(strings.TrimSpace(args.String("mode", "auto")))
	maxInserts, err := args.Int("max-inserts", 8)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}
	sceneUnits, err := loadSceneUnitsForPIP(scenesPath)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("read scenes failed: %v\n", err))
	}
	materials, err := loadMaterialMapsForPIP(materialsPath)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("read materials failed: %v\n", err))
	}
	inserts, modeUsed, err := matchPIPMaterials(sceneUnits, materials, mode, maxInserts)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("material match failed: %v\n", err))
	}
	assignments := buildPIPAssignments(inserts)
	absOut, err := absoluteOutputPath(outputPath)
	if err != nil {
		return output.ErrValidation(fmt.Sprintf("bad output path: %v\n", err))
	}
	if err := writeJSONFile(absOut, map[string]any{
		"scene_units": sceneUnits,
		"materials":   materials,
		"assignments": assignments,
		"inserts":     inserts,
		"mode":        modeUsed,
	}); err != nil {
		return output.ErrSystem(fmt.Sprintf("write output failed: %v\n", err))
	}
	recordProjectArtifact("material_matches", absOut, "pip.match")
	writeSimpleResult(map[string]any{"output_path": absOut, "matched_count": len(inserts), "mode": modeUsed})
	return nil
}

func cmdPIPScan(raw []string) error {
	args := cmdutil.Parse(raw)
	inputPath := args.Pos(0)
	if inputPath == "" {
		inputPath = args.String("input", "")
	}
	if inputPath == "" {
		fmt.Println("usage: luma-cli pip scan <material_dir> [--output materials.json]")
		return nil
	}
	outputPath := args.String("output", "materials.json")
	materials, err := describeMaterials(inputPath)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("pip scan failed: %v\n", err))
	}
	absOut, err := absoluteOutputPath(outputPath)
	if err != nil {
		return output.ErrValidation(fmt.Sprintf("bad output path: %v\n", err))
	}
	if err := writeJSONFile(absOut, map[string]any{"materials": materials}); err != nil {
		return output.ErrSystem(fmt.Sprintf("write output failed: %v\n", err))
	}
	recordProjectArtifact("materials", absOut, "pip.scan")
	writeSimpleResult(map[string]any{"output_path": absOut, "count": len(materials)})
	return nil
}

func cmdPIPPlan(raw []string) error {
	args := cmdutil.Parse(raw)
	segmentsPath := strings.TrimSpace(args.String("segments", ""))
	materialsPath := strings.TrimSpace(args.String("materials", ""))
	if segmentsPath == "" || materialsPath == "" {
		fmt.Println("usage: luma-cli pip plan --segments segments.json --materials materials.json [--output step5_picture_in_picture_plan.json]")
		return nil
	}
	outputPath := args.String("output", "step5_picture_in_picture_plan.json")
	landscapeRatio, err := args.Float("landscape-height-ratio", 0.32)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}
	matchMode := strings.ToLower(strings.TrimSpace(args.String("match-mode", "auto")))
	maxInserts, err := args.Int("max-inserts", 8)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	segments, err := loadSegmentsForPIP(segmentsPath)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("read segments failed: %v\n", err))
	}
	materials, err := loadMaterialsForPIP(materialsPath)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("read materials failed: %v\n", err))
	}
	if len(segments) == 0 || len(materials) == 0 {
		return output.ErrValidation("segments and materials cannot be empty")
	}
	sceneUnits, err := cloudSceneUnits(segments, cfg.CardKey)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("scene plan failed: %v\n", err))
	}
	materialMaps := listMapFromAny(materials)
	inserts, modeUsed, err := matchPIPMaterials(listMapFromAny(sceneUnits), materialMaps, matchMode, maxInserts)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("material match failed: %v\n", err))
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
		return output.ErrValidation(fmt.Sprintf("bad output path: %v\n", err))
	}
	if err := writeJSONFile(absOut, plan); err != nil {
		return output.ErrSystem(fmt.Sprintf("write output failed: %v\n", err))
	}
	recordProjectArtifact("pip_plan", absOut, "pip.plan")
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"output_path": absOut, "scene_count": len(sceneUnits), "matched_count": len(inserts)}})
		return nil
	}
	fmt.Printf("Plan saved to: %s\n", absOut)
	fmt.Printf("Scenes: %d, matched inserts: %d\n", len(sceneUnits), len(inserts))
	return nil
}

func cmdPIPRender(raw []string) error {
	parsed := cmdutil.Parse(raw)
	videoPath := parsed.Pos(0)
	planPath := parsed.String("plan", "")
	if videoPath == "" || planPath == "" {
		fmt.Println("usage: luma-cli pip render <video> --plan step5_picture_in_picture_plan.json [--output step5_picture_in_picture.mp4]")
		return nil
	}
	outputPath := parsed.String("output", "step5_picture_in_picture.mp4")
	landscapeRatio := 0.0
	var err error
	if parsed.Has("landscape-height-ratio") {
		landscapeRatio, err = parsed.Float("landscape-height-ratio", 0.32)
	}
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}
	absOut, err := absoluteOutputPath(outputPath)
	if err != nil {
		return output.ErrValidation(fmt.Sprintf("bad output path: %v\n", err))
	}
	if err := renderPIPVideo(videoPath, planPath, absOut, landscapeRatio); err != nil {
		return output.ErrSystem(fmt.Sprintf("pip render failed: %v\n", err))
	}
	recordProjectArtifact("pip", absOut, "pip.render")
	writeSimpleResult(map[string]any{"output_path": absOut, "plan_path": planPath})
	return nil
}

func printPIPUsage() {
	fmt.Println("luma-cli pip <subcommand>")
	fmt.Println("  scan <material_dir> [--output materials.json]")
	fmt.Println("  scene --segments segments.json [--output step4_scene_units.json]")
	fmt.Println("  match --scenes step4_scene_units.json --materials materials.json [--mode auto|cloud|local] [--output step4_material_matches.json]")
	fmt.Println("  plan --segments segments.json --materials materials.json [--match-mode auto|cloud|local] [--output step5_picture_in_picture_plan.json]")
	fmt.Println("  render <video> --plan step5_picture_in_picture_plan.json [--output step5_picture_in_picture.mp4] [--landscape-height-ratio 0.32]")
}
