package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/project"
)

func cmdEnhance(args []string) error {
	parsed := cmdutil.Parse(args)
	if len(parsed.Positionals) < 1 {
		fmt.Println("usage: luma-cli enhance <video> [--scale 2|4] [--output <path>]")
		return nil
	}

	videoPath := parsed.Pos(0)
	if _, err := os.Stat(videoPath); err != nil {
		fmt.Printf("Error: file not found: %s\n", videoPath)
		return nil
	}

	scale, err := parsed.Int("scale", 2)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}

	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return nil
	}

	proj := resolveProjectByName("")
	outputPath := parsed.String("output", "")
	if outputPath == "" {
		if proj != nil {
			outputPath = filepath.Join(proj.SubDir(project.DirOutput), strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))+"_enhanced"+filepath.Ext(videoPath))
		} else {
			ext := filepath.Ext(videoPath)
			outputPath = strings.TrimSuffix(videoPath, ext) + "_enhanced" + ext
		}
	}
	outputPath, err = absoluteOutputPath(outputPath)
	if err != nil {
		fmt.Printf("Error: invalid output path: %v\n", err)
		return nil
	}

	fmt.Println("Uploading video...")
	fmt.Println("Submitting Enhance task...")
	fmt.Printf("  Output: %s\n", outputPath)
	result, err := atom.RunEnhance(atom.EnhanceOptions{
		VideoPath:  videoPath,
		Scale:      scale,
		CardKey:    cfg.CardKey,
		OutputPath: outputPath,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	fmt.Printf("  Uploaded: %s\n", result.ObjectKey)
	fmt.Printf("  Task ID: %s\n", result.TaskID)

	fmt.Printf("\nDone! Saved to: %s\n", outputPath)
	recordStep(proj, "enhance", videoPath, outputPath)
	return nil
}
