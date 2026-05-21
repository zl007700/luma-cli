package commands

import (
	"crypto/sha1"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

type researchPersona struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Summary         string `json:"summary"`
	RoleDescription string `json:"role_description"`
	UpdatedAt       string `json:"updated_at"`
}

type personaStore struct {
	Personas []researchPersona `json:"personas"`
}

func cmdResearch(args []string) {
	if len(args) < 1 {
		printResearchUsage()
		return
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
}

func cmdResearchKeywords(raw []string) {
	args := cmdutil.Parse(raw)
	inputPath := strings.TrimSpace(args.String("input", ""))
	if inputPath == "" {
		inputPath = strings.TrimSpace(args.Pos(0))
	}
	if inputPath == "" {
		fmt.Println("usage: luma-cli research keywords --input step0_content_research.json [--output step0_keywords.json] [--csv step0_keywords.csv]")
		return
	}
	outputPath := strings.TrimSpace(args.String("output", "step0_keywords.json"))
	csvPath := strings.TrimSpace(args.String("csv", ""))
	payload, err := readJSONObject(inputPath)
	if err != nil {
		printResearchError("read_input_failed", fmt.Sprintf("Error: read input failed: %v\n", err))
		return
	}
	rows, summary := extractResearchKeywords(researchResults(payload))
	savedPath := ""
	if outputPath != "" {
		abs, err := absoluteOutputPath(outputPath)
		if err != nil {
			printResearchError("bad_output_path", fmt.Sprintf("Error: bad output path: %v\n", err))
			return
		}
		if err := writeJSONFile(abs, map[string]any{"keywords": rows, "summary": summary}); err != nil {
			printResearchError("write_output_failed", fmt.Sprintf("Error: write output failed: %v\n", err))
			return
		}
		savedPath = abs
		recordProjectArtifact("research_keywords", abs, "research.keywords")
	}
	csvSavedPath := ""
	if csvPath != "" {
		abs, err := absoluteOutputPath(csvPath)
		if err != nil {
			printResearchError("bad_output_path", fmt.Sprintf("Error: bad csv path: %v\n", err))
			return
		}
		if err := writeResearchKeywordsCSV(abs, rows); err != nil {
			printResearchError("write_output_failed", fmt.Sprintf("Error: write csv failed: %v\n", err))
			return
		}
		csvSavedPath = abs
		recordProjectArtifact("research_keywords_table", abs, "research.keywords")
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"keywords": rows, "summary": summary, "output_path": savedPath, "csv_path": csvSavedPath}})
		return
	}
	fmt.Printf("Keywords: %d\n", len(rows))
	if savedPath != "" {
		fmt.Printf("Saved to: %s\n", savedPath)
	}
	if csvSavedPath != "" {
		fmt.Printf("CSV saved to: %s\n", csvSavedPath)
	}
}

func cmdResearchRun(raw []string) {
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
			return
		}
		role = strings.TrimSpace(string(data))
	}
	if role == "" {
		role = strings.TrimSpace(args.Pos(0))
	}
	if role == "" {
		printResearchError("role_required", "usage: luma-cli research run --role <description> [--mode precise|expanded] [--date-range 24h|7d] [--output research.json]\n")
		return
	}
	mode := strings.TrimSpace(args.String("mode", "precise"))
	dateRange := strings.TrimSpace(args.String("date-range", "7d"))
	outputPath := strings.TrimSpace(args.String("output", "step0_content_research.json"))
	cfg := loadConfig()
	if cfg == nil {
		printResearchError("not_logged_in", "Error: not logged in. Run: luma-cli auth login <card_key>\n")
		return
	}
	resp, err := cloud.RunResearch(role, mode, dateRange, cfg.CardKey)
	if err != nil {
		printResearchError("research_failed", fmt.Sprintf("Error: research failed: %v\n", err))
		return
	}
	savedPath := ""
	if outputPath != "" {
		abs, err := absoluteOutputPath(outputPath)
		if err != nil {
			printResearchError("bad_output_path", fmt.Sprintf("Error: bad output path: %v\n", err))
			return
		}
		if err := writeJSONFile(abs, resp); err != nil {
			printResearchError("write_output_failed", fmt.Sprintf("Error: write output failed: %v\n", err))
			return
		}
		savedPath = abs
		recordProjectArtifact("research", savedPath, "research.run")
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"response": resp, "output_path": savedPath}})
		return
	}
	fmt.Println("Research completed.")
	if savedPath != "" {
		fmt.Printf("Saved to: %s\n", savedPath)
	}
	if count, ok := resp.Result["result_count"]; ok {
		fmt.Printf("Result count: %v\n", count)
	}
}

