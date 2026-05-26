package commands

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

func cmdResearch(args []string) error {
	if len(args) < 1 {
		printResearchUsage()
		return nil
	}
	switch args[0] {
	case "run":
		cmdResearchRun(args[1:])
	case "export":
		cmdResearchExport(args[1:])
	case "keywords":
		cmdResearchKeywords(args[1:])
	case "persona":
		cmdResearchPersona(args[1:])
	default:
		printResearchUsage()
	}
	return nil
}

func cmdResearchRun(raw []string) error {
	args := cmdutil.Parse(raw)
	role := strings.TrimSpace(args.String("role", ""))
	personaName := strings.TrimSpace(args.String("persona", ""))
	if role == "" && personaName != "" {
		if persona, ok := loadPersonaByName(personaName); ok {
			role = persona.RoleDescription
		}
	}
	if input := strings.TrimSpace(args.String("input", "")); input != "" {
		data, err := os.ReadFile(input)
		if err != nil {
			printResearchError("read_input_failed", fmt.Sprintf("Error: read input failed: %v\n", err))
			return nil
		}
		role = strings.TrimSpace(string(data))
	}
	if role == "" {
		role = strings.TrimSpace(args.Pos(0))
	}
	if role == "" {
		printResearchError("role_required", "usage: luma-cli research run --role <description> [--mode precise|expanded] [--date-range 24h|7d] [--output research.json]\n")
		return nil
	}
	mode := strings.TrimSpace(args.String("mode", "precise"))
	dateRange := strings.TrimSpace(args.String("date-range", "7d"))
	outputPath := strings.TrimSpace(args.String("output", "step0_content_research.json"))
	cfg := loadConfig()
	if cfg == nil {
		printResearchError("not_logged_in", "Error: not logged in. Run: luma-cli auth login <card_key>\n")
		return nil
	}
	resp, err := cloud.RunResearch(role, mode, dateRange, cfg.CardKey)
	if err != nil {
		printResearchError("research_failed", fmt.Sprintf("Error: research failed: %v\n", err))
		return nil
	}
	savedPath := ""
	if outputPath != "" {
		abs, err := absoluteOutputPath(outputPath)
		if err != nil {
			printResearchError("bad_output_path", fmt.Sprintf("Error: bad output path: %v\n", err))
			return nil
		}
		if err := writeJSONFile(abs, resp); err != nil {
			printResearchError("write_output_failed", fmt.Sprintf("Error: write output failed: %v\n", err))
			return nil
		}
		savedPath = abs
		recordProjectArtifact("research", savedPath, "research.run")
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"response": resp, "output_path": savedPath}})
		return nil
	}
	fmt.Println("Research completed.")
	if savedPath != "" {
		fmt.Printf("Saved to: %s\n", savedPath)
	}
	if count, ok := resp.Result["result_count"]; ok {
		fmt.Printf("Result count: %v\n", count)
	}
	return nil
}

func cmdResearchExport(raw []string) error {
	args := cmdutil.Parse(raw)
	inputPath := strings.TrimSpace(args.String("input", ""))
	if inputPath == "" {
		inputPath = strings.TrimSpace(args.Pos(0))
	}
	if inputPath == "" {
		fmt.Println("usage: luma-cli research export --input step0_content_research.json [--output step0_content_research.csv]")
		return nil
	}
	outputPath := strings.TrimSpace(args.String("output", "step0_content_research.csv"))
	payload, err := readJSONObject(inputPath)
	if err != nil {
		printResearchError("read_input_failed", fmt.Sprintf("Error: read input failed: %v\n", err))
		return nil
	}
	results := researchResults(payload)
	abs, err := absoluteOutputPath(outputPath)
	if err != nil {
		printResearchError("bad_output_path", fmt.Sprintf("Error: bad output path: %v\n", err))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		printResearchError("create_output_dir_failed", fmt.Sprintf("Error: create output dir failed: %v\n", err))
		return nil
	}
	file, err := os.Create(abs)
	if err != nil {
		printResearchError("write_output_failed", fmt.Sprintf("Error: write output failed: %v\n", err))
		return nil
	}
	defer file.Close()
	if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		printResearchError("write_output_failed", fmt.Sprintf("Error: write output failed: %v\n", err))
		return nil
	}
	writer := csv.NewWriter(file)
	headers := []string{"keyword", "title", "author", "likes", "followers", "duration_seconds", "publish_time", "link", "video_id", "content_type", "note"}
	_ = writer.Write(headers)
	for _, item := range results {
		row := make([]string, 0, len(headers))
		for _, key := range headers {
			row = append(row, strAny(item[key]))
		}
		_ = writer.Write(row)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		printResearchError("write_output_failed", fmt.Sprintf("Error: write output failed: %v\n", err))
		return nil
	}
	recordProjectArtifact("research_table", abs, "research.export")
	writeSimpleResult(map[string]any{"output_path": abs, "count": len(results)})
	return nil
}

func printResearchError(code, message string) {
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: false, Code: code, Error: strings.TrimSpace(message)})
		return
	}
	fmt.Print(message)
}

func printResearchUsage() {
	fmt.Println("luma-cli research <subcommand>")
	fmt.Println("  run          Run content research")
	fmt.Println("  export       Export research results")
	fmt.Println("  keywords     Extract keywords from research")
	fmt.Println("  persona      Manage researcher personas")
}
