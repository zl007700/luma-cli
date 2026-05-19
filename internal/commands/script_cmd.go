package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

func cmdScript(args []string) {
	if len(args) < 1 {
		printScriptUsage()
		return
	}
	switch args[0] {
	case "rewrite":
		cmdScriptRewrite(args[1:])
	default:
		fmt.Printf("unknown script subcommand: %s\n\n", args[0])
		printScriptUsage()
	}
}

func cmdScriptRewrite(raw []string) {
	args := cmdutil.Parse(raw)
	text := strings.TrimSpace(args.String("text", ""))
	if input := strings.TrimSpace(args.String("input", "")); input != "" {
		data, err := os.ReadFile(input)
		if err != nil {
			printScriptError("read_input_failed", fmt.Sprintf("Error: read input failed: %v\n", err))
			return
		}
		text = strings.TrimSpace(string(data))
	}
	if text == "" {
		text = strings.TrimSpace(args.Pos(0))
	}
	if text == "" {
		printScriptError("text_required", "usage: luma-cli script rewrite <text> [--length short|medium|long] [--output <path>]\n")
		return
	}
	length := strings.TrimSpace(args.String("length", "short"))
	model := strings.TrimSpace(args.String("model", ""))
	outputPath := strings.TrimSpace(args.String("output", "rewrite.json"))

	cfg := loadConfig()
	if cfg == nil {
		printScriptError("not_logged_in", "Error: not logged in. Run: luma-cli auth login <card_key>\n")
		return
	}

	resp, err := cloud.RewriteScript(text, length, model, cfg.CardKey)
	if err != nil {
		printScriptError("rewrite_failed", fmt.Sprintf("Error: script rewrite failed: %v\n", err))
		return
	}
	if outputPath != "" {
		abs, err := absoluteOutputPath(outputPath)
		if err != nil {
			printScriptError("bad_output_path", fmt.Sprintf("Error: bad output path: %v\n", err))
			return
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			printScriptError("create_output_dir_failed", fmt.Sprintf("Error: create output dir failed: %v\n", err))
			return
		}
		data, _ := json.MarshalIndent(resp, "", "  ")
		if err := os.WriteFile(abs, data, 0644); err != nil {
			printScriptError("write_output_failed", fmt.Sprintf("Error: write output failed: %v\n", err))
			return
		}
		outputPath = abs
	}

	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{
			"response":    resp,
			"output_path": outputPath,
		}})
		return
	}
	rewritten, _ := resp.Result["rewritten_text"].(string)
	fmt.Println("Done! Script rewritten.")
	if outputPath != "" {
		fmt.Printf("Saved to: %s\n", outputPath)
	}
	if rewritten != "" {
		fmt.Println("")
		fmt.Println(rewritten)
	}
}

func printScriptError(code, message string) {
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: false, Code: code, Error: strings.TrimSpace(message)})
		return
	}
	fmt.Print(message)
}

func printScriptUsage() {
	fmt.Println("luma-cli script <subcommand>")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  rewrite <text> [--length short|medium|long] [--output <path>]")
	fmt.Println("  rewrite --input <file> [--length short|medium|long] [--output <path>]")
}