func cmdResearchExport(raw []string) {
	args := cmdutil.Parse(raw)
	inputPath := strings.TrimSpace(args.String("input", ""))
	if inputPath == "" {
		inputPath = strings.TrimSpace(args.Pos(0))
	}
	if inputPath == "" {
		fmt.Println("usage: luma-cli research export --input step0_content_research.json [--output step0_content_research.csv]")
		return
	}
	outputPath := strings.TrimSpace(args.String("output", "step0_content_research.csv"))
	payload, err := readJSONObject(inputPath)
	if err != nil {
		printResearchError("read_input_failed", fmt.Sprintf("Error: read input failed: %v\n", err))
		return
	}
	results := researchResults(payload)
	abs, err := absoluteOutputPath(outputPath)
	if err != nil {
		printResearchError("bad_output_path", fmt.Sprintf("Error: bad output path: %v\n", err))
		return
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		printResearchError("create_output_dir_failed", fmt.Sprintf("Error: create output dir failed: %v\n", err))
		return
	}
	file, err := os.Create(abs)
	if err != nil {
		printResearchError("write_output_failed", fmt.Sprintf("Error: write output failed: %v\n", err))
		return
	}
	defer file.Close()
	if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		printResearchError("write_output_failed", fmt.Sprintf("Error: write output failed: %v\n", err))
		return
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
		return
	}
	recordProjectArtifact("research_table", abs, "research.export")
	writeSimpleResult(map[string]any{"output_path": abs, "count": len(results)})
}

func cmdResearchPersona(raw []string) {
	if len(raw) < 1 {
		printResearchUsage()
		return
	}
	switch raw[0] {
	case "list":
		store := loadPersonaStore()
		if runtimeOpts.JSON {
			_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: store})
			return
		}
		if len(store.Personas) == 0 {
			fmt.Println("No personas saved.")
			return
		}
		fmt.Printf("%-24s %-18s %s\n", "ID", "NAME", "SUMMARY")
		for _, item := range store.Personas {
			fmt.Printf("%-24s %-18s %s\n", item.ID, item.Name, item.Summary)
		}
	case "get":
		name := strings.TrimSpace("")
		if len(raw) > 1 {
			name = raw[1]
		}
		if name == "" {
			fmt.Println("usage: luma-cli research persona get <name_or_id>")
			return
		}
		persona, ok := loadPersonaByName(name)
		if !ok {
			printResearchError("persona_not_found", fmt.Sprintf("Error: persona not found: %s\n", name))
			return
		}
		if runtimeOpts.JSON {
			_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: persona})
			return
		}
		data, _ := json.MarshalIndent(persona, "", "  ")
		fmt.Println(string(data))
	case "save":
		args := cmdutil.Parse(raw[1:])
		name := strings.TrimSpace(args.Pos(0))
		if name == "" {
			name = strings.TrimSpace(args.String("name", ""))
		}
		role := strings.TrimSpace(args.String("role", ""))
		if input := strings.TrimSpace(args.String("input", "")); input != "" {
			data, err := os.ReadFile(input)
			if err != nil {
				printResearchError("read_input_failed", fmt.Sprintf("Error: read input failed: %v\n", err))
				return
			}
			role = strings.TrimSpace(string(data))
		}
		if name == "" || role == "" {
			fmt.Println("usage: luma-cli research persona save <name> --role <description>")
			return
		}
		summary := strings.TrimSpace(args.String("summary", ""))
		if summary == "" {
			summary = name
		}
		persona := savePersona(name, role, summary)
		if runtimeOpts.JSON {
			_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: persona})
			return
		}
		fmt.Printf("Saved persona: %s\n", persona.Name)
	default:
		printResearchUsage()
	}
}

func personaStorePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".luma-cli")
	}
	return filepath.Join(configDir, "luma-cli", "personas.json")
}

func loadPersonaStore() personaStore {
	path := personaStorePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return personaStore{}
	}
	var store personaStore
	if err := json.Unmarshal(data, &store); err != nil {
		return personaStore{}
	}
	return store
}

func savePersonaStore(store personaStore) error {
	return writeJSONFile(personaStorePath(), store)
}

func loadPersonaByName(name string) (researchPersona, bool) {
	name = strings.TrimSpace(name)
	store := loadPersonaStore()
	for _, item := range store.Personas {
		if item.ID == name || item.Name == name {
			return item, true
		}
	}
	return researchPersona{}, false
}

