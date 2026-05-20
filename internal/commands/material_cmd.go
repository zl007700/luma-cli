package commands

import (
	"crypto/sha1"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
		return r == '_' || r == '-' || r == ' ' || r == '.' || r == '，' || r == ','
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
}
