package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
	"github.com/luma-cli/lumer-cli/project"
)

func cmdASR(args []string) error {
	parsed := cmdutil.Parse(args)
	if len(parsed.Positionals) < 1 {
		fmt.Println("usage: luma-cli asr <video_or_audio> [--language zh|en]")
		return nil
	}

	filePath := parsed.Pos(0)
	if _, err := os.Stat(filePath); err != nil {
		return output.ErrValidation("file not found: %s\n", filePath)
	}

	proj := resolveProjectByName("")
	language := parsed.String("language", "zh")
	outputPath := parsed.String("output", "")

	return runASRFileWithOutput(filePath, language, outputPath, proj)
}

func runASRFileWithOutput(filePath, language, outputPath string, proj *project.Project) error {
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	fmt.Println("Uploading file...")
	fmt.Println("Submitting ASR task...")
	result, err := atom.RunASR(atom.ASROptions{
		FilePath: filePath,
		Language: language,
		CardKey:  cfg.CardKey,
	})
	if err != nil {
		return output.ErrSystem("%v\n", err)
	}
	fmt.Printf("  Uploaded: %s\n", result.ObjectKey)
	fmt.Printf("  Task ID: %s\n", result.TaskID)

	fmt.Printf("\n--- ASR Result ---\n")
	fmt.Printf("Text (%d chars):\n%s\n", len(result.Text), result.Text)
	if len(result.Segments) > 0 {
		fmt.Printf("Segments: %d\n", len(result.Segments))
	}

	if outputPath == "" && proj != nil {
		outputPath = filepath.Join(proj.SubDir(project.DirAudio), "asr_result.json")
	}
	if outputPath != "" {
		abs, err := absoluteOutputPath(outputPath)
		if err != nil {
			return output.ErrValidation("bad output path: %v\n", err)
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			return output.ErrSystem("create output dir failed: %v\n", err)
		}
		data, _ := json.MarshalIndent(map[string]any{
			"text":       result.Text,
			"segments":   result.Segments,
			"language":   language,
			"task_id":    result.TaskID,
			"object_key": result.ObjectKey,
			"source":     filePath,
		}, "", "  ")
		if err := os.WriteFile(abs, data, 0644); err != nil {
			return output.ErrSystem("write output failed: %v\n", err)
		}
		fmt.Printf("  Saved to: %s\n", abs)
		recordStep(proj, "asr", filePath, abs)
		if runtimeOpts.JSON {
			_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{
				"task_id":     result.TaskID,
				"object_key":  result.ObjectKey,
				"text":        result.Text,
				"segments":    result.Segments,
				"output_path": abs,
			}})
		}
	}
	return nil
}