func savePersona(name, role, summary string) researchPersona {
	store := loadPersonaStore()
	now := time.Now().Format(time.RFC3339)
	item := researchPersona{
		ID:              "persona_" + slugForID(name),
		Name:            name,
		Summary:         summary,
		RoleDescription: role,
		UpdatedAt:       now,
	}
	replaced := false
	for i := range store.Personas {
		if store.Personas[i].Name == name || store.Personas[i].ID == item.ID {
			store.Personas[i] = item
			replaced = true
			break
		}
	}
	if !replaced {
		store.Personas = append([]researchPersona{item}, store.Personas...)
	}
	_ = savePersonaStore(store)
	return item
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
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  run --role <description> [--mode precise|expanded] [--date-range 24h|7d] [--output step0_content_research.json]")
	fmt.Println("  export --input step0_content_research.json [--output step0_content_research.csv]")
	fmt.Println("  keywords --input step0_content_research.json [--output step0_keywords.json] [--csv step0_keywords.csv]")
	fmt.Println("  run --persona <name_or_id> [--mode precise|expanded] [--output step0_content_research.json]")
	fmt.Println("  persona list")
	fmt.Println("  persona get <name_or_id>")
	fmt.Println("  persona save <name> --role <description>")
}

func researchResults(payload map[string]any) []map[string]any {
	if items := listMap(payload["results"]); len(items) > 0 {
		return items
	}
	if result, ok := payload["result"].(map[string]any); ok {
		return listMap(result["results"])
	}
	return nil
}

func extractResearchKeywords(results []map[string]any) ([]map[string]any, map[string]any) {
	type keywordAgg struct {
		Keyword string
		Count   int
		Likes   float64
		Links   []string
		Titles  []string
	}
	index := map[string]*keywordAgg{}
	for _, item := range results {
		keywords := researchItemKeywords(item)
		for _, keyword := range keywords {
			key := strings.ToLower(keyword)
			agg := index[key]
			if agg == nil {
				agg = &keywordAgg{Keyword: keyword}
				index[key] = agg
			}
			agg.Count++
			if likes, ok := numberAny(item["likes"]); ok {
				agg.Likes += likes
			}
			if link := firstString(item, "link", "url", "share_url"); link != "" {
				agg.Links = appendUniqueLimit(agg.Links, link, 3)
			}
			if title := firstString(item, "title", "desc", "description"); title != "" {
				agg.Titles = appendUniqueLimit(agg.Titles, title, 3)
			}
		}
	}
	rows := make([]map[string]any, 0, len(index))
	for _, agg := range index {
		rows = append(rows, map[string]any{
			"keyword":       agg.Keyword,
			"count":         agg.Count,
			"total_likes":   int(agg.Likes),
			"sample_links":  agg.Links,
			"sample_titles": agg.Titles,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		ci, _ := intAny(rows[i]["count"])
		cj, _ := intAny(rows[j]["count"])
		if ci == cj {
			li, _ := intAny(rows[i]["total_likes"])
			lj, _ := intAny(rows[j]["total_likes"])
			return li > lj
		}
		return ci > cj
	})
	return rows, map[string]any{"keyword_count": len(rows), "source_count": len(results)}
}

func researchItemKeywords(item map[string]any) []string {
	keywords := []string{}
	for _, key := range []string{"keyword", "query", "search_keyword", "topic", "category"} {
		if value := firstString(item, key); value != "" {
			keywords = append(keywords, value)
		}
	}
	keywords = append(keywords, stringListFromKeys(item, "keywords", "tags", "labels", "hashtags")...)
	if len(keywords) == 0 {
		title := firstString(item, "title", "desc", "description")
		keywords = append(keywords, compactTitleTerms(title)...)
	}
	return uniqueStrings(keywords)
}

func compactTitleTerms(title string) []string {
	parts := strings.FieldsFunc(title, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == ',' || r == ';' || r == '#' || r == '，' || r == '。' || r == '、' || r == '：' || r == ':'
	})
	out := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len([]rune(part)) >= 2 {
			out = append(out, part)
		}
	}
	if len(out) > 6 {
		return out[:6]
	}
	return out
}

func writeResearchKeywordsCSV(path string, rows []map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	writer := csv.NewWriter(file)
	headers := []string{"keyword", "count", "total_likes", "sample_titles", "sample_links"}
	_ = writer.Write(headers)
	for _, row := range rows {
		_ = writer.Write([]string{
			strAny(row["keyword"]),
			strAny(row["count"]),
			strAny(row["total_likes"]),
			strings.Join(stringListFromKeys(row, "sample_titles"), " | "),
			strings.Join(stringListFromKeys(row, "sample_links"), " | "),
		})
	}
	writer.Flush()
	return writer.Error()
}

func appendUniqueLimit(items []string, value string, limit int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return items
	}
	for _, item := range items {
		if item == value {
			return items
		}
	}
	if len(items) >= limit {
		return items
	}
	return append(items, value)
}

func slugForID(value string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(value)))
	return fmt.Sprintf("%x", sum[:6])
}
