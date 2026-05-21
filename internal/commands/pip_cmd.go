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
