package commands

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

func cmdMaterial(args []string) {
	if len(args) < 1 {
		printMaterialUsage()
		return
	}
	switch args[0] {
	case "describe":
		cmdMaterialDescribe(args[1:])
	case "understand":
		cmdMaterialUnderstand(args[1:])
	case "merge":
		cmdMaterialMerge(args[1:])
	default:
		printMaterialUsage()
	}
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

func describeMaterials(inputPath string) ([]materialDescriptor, error) {
	root, err := filepath.Abs(inputPath)
	if err != nil {
		return nil, err
	}
	stat, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	paths := []string{}
	if stat.IsDir() {
		err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if isMaterialFile(path) {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else if isMaterialFile(root) {
		paths = append(paths, root)
	}
	ffprobe := bestEffortFFprobe()
	materials := make([]materialDescriptor, 0, len(paths))
	for _, path := range paths {
		abs, _ := filepath.Abs(path)
		mediaType := inferPIPMediaType(abs)
		info := pipMediaInfo{}
		if ffprobe != "" {
			if probed, err := probeMedia(ffprobe, abs); err == nil {
				info = probed
			}
		}
		resolution := ""
		if info.Width > 0 && info.Height > 0 {
			resolution = fmt.Sprintf("%dx%d", info.Width, info.Height)
		}
		title := strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
		materials = append(materials, materialDescriptor{
			MaterialID:  materialID(abs),
			MediaType:   mediaType,
			Path:        abs,
			Title:       title,
			Summary:     title,
			Tags:        tagsFromName(title),
			Duration:    info.Duration,
			Resolution:  resolution,
			VisualFocus: title,
		})
	}
	return materials, nil
}

func bestEffortFFprobe() string {
	if ffmpeg, err := installedFFmpegPath(); err == nil {
		if ffprobe, err := installedFFprobePath(ffmpeg); err == nil {
			return ffprobe
		}
	}
	if ffprobe, err := exec.LookPath("ffprobe"); err == nil {
		return ffprobe
	}
	return ""
}

func isMaterialFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".mov", ".m4v", ".avi", ".mkv", ".webm", ".jpg", ".jpeg", ".png", ".webp", ".bmp":
		return true
	default:
		return false
	}
}

func materialID(path string) string {
	sum := sha1.Sum([]byte(filepath.Clean(path)))
	return "mat_" + fmt.Sprintf("%x", sum[:8])
}

func tagsFromName(name string) []string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-' || r == ' ' || r == '.' || r == ',' || r == '，'
	})
	tags := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			tags = append(tags, part)
		}
	}
	return tags
}

func printMaterialUsage() {
	fmt.Println("luma-cli material <subcommand>")
	fmt.Println("  describe <file_or_dir> [--output materials.json]")
	fmt.Println("  understand <file> [--group pip_materials] [--output material_meta.json] [--descriptor-output material.json]")
	fmt.Println("  merge --materials materials.json --meta material_meta.json_or_dir [--output materials_enriched.json]")
}

func materialDescriptorsFromPayload(payload map[string]any) ([]materialDescriptor, error) {
	data, err := json.Marshal(payload["materials"])
	if err != nil {
		return nil, err
	}
	var materials []materialDescriptor
	if err := json.Unmarshal(data, &materials); err != nil {
		return nil, err
	}
	return materials, nil
}

func readMaterialMetas(path string) ([]map[string]any, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		var metas []map[string]any
		err := filepath.WalkDir(path, func(itemPath string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || strings.ToLower(filepath.Ext(itemPath)) != ".json" {
				return nil
			}
			payload, err := readJSONObject(itemPath)
			if err != nil {
				return err
			}
			if payload["object_name"] == nil {
				payload["object_name"] = filepath.Base(itemPath)
			}
			if payload["file_name"] == nil {
				payload["file_name"] = strings.TrimSuffix(filepath.Base(itemPath), filepath.Ext(itemPath))
			}
			metas = append(metas, payload)
			return nil
		})
		return metas, err
	}
	payload, err := readJSONObject(path)
	if err != nil {
		return nil, err
	}
	if items := listMap(payload["items"]); len(items) > 0 {
		return items, nil
	}
	if items := listMap(payload["metas"]); len(items) > 0 {
		return items, nil
	}
	if items := listMap(payload["materials"]); len(items) > 0 {
		return items, nil
	}
	return []map[string]any{payload}, nil
}

func mergeMetaIntoMaterials(materials []materialDescriptor, metas []map[string]any) []materialDescriptor {
	if len(materials) == 0 || len(metas) == 0 {
		return materials
	}
	for _, meta := range metas {
		index := matchMaterialMeta(materials, meta)
		if index < 0 {
			continue
		}
		applyMaterialMeta(&materials[index], meta)
	}
	return materials
}

func matchMaterialMeta(materials []materialDescriptor, meta map[string]any) int {
	if len(materials) == 1 {
		return 0
	}
	names := []string{
		firstString(meta, "material_id", "id", "resource_id"),
		firstString(meta, "filename", "file_name", "object_name", "object_key", "title", "name"),
	}
	for i, material := range materials {
		for _, name := range names {
			if name == "" {
				continue
			}
			base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
			materialBase := strings.TrimSuffix(filepath.Base(material.Path), filepath.Ext(material.Path))
			if material.MaterialID == name || materialBase == base || strings.Contains(material.Path, name) || strings.Contains(material.Title, base) || strings.Contains(base, material.Title) {
				return i
			}
		}
	}
	return -1
}

func applyMaterialMeta(material *materialDescriptor, meta map[string]any) {
	flat := flattenMaterialMeta(meta)
	if value := firstString(flat, "summary", "description", "content_summary", "visual_summary", "caption"); value != "" {
		material.Summary = value
		material.Description = value
	}
	if value := firstString(flat, "visual_focus", "main_object", "subject", "scene"); value != "" {
		material.VisualFocus = value
	}
	if tags := stringListFromKeys(flat, "tags", "keywords", "labels", "scenes"); len(tags) > 0 {
		material.Tags = uniqueStrings(append(material.Tags, tags...))
	}
	if points := stringListFromKeys(flat, "selling_points", "highlights", "features"); len(points) > 0 {
		material.SellingPoints = uniqueStrings(append(material.SellingPoints, points...))
	}
}

func flattenMaterialMeta(meta map[string]any) map[string]any {
	out := cloneMap(meta)
	for _, key := range []string{"result", "meta", "metadata", "analysis", "understanding", "data"} {
		if nested, ok := meta[key].(map[string]any); ok {
			for nestedKey, value := range nested {
				if _, exists := out[nestedKey]; !exists {
					out[nestedKey] = value
				}
			}
		}
	}
	return out
}

func stringListFromKeys(m map[string]any, keys ...string) []string {
	for _, key := range keys {
		switch value := m[key].(type) {
		case []any:
			var out []string
			for _, item := range value {
				if text := strAny(item); text != "" {
					out = append(out, text)
				}
			}
			if len(out) > 0 {
				return out
			}
		case []string:
			if len(value) > 0 {
				return value
			}
		case string:
			if strings.TrimSpace(value) != "" {
				return tagsFromName(value)
			}
		}
	}
	return nil
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}
