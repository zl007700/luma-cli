package commands

import (
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
