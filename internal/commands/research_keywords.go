package commands

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

func cmdResearchKeywords(raw []string) error {
	args := cmdutil.Parse(raw)
	inputPath := strings.TrimSpace(args.String("input", ""))
	if inputPath == "" {
		inputPath = strings.TrimSpace(args.Pos(0))
	}
	if inputPath == "" {
		fmt.Println("usage: luma-cli research keywords --input step0_content_research.json [--output step0_keywords.json] [--csv step0_keywords.csv]")
		return nil
	}
	outputPath := strings.TrimSpace(args.String("output", "step0_keywords.json"))
	csvPath := strings.TrimSpace(args.String("csv", ""))
	payload, err := readJSONObject(inputPath)
	if err != nil {
		printResearchError("read_input_failed", fmt.Sprintf("Error: read input failed: %v\n", err))
		return nil
	}
	rows, summary := extractResearchKeywords(researchResults(payload))
	savedPath := ""
	if outputPath != "" {
		abs, err := absoluteOutputPath(outputPath)
		if err != nil {
			printResearchError("bad_output_path", fmt.Sprintf("Error: bad output path: %v\n", err))
			return nil
		}
		if err := writeJSONFile(abs, map[string]any{"keywords": rows, "summary": summary}); err != nil {
			printResearchError("write_output_failed", fmt.Sprintf("Error: write output failed: %v\n", err))
			return nil
		}
		savedPath = abs
		recordProjectArtifact("research_keywords", abs, "research.keywords")
	}
	csvSavedPath := ""
	if csvPath != "" {
		abs, err := absoluteOutputPath(csvPath)
		if err != nil {
			printResearchError("bad_output_path", fmt.Sprintf("Error: bad csv path: %v\n", err))
			return nil
		}
		if err := writeResearchKeywordsCSV(abs, rows); err != nil {
			printResearchError("write_output_failed", fmt.Sprintf("Error: write csv failed: %v\n", err))
			return nil
		}
		csvSavedPath = abs
		recordProjectArtifact("research_keywords_table", abs, "research.keywords")
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"keywords": rows, "summary": summary, "output_path": savedPath, "csv_path": csvSavedPath}})
		return nil
	}
	fmt.Printf("Keywords: %d\n", len(rows))
	if savedPath != "" {
		fmt.Printf("Saved to: %s\n", savedPath)
	}
	if csvSavedPath != "" {
		fmt.Printf("CSV saved to: %s\n", csvSavedPath)
	}
	return nil
}

func extractResearchKeywords(results []map[string]any) ([]map[string]any, map[string]any) {
	type entry struct {
		Keyword string `json:"keyword"`
		Count   int    `json:"count"`
		Items   []int  `json:"-"`
	}
	keywordMap := map[string]*entry{}
	for i, item := range results {
		for _, kw := range researchItemKeywords(item) {
			kw = strings.TrimSpace(kw)
			if kw == "" {
				continue
			}
			e, ok := keywordMap[kw]
			if !ok {
				e = &entry{Keyword: kw}
				keywordMap[kw] = e
			}
			e.Count++
			e.Items = append(e.Items, i)
		}
	}
	list := make([]*entry, 0, len(keywordMap))
	for _, e := range keywordMap {
		list = append(list, e)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Count > list[j].Count })
	var topKWs []string
	var resultIndices []int
	for _, e := range list {
		if len(topKWs) < 1 {
			if len(e.Items) > 0 {
				resultIndices = e.Items
			}
		}
		topKWs = appendUniqueLimit(topKWs, e.Keyword, 30)
	}
	rows := make([]map[string]any, 0, len(list))
	for _, e := range list {
		rows = append(rows, map[string]any{
			"keyword":           e.Keyword,
			"count":             e.Count,
			"recommended_score": e.Count,
		})
	}
	summary := map[string]any{}
	if len(topKWs) > 0 {
		summary["top_keywords"] = topKWs
	}
	if len(resultIndices) > 0 {
		var topResults []map[string]any
		for _, i := range resultIndices {
			topResults = append(topResults, results[i])
		}
		summary["top_results"] = topResults
	}
	return rows, summary
}

func researchItemKeywords(item map[string]any) []string {
	var keywords []string
	if text, ok := item["title"].(string); ok {
		keywords = append(keywords, compactTitleTerms(text)...)
	}
	if text, ok := item["keyword"].(string); ok {
		keywords = append(keywords, compactTitleTerms(text)...)
	}
	if text, ok := item["description"].(string); ok {
		keywords = append(keywords, compactTitleTerms(text)...)
	}
	return uniqueStrings(keywords)
}

func compactTitleTerms(title string) []string {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil
	}
	title = strings.TrimLeft(title, "# ")
	delims := []string{"#", " ", "，", "。", "、", "/", "|", "-"}
	start := 0
	var terms []string
	for i, ch := range title {
		isSpace := ch == '#' || ch == ' ' || ch == '\t'
		isBreak := isSpace
		for _, d := range delims {
			if string(ch) == d {
				isBreak = true
				break
			}
		}
		if isBreak && i > start {
			term := strings.TrimSpace(title[start:i])
			if len(term) >= 2 {
				terms = append(terms, term)
			}
			start = i + 1
		} else if isSpace {
			start = i + 1
		}
	}
	if start < len([]rune(title)) {
		term := strings.TrimSpace(title[start:])
		if len(term) >= 2 {
			terms = append(terms, term)
		}
	}
	return terms
}

func writeResearchKeywordsCSV(path string, rows []map[string]any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	writer := csv.NewWriter(file)
	headers := []string{"keyword", "count", "recommended_score"}
	_ = writer.Write(headers)
	for _, row := range rows {
		record := make([]string, 0, len(headers))
		for _, key := range headers {
			record = append(record, strAny(row[key]))
		}
		_ = writer.Write(record)
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
	candidates := compactTitleTerms(value)
	if len(candidates) > 0 {
		slug := strings.ToLower(candidates[0])
		slug = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
				return r
			}
			return '_'
		}, slug)
		return strings.Trim(slug, "_")
	}
	return "persona"
}

func researchResults(payload map[string]any) []map[string]any {
	if list, ok := payload["result"].(map[string]any)["results"].([]any); ok {
		var out []map[string]any
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	}
	if list, ok := payload["results"].([]any); ok {
		var out []map[string]any
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	}
	return nil
}
