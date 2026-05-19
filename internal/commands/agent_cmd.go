package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

var agentAbilityPaths = map[string]string{
	"subtitle.split":     "/v1/agent/subtitle/split",
	"subtitle.highlight": "/v1/agent/subtitle/highlight",
	"subtitle.effects":   "/v1/agent/subtitle/effects",
	"script.rewrite":     "/v1/agent/script/rewrite",
	"title.generate":     "/v1/agent/title/generate",
	"storyboard.scene":   "/v1/agent/storyboard/scene",
	"storyboard.shot":    "/v1/agent/storyboard/shot",
	"material.match":     "/v1/agent/material/match",
	"pip.plan":           "/v1/agent/pip/plan",
}

func cmdAgent(args []string) {
	if len(args) < 1 {
		printAgentUsage()
		return
	}
	switch args[0] {
	case "run":
		cmdAgentRun(args[1:])
	default:
		fmt.Printf("unknown agent subcommand: %s\n\n", args[0])
		printAgentUsage()
	}
}

func cmdAgentRun(raw []string) {
	args := cmdutil.Parse(raw)
	ability := strings.TrimSpace(args.Pos(0))
	if ability == "" {
		printAgentError("ability_required", "usage: luma-cli agent run <ability> --input payload.json [--output result.json]\n")
		return
	}
	path := agentAbilityPaths[ability]
	if path == "" {
		printAgentError("unknown_ability", fmt.Sprintf("Error: unknown agent ability: %s\n", ability))
		return
	}
	inputPath := strings.TrimSpace(args.String("input", ""))
	if inputPath == "" {
		printAgentError("input_required", "Error: --input payload.json is required\n")
		return
	}
	payload, err := readAgentPayload(inputPath)
	if err != nil {
		printAgentError("read_input_failed", fmt.Sprintf("Error: read input failed: %v\n", err))
		return
	}
	cfg := loadConfig()
	if cfg == nil {
		printAgentError("not_logged_in", "Error: not logged in. Run: luma-cli auth login <card_key>\n")
		return
	}
	input, _ := payload["input"].(map[string]any)
	if input == nil {
		input = payload
	}
	options, _ := payload["options"].(map[string]any)
	resp, err := cloud.RunAgentAbility(path, input, options, cfg.CardKey)
	if err != nil {
		printAgentError("agent_failed", fmt.Sprintf("Error: agent ability failed: %v\n", err))
		return
	}
	outputPath := strings.TrimSpace(args.String("output", ""))
	if outputPath != "" {
		abs, err := absoluteOutputPath(outputPath)
		if err != nil {
			printAgentError("bad_output_path", fmt.Sprintf("Error: bad output path: %v\n", err))
			return
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			printAgentError("create_output_dir_failed", fmt.Sprintf("Error: create output dir failed: %v\n", err))
			return
		}
		data, _ := json.MarshalIndent(resp, "", "  ")
		if err := os.WriteFile(abs, data, 0644); err != nil {
			printAgentError("write_output_failed", fmt.Sprintf("Error: write output failed: %v\n", err))
			return
		}
		outputPath = abs
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"response": resp, "output_path": outputPath}})
		return
	}
	fmt.Printf("Done! Ability: %s\n", ability)
	fmt.Printf("Request ID: %s\n", resp.RequestID)
	if outputPath != "" {
		fmt.Printf("Saved to: %s\n", outputPath)
	}
}

func readAgentPayload(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func printAgentError(code, message string) {
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: false, Code: code, Error: strings.TrimSpace(message)})
		return
	}
	fmt.Print(message)
}

func printAgentUsage() {
	fmt.Println("luma-cli agent <subcommand>")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  run <ability> --input payload.json [--output result.json]")
	fmt.Println("")
	fmt.Println("Abilities:")
	for ability := range agentAbilityPaths {
		fmt.Printf("  %s\n", ability)
	}
}
