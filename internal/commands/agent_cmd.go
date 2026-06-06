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
	"script.write":       "/v1/agent/script/write",
	"topic.review":       "/v1/agent/topic/review",
	"title.generate":     "/v1/agent/title/generate",
	"storyboard.scene":   "/v1/agent/storyboard/scene",
	"storyboard.shot":    "/v1/agent/storyboard/shot",
	"material.match":     "/v1/agent/material/match",
	"pip.plan":           "/v1/agent/pip/plan",
}

func cmdAgent(args []string) error {
	if len(args) < 1 {
		printAgentUsage()
		return nil
	}
	switch args[0] {
	case "run":
		return cmdAgentRun(args[1:])
	default:
		fmt.Printf("unknown agent subcommand: %s\n\n", args[0])
		printAgentUsage()
	}
	return nil
}

func cmdAgentRun(raw []string) error {
	args := cmdutil.Parse(raw)
	ability := strings.TrimSpace(args.Pos(0))
	if ability == "" {
		return output.ErrValidation("ability is required")
	}
	path := agentAbilityPaths[ability]
	if path == "" {
		return output.ErrValidation("unknown agent ability: %s", ability)
	}
	inputPath := strings.TrimSpace(args.String("input", ""))
	if inputPath == "" {
		return output.ErrValidation("--input payload.json is required")
	}
	payload, err := readAgentPayload(inputPath)
	if err != nil {
		return output.ErrSystem("read input failed: %v", err)
	}
	cfg := loadConfig()
	if cfg == nil {
		return output.ErrAuth("not logged in. Run: luma-cli auth login <phone_or_account>")
	}
	input, _ := payload["input"].(map[string]any)
	if input == nil {
		input = payload
	}
	options, _ := payload["options"].(map[string]any)
	resp, err := cloud.RunAgentAbility(path, input, options, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("agent ability failed: %v", err)
	}
	outputPath := strings.TrimSpace(args.String("output", ""))
	if outputPath != "" {
		abs, err := absoluteOutputPath(outputPath)
		if err != nil {
			return output.ErrValidation("bad output path: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			return output.ErrSystem("create output dir failed: %v", err)
		}
		data, _ := json.MarshalIndent(resp, "", "  ")
		if err := os.WriteFile(abs, data, 0644); err != nil {
			return output.ErrSystem("write output failed: %v", err)
		}
		outputPath = abs
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"response": resp, "output_path": outputPath}})
		return nil
	}
	fmt.Printf("Done! Ability: %s\n", ability)
	fmt.Printf("Request ID: %s\n", resp.RequestID)
	if outputPath != "" {
		fmt.Printf("Saved to: %s\n", outputPath)
	}
	return nil
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
