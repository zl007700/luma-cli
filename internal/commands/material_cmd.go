package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

type materialDescriptor struct {
	MaterialID    string   `json:"material_id"`
	MediaType     string   `json:"media_type"`
	Path          string   `json:"path"`
	Title         string   `json:"title"`
	Summary       string   `json:"summary,omitempty"`
	Description   string   `json:"description,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Duration      float64  `json:"duration"`
	Resolution    string   `json:"resolution,omitempty"`
	VisualFocus   string   `json:"visual_focus,omitempty"`
	SellingPoints []string `json:"selling_points,omitempty"`
}

type materialGroupView struct {
	ID            string               `json:"id"`
	Name          string               `json:"name,omitempty"`
	Path          string               `json:"path"`
	MaterialCount int                  `json:"material_count"`
	MetaCount     int                  `json:"meta_count,omitempty"`
	Manifest      map[string]any       `json:"manifest,omitempty"`
	Materials     []materialDescriptor `json:"materials,omitempty"`
}

type materialSearchMatch struct {
	Material      materialDescriptor `json:"material"`
	Score         int                `json:"score"`
	MatchedFields []string           `json:"matched_fields"`
}

func cmdMaterial(args []string) {
	if len(args) < 1 {
		printMaterialUsage()
		return
	}
	switch args[0] {
	case "describe":
		cmdMaterialDescribe(args[1:])
	case "group":
		cmdMaterialGroup(args[1:])
	case "library":
		cmdMaterialLibrary(args[1:])
	case "understand":
		cmdMaterialUnderstand(args[1:])
	case "merge":
		cmdMaterialMerge(args[1:])
	case "search":
		cmdMaterialSearch(args[1:])
	default:
		printMaterialUsage()
	}
}

func printMaterialUsage() {
	fmt.Println("luma-cli material <subcommand>")
	fmt.Println("  describe <file_or_dir> [--output materials.json]")
	fmt.Println("  group list [groups_root] [--output material_groups.json]")
	fmt.Println("  group describe <group_name_or_dir> [--output materials.json]")
	fmt.Println("  library path")
	fmt.Println("  library import <group_dir> [--name <group_name>] [--replace]")
	fmt.Println("  understand <file> [--group pip_materials] [--output material_meta.json] [--descriptor-output material.json]")
	fmt.Println("  merge --materials materials.json --meta material_meta.json_or_dir [--output materials_enriched.json]")
	fmt.Println("  search --materials materials.json --query <text> [--limit 10] [--output material_matches.json]")
}

func cmdMaterialDescribe(raw []string) {
	args := cmdutil.Parse(raw)
	inputPath := strings.TrimSpace(args.Pos(0))
	if inputPath == "" {
		inputPath = strings.TrimSpace(args.String("input", ""))
	}
	if inputPath == "" {
		fmt.Println("usage: luma-cli material describe <file_or_dir> [--output materials.json]")
		return
	}
	outputPath := strings.TrimSpace(args.String("output", "materials.json"))
	materials, err := describeMaterials(inputPath)
	if err != nil {
		if runtimeOpts.JSON {
			_ = output.WriteJSON(os.Stdout, output.Envelope{OK: false, Code: "describe_failed", Error: err.Error()})
			return
		}
		fmt.Printf("Error: material describe failed: %v\n", err)
		return
	}
	savedPath := ""
	if outputPath != "" {
		abs, err := absoluteOutputPath(outputPath)
		if err != nil {
			fmt.Printf("Error: bad output path: %v\n", err)
			return
		}
		if err := writeJSONFile(abs, map[string]any{"materials": materials}); err != nil {
			fmt.Printf("Error: write output failed: %v\n", err)
			return
		}
		savedPath = abs
		recordProjectArtifact("materials", savedPath, "material.describe")
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"materials": materials, "output_path": savedPath}})
		return
	}
	fmt.Printf("Materials: %d\n", len(materials))
	if savedPath != "" {
		fmt.Printf("Saved to: %s\n", savedPath)
	}
}

func cmdMaterialUnderstand(raw []string) {
	args := cmdutil.Parse(raw)
	inputPath := strings.TrimSpace(args.Pos(0))
	if inputPath == "" {
		inputPath = strings.TrimSpace(args.String("input", ""))
	}
	if inputPath == "" {
		fmt.Println("usage: luma-cli material understand <file> [--group pip_materials] [--output material_meta.json] [--descriptor-output material.json]")
		return
	}
	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return
	}
	group := strings.TrimSpace(args.String("group", "pip_materials"))
	objectKey, err := cloud.UploadFile(inputPath, cfg.CardKey, group)
	if err != nil {
		fmt.Printf("Error: upload failed: %v\n", err)
		return
	}
	objectName := filepath.Base(objectKey)
	meta, err := cloud.UnderstandResource(group, objectName, cfg.CardKey)
	if err != nil {
		fmt.Printf("Error: understand failed: %v\n", err)
		return
	}
	meta["group"] = group
	meta["object_key"] = objectKey
	meta["object_name"] = objectName

	metaOutput := strings.TrimSpace(args.String("output", "material_meta.json"))
	if metaOutput != "" {
		abs, err := absoluteOutputPath(metaOutput)
		if err != nil {
			fmt.Printf("Error: bad output path: %v\n", err)
			return
		}
		if err := writeJSONFile(abs, meta); err != nil {
			fmt.Printf("Error: write output failed: %v\n", err)
			return
		}
		meta["output_path"] = abs
		recordProjectArtifact("material_meta", abs, "material.understand")
	}

	descriptorOutput := strings.TrimSpace(args.String("descriptor-output", ""))
	if descriptorOutput != "" {
		materials, err := describeMaterials(inputPath)
		if err != nil {
			fmt.Printf("Error: describe local material failed: %v\n", err)
			return
		}
		materials = mergeMetaIntoMaterials(materials, []map[string]any{meta})
		abs, err := absoluteOutputPath(descriptorOutput)
		if err != nil {
			fmt.Printf("Error: bad descriptor output path: %v\n", err)
			return
		}
		if err := writeJSONFile(abs, map[string]any{"materials": materials}); err != nil {
			fmt.Printf("Error: write descriptor failed: %v\n", err)
			return
		}
		meta["descriptor_output_path"] = abs
		recordProjectArtifact("materials", abs, "material.understand")
	}

	writeSimpleResult(map[string]any{
		"group":                  group,
		"object_name":            objectName,
		"output_path":            meta["output_path"],
		"descriptor_output_path": meta["descriptor_output_path"],
	})
}

func cmdMaterialMerge(raw []string) {
	args := cmdutil.Parse(raw)
	materialsPath := strings.TrimSpace(args.String("materials", ""))
	metaPath := strings.TrimSpace(args.String("meta", ""))
	if materialsPath == "" || metaPath == "" {
		fmt.Println("usage: luma-cli material merge --materials materials.json --meta material_meta.json_or_dir [--output materials_enriched.json]")
		return
	}
	outputPath := strings.TrimSpace(args.String("output", "materials_enriched.json"))
	materialsPayload, err := readJSONObject(materialsPath)
	if err != nil {
		fmt.Printf("Error: read materials failed: %v\n", err)
		return
	}
	materials, err := materialDescriptorsFromPayload(materialsPayload)
	if err != nil {
		fmt.Printf("Error: parse materials failed: %v\n", err)
		return
	}
	metas, err := readMaterialMetas(metaPath)
	if err != nil {
		fmt.Printf("Error: read meta failed: %v\n", err)
		return
	}
	materials = mergeMetaIntoMaterials(materials, metas)
	abs, err := absoluteOutputPath(outputPath)
	if err != nil {
		fmt.Printf("Error: bad output path: %v\n", err)
		return
	}
	if err := writeJSONFile(abs, map[string]any{"materials": materials}); err != nil {
		fmt.Printf("Error: write output failed: %v\n", err)
		return
	}
	recordProjectArtifact("materials", abs, "material.merge")
	writeSimpleResult(map[string]any{"output_path": abs, "count": len(materials)})
}

func cmdMaterialSearch(raw []string) {
	args := cmdutil.Parse(raw)
	materialsPath := strings.TrimSpace(args.String("materials", ""))
	query := strings.TrimSpace(args.String("query", ""))
	if materialsPath == "" || query == "" {
		fmt.Println("usage: luma-cli material search --materials materials.json --query <text> [--limit 10] [--output material_matches.json]")
		return
	}
	limit, err := args.Int("limit", 10)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if limit <= 0 {
		limit = 10
	}
	payload, err := readJSONObject(materialsPath)
	if err != nil {
		fmt.Printf("Error: read materials failed: %v\n", err)
		return
	}
	materials, err := materialDescriptorsFromPayload(payload)
	if err != nil {
		fmt.Printf("Error: parse materials failed: %v\n", err)
		return
	}
	matches := searchMaterials(materials, query, limit)
	outputPath := strings.TrimSpace(args.String("output", ""))
	savedPath := ""
	if outputPath != "" {
		abs, err := absoluteOutputPath(outputPath)
		if err != nil {
			fmt.Printf("Error: bad output path: %v\n", err)
			return
		}
		if err := writeJSONFile(abs, map[string]any{"query": query, "matches": matches}); err != nil {
			fmt.Printf("Error: write output failed: %v\n", err)
			return
		}
		savedPath = abs
		recordProjectArtifact("material_matches", savedPath, "material.search")
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"query": query, "matches": matches, "output_path": savedPath}})
		return
	}
	fmt.Printf("Matches: %d\n", len(matches))
	for _, match := range matches {
		fmt.Printf("  score=%d  %s  %s\n", match.Score, match.Material.Title, match.Material.Path)
	}
	if savedPath != "" {
		fmt.Printf("Saved to: %s\n", savedPath)
	}
}

