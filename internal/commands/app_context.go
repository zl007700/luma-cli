package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	appconfig "github.com/luma-cli/lumer-cli/internal/config"
	"github.com/luma-cli/lumer-cli/project"
)

type config = appconfig.Config

func loadConfig() *config {
	cfg, err := appconfig.Load()
	if err != nil {
		return nil
	}
	return cfg
}

func requireConfig() (*config, error) {
	cfg := loadConfig()
	if cfg == nil || cfg.CardKey == "" {
		return nil, fmt.Errorf("not logged in. Run: luma-cli auth login <card_key>")
	}
	return cfg, nil
}

// recordStep records a processing step to the project if one is active.
func recordStep(proj *project.Project, command, input, output string) {
	if proj == nil {
		return
	}
	proj.AddStep(project.Step{
		Command:   command,
		Timestamp: time.Now(),
		Input:     input,
		Output:    output,
	})
}

func absoluteOutputPath(outputPath string) (string, error) {
	if outputPath == "" {
		return "", nil
	}
	if filepath.IsAbs(outputPath) {
		return filepath.Clean(outputPath), nil
	}
	return filepath.Abs(outputPath)
}

// ensureOutputDir resolves an output path to absolute and creates its
// parent directory. Returns the absolute path on success.

// resolveProjectOutput routes a relative output path into the project output
// directory when a project is active. Absolute paths are returned unchanged.
func resolveProjectOutput(proj *project.Project, outputPath string) string {
	if proj == nil || outputPath == "" || filepath.IsAbs(outputPath) {
		return outputPath
	}
	return filepath.Join(proj.SubDir(project.DirOutput), outputPath)
}

func ensureOutputDir(outputPath string) (string, error) {
	abs, err := absoluteOutputPath(outputPath)
	if err != nil {
		return "", fmt.Errorf("bad output path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return "", fmt.Errorf("create output dir failed: %w", err)
	}
	return abs, nil
}
