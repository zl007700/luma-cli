package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/output"
	"github.com/luma-cli/lumer-cli/project"
)

func cmdProjectManifest(args []string) error {
	p, err := resolveProject(args)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	data, _ := json.MarshalIndent(p, "", "  ")
	fmt.Println(string(data))
	return nil
}

func cmdProjectArtifact(args []string) error {
	if len(args) < 1 {
		printProjectArtifactUsage()
		return nil
	}
	switch args[0] {
	case "add":
		cmdProjectArtifactAdd(args[1:])
	case "list", "ls":
		cmdProjectArtifactList(args[1:])
	case "schema":
		cmdProjectArtifactSchema()
	default:
		printProjectArtifactUsage()
	}
	return nil
}

func printProjectArtifactUsage() {
	fmt.Println("luma-cli project artifact <subcommand>")
	fmt.Println("  add <path> --type <type> [--id <id>] [--step <step>]")
	fmt.Println("  list [--type <type>]")
	fmt.Println("  schema")
}

func cmdProjectArtifactAdd(args []string) error {
	parsed := parseProjectKV(args)
	path := parsed["path"]
	if path == "" {
		path = parsed["_pos0"]
	}
	artifactType := parsed["type"]
	if path == "" || artifactType == "" {
		fmt.Println("usage: luma-cli project artifact add <path> --type <type> [--id <id>] [--step <step>]")
		return nil
	}
	p, err := resolveProject(args[1:])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("Error: bad artifact path: %v\n", err)
		return nil
	}
	if err := p.AddArtifact(project.Artifact{
		ID:   parsed["id"],
		Type: artifactType,
		Path: abs,
		Step: parsed["step"],
	}); err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	fmt.Printf("Artifact added: %s\n", abs)
	return nil
}

func cmdProjectArtifactList(args []string) error {
	parsed := parseProjectKV(args)
	filterType := parsed["type"]
	p, err := resolveProject(args)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	artifacts := make([]project.Artifact, 0, len(p.Artifacts))
	for _, artifact := range p.Artifacts {
		if filterType != "" && artifact.Type != filterType {
			continue
		}
		artifacts = append(artifacts, artifact)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"project": p.Name, "artifacts": artifacts}})
		return nil
	}
	if len(artifacts) == 0 {
		fmt.Println("No artifacts found.")
		return nil
	}
	fmt.Printf("Artifacts for project: %s\n", p.Name)
	fmt.Println("TYPE                 STEP                         STATUS       PATH")
	fmt.Println("--------------------------------------------------------------------------------")
	for _, artifact := range artifacts {
		status := artifact.Status
		if status == "" {
			status = "unknown"
		}
		step := artifact.Step
		if step == "" {
			step = artifact.Ability
		}
		fmt.Printf("%-20s %-28s %-12s %s\n", artifact.Type, step, status, artifact.Path)
	}
	return nil
}

func cmdProjectArtifactSchema() error {
	schema := map[string]any{
		"id":         "stable artifact id; generated when omitted",
		"type":       "artifact category, for example script, audio, video, segments, materials, pip_plan, cover",
		"path":       "absolute local path, or cloud object key when the artifact is remote",
		"step":       "workflow step or atom id that produced this artifact",
		"ability":    "atom id, usually same as step",
		"command":    "optional original command line",
		"inputs":     []string{"optional input paths or ids"},
		"outputs":    []string{"output paths or ids"},
		"status":     "completed | failed | skipped | pending",
		"created_at": "RFC3339 timestamp",
		"meta":       "optional atom-specific metadata",
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: schema})
		return nil
	}
	data, _ := json.MarshalIndent(schema, "", "  ")
	fmt.Println(string(data))
	return nil
}

func parseProjectKV(args []string) map[string]string {
	out := map[string]string{}
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") && i+1 < len(args) {
			out[strings.TrimPrefix(args[i], "--")] = args[i+1]
			i++
			continue
		}
		if out["_pos0"] == "" {
			out["_pos0"] = args[i]
		}
	}
	return out
}
