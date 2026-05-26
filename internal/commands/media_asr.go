package commands

import (
	"github.com/luma-cli/lumer-cli/internal/output"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
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
		return output.ErrValidation(fmt.Sprintf("file not found: %s\n", filePath))
	}

	cfg, err := requireConfig()
	if err != nil {
		return err
	}

	proj := resolveProjectByName("")
	language := parsed.String("language", "zh")

	fmt.Println("Uploading file...")
	fmt.Println("Submitting ASR task...")
	result, err := atom.RunASR(atom.ASROptions{
		FilePath: filePath,
		Language: language,
		CardKey:  cfg.CardKey,
	})
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}
	fmt.Printf("  Uploaded: %s\n", result.ObjectKey)
	fmt.Printf("  Task ID: %s\n", result.TaskID)

	fmt.Printf("\n--- ASR Result ---\n")
	fmt.Printf("Text (%d chars):\n%s\n", len(result.Text), result.Text)
	if len(result.Segments) > 0 {
		fmt.Printf("Segments: %d\n", len(result.Segments))
	}

	if proj != nil {
		asrPath := filepath.Join(proj.SubDir(project.DirAudio), "asr_result.json")
		data, _ := json.MarshalIndent(map[string]any{"text": result.Text, "segments": result.Segments, "language": language}, "", "  ")
		os.WriteFile(asrPath, data, 0644)
		fmt.Printf("  Saved to: %s\n", asrPath)
		recordStep(proj, "asr", filePath, asrPath)
	}
	return nil
}
