package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

type originalScriptRunState struct {
	RunID        string         `json:"run_id"`
	ProfileID    string         `json:"profile_id"`
	TopicHint    string         `json:"topic_hint,omitempty"`
	Status       string         `json:"status"`
	CurrentStep  string         `json:"current_step"`
	StartedAt    time.Time      `json:"started_at"`
	FinishedAt   time.Time      `json:"finished_at,omitempty"`
	OutputDir    string         `json:"output_dir"`
	Warnings     []string       `json:"warnings,omitempty"`
	CloudAssets  map[string]any `json:"cloud_assets,omitempty"`
	StageUsage   map[string]any `json:"stage_usage,omitempty"`
	ReviewRounds int            `json:"review_rounds"`
}

func cmdContentOriginalScript(raw []string) error {
	if len(raw) < 1 {
		printContentOriginalScriptUsage()
		return nil
	}
	switch raw[0] {
	case "run":
		return cmdContentOriginalScriptRun(raw[1:])
	default:
		printContentOriginalScriptUsage()
		return nil
	}
}

func cmdContentOriginalScriptRun(raw []string) error {
	args := cmdutil.Parse(raw)
	profileID := strings.TrimSpace(args.String("profile", args.Pos(0)))
	if profileID == "" {
		return output.ErrValidation("--profile <profile_id> is required")
	}
	cfg, err := requireConfig()
	if err != nil {
		return output.ErrAuth("%v", err)
	}
	timeoutSec, err := args.Int("timeout", 240)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	if timeoutSec <= 0 {
		timeoutSec = 240
	}
	maxTokens, err := args.Int("max-tokens", 5000)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	topicHint := strings.TrimSpace(args.String("topic-hint", ""))
	runID := strings.TrimSpace(args.String("run-id", ""))
	if runID == "" {
		runID = "original_script_" + time.Now().Format("20060102_150405")
	}
	outputDir := strings.TrimSpace(args.String("output", filepath.Join("runs", runID)))
	outputAbs, err := absoluteOutputPath(outputDir)
	if err != nil {
		return output.ErrValidation("bad output path: %v", err)
	}
	if err := os.MkdirAll(outputAbs, 0755); err != nil {
		return output.ErrSystem("create output dir failed: %v", err)
	}

	state := originalScriptRunState{
		RunID:       runID,
		ProfileID:   profileID,
		TopicHint:   topicHint,
		Status:      "running",
		CurrentStep: "load_profile",
		StartedAt:   time.Now(),
		OutputDir:   outputAbs,
		CloudAssets: map[string]any{},
		StageUsage:  map[string]any{},
	}
	saveOriginalScriptState(outputAbs, state)

	profile, err := loadProfile(profileID)
	if err != nil {
		return output.ErrSystem("load profile failed: %v", err)
	}
	profileMap, err := structToMap(profile)
	if err != nil {
		return output.ErrSystem("serialize profile failed: %v", err)
	}
	if err := writeJSONFile(filepath.Join(outputAbs, "01_profile.json"), profileMap); err != nil {
		return output.ErrSystem("write profile failed: %v", err)
	}

	state.CurrentStep = "load_memory"
	saveOriginalScriptState(outputAbs, state)
	memory := loadOriginalScriptMemory(profileID, cfg.CardKey, &state)
	if err := writeJSONFile(filepath.Join(outputAbs, "02_memory.json"), memory); err != nil {
		return output.ErrSystem("write memory failed: %v", err)
	}

	options := map[string]any{
		"model_tier": "basic_model",
		"max_tokens": maxTokens,
	}
	timeout := time.Duration(timeoutSec) * time.Second

	state.CurrentStep = "topic_refine"
	saveOriginalScriptState(outputAbs, state)
	topicInput := map[string]any{
		"profile":             profileMap,
		"topic_hint":          topicHint,
		"memory":              memory,
		"avoid_recent_topics": originalScriptAvoidRecent(memory, 8),
		"instruction":         "Select or refine one original video-script topic. Avoid repeating recent history even if topic_hint is similar; choose a meaningfully different angle, public entry, and concrete scene.",
	}
	topicResp, err := cloud.ContentStage("topic_refine", map[string]any{"payload": topicInput}, options, cfg.CardKey, timeout)
	if err != nil {
		return output.ErrNetwork("topic refine failed: %v", err)
	}
	state.StageUsage["topic_refine"] = topicResp.Usage
	if err := writeJSONFile(filepath.Join(outputAbs, "03_topic_selection.json"), topicResp.Result); err != nil {
		return output.ErrSystem("write topic selection failed: %v", err)
	}

	state.CurrentStep = "article_write"
	saveOriginalScriptState(outputAbs, state)
	writeInput := map[string]any{
		"profile":          profileMap,
		"topic":            topicResp.Result,
		"topic_hint":       topicHint,
		"memory":           memory,
		"material_context": map[string]any{"note": "MVP original-script runner has not migrated full search expansion yet. Do not invent facts beyond provided context."},
	}
	writeResp, err := cloud.ContentStage("article_write", map[string]any{"payload": writeInput}, options, cfg.CardKey, timeout)
	if err != nil {
		return output.ErrNetwork("article write failed: %v", err)
	}
	state.StageUsage["article_write"] = writeResp.Usage
	if err := writeJSONFile(filepath.Join(outputAbs, "04_article_v1.json"), writeResp.Result); err != nil {
		return output.ErrSystem("write article json failed: %v", err)
	}
	article := originalScriptArticleText(writeResp.Result)
	if strings.TrimSpace(article) == "" {
		return output.ErrSystem("article_write returned empty article")
	}
	if err := os.WriteFile(filepath.Join(outputAbs, "04_article_v1.md"), []byte(article), 0644); err != nil {
		return output.ErrSystem("write article draft failed: %v", err)
	}

	finalArticle, finalReview, rewriteResp, err := reviewAndMaybeRewriteOriginalScript(
		profileMap,
		topicResp.Result,
		memory,
		article,
		options,
		cfg.CardKey,
		timeout,
		outputAbs,
		&state,
	)
	if err != nil {
		return err
	}
	if rewriteResp != nil {
		state.StageUsage["article_rewrite"] = rewriteResp.Usage
	}
	if err := os.WriteFile(filepath.Join(outputAbs, "final.md"), []byte(finalArticle), 0644); err != nil {
		return output.ErrSystem("write final article failed: %v", err)
	}
	if err := writeJSONFile(filepath.Join(outputAbs, "final_review.json"), finalReview); err != nil {
		return output.ErrSystem("write final review failed: %v", err)
	}

	state.CurrentStep = "save_artifacts"
	saveOriginalScriptState(outputAbs, state)
	saveOriginalScriptCloudArtifacts(profileID, runID, topicResp.Result, finalArticle, finalReview, cfg.CardKey, &state)
	updateOriginalScriptHistory(profileID, runID, topicHint, topicResp.Result, finalArticle, finalReview, cfg.CardKey, &state)

	state.Status = "done"
	state.CurrentStep = "done"
	state.FinishedAt = time.Now()
	saveOriginalScriptState(outputAbs, state)

	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{
			"run_id":       runID,
			"output_dir":   outputAbs,
			"final_path":   filepath.Join(outputAbs, "final.md"),
			"review_path":  filepath.Join(outputAbs, "final_review.json"),
			"state":        state,
			"final_review": finalReview,
		}})
		return nil
	}
	fmt.Printf("Original script run complete: %s\n", runID)
	fmt.Printf("Output: %s\n", outputAbs)
	fmt.Printf("Final: %s\n", filepath.Join(outputAbs, "final.md"))
	if len(state.Warnings) > 0 {
		fmt.Println("Warnings:")
		for _, warning := range state.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}
	return nil
}

func loadOriginalScriptMemory(profileID, cardKey string, state *originalScriptRunState) map[string]any {
	memory := map[string]any{}
	for _, artifactType := range []string{"evergreen_topics", "topic_bank", "content_history"} {
		data, item, err := cloud.ContentMemoryRead(profileID, artifactType, cardKey)
		if err != nil {
			defaultValue := defaultOriginalScriptMemoryValue(artifactType)
			memory[artifactType] = map[string]any{
				"asset_id": "",
				"value":    defaultValue,
			}
			if saved, saveErr := cloud.ContentMemorySave(profileID, artifactType, defaultValue, cardKey); saveErr == nil {
				memory[artifactType].(map[string]any)["asset_id"] = saved.AssetID
				state.CloudAssets[artifactType] = saved.AssetID
				state.Warnings = append(state.Warnings, fmt.Sprintf("memory %s initialized because it was unavailable: %v", artifactType, err))
			} else {
				state.Warnings = append(state.Warnings, fmt.Sprintf("memory %s unavailable and init failed: read=%v save=%v", artifactType, err, saveErr))
			}
			continue
		}
		var parsed any
		if err := json.Unmarshal(data, &parsed); err != nil {
			parsed = string(data)
		}
		memory[artifactType] = map[string]any{
			"asset_id": item.AssetID,
			"value":    parsed,
		}
	}
	return memory
}

func defaultOriginalScriptMemoryValue(artifactType string) map[string]any {
	now := time.Now().Format(time.RFC3339)
	switch artifactType {
	case "evergreen_topics":
		return map[string]any{
			"items":          []any{},
			"schema_version": 1,
			"initialized_at": now,
			"note":           "Initialized by luma-cli original-script runner. Add reusable evergreen topics here as the workflow matures.",
		}
	case "topic_bank":
		return map[string]any{
			"items":          []any{},
			"schema_version": 1,
			"initialized_at": now,
			"note":           "Initialized by luma-cli original-script runner. Stores reusable topic candidates and rejected duplicate angles.",
		}
	default:
		return map[string]any{
			"items":          []any{},
			"schema_version": 1,
			"initialized_at": now,
		}
	}
}

func originalScriptAvoidRecent(memory map[string]any, limit int) []map[string]any {
	if limit <= 0 {
		limit = 5
	}
	historyValue := originalScriptMemoryValue(memory, "content_history")
	items, _ := historyValue["items"].([]any)
	out := []map[string]any{}
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		avoid := map[string]any{}
		for _, key := range []string{"run_id", "topic_hint", "article_head", "created_at"} {
			if value := entry[key]; value != nil {
				avoid[key] = value
			}
		}
		if topic, ok := entry["topic"].(map[string]any); ok {
			avoid["topic_title"] = firstNonEmpty(
				strAny(topic["title"]),
				strAny(topic["selected_topic"]),
				strAny(topic["public_entry"]),
			)
			avoid["public_entry"] = strAny(topic["public_entry"])
			avoid["reason"] = strAny(topic["reason"])
			avoid["duplicate_risk"] = strAny(topic["duplicate_risk"])
		}
		if len(avoid) == 0 {
			continue
		}
		out = append(out, avoid)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func originalScriptMemoryValue(memory map[string]any, artifactType string) map[string]any {
	wrapped, ok := memory[artifactType].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	value, ok := wrapped["value"].(map[string]any)
	if ok {
		return value
	}
	return map[string]any{}
}

func reviewAndMaybeRewriteOriginalScript(profile map[string]any, topic map[string]any, memory map[string]any, article string, options map[string]any, cardKey string, timeout time.Duration, outputAbs string, state *originalScriptRunState) (string, map[string]any, *cloud.AgentAbilityResponse, error) {
	reviewInput := map[string]any{
		"article":      article,
		"user_profile": profile,
		"profile":      profile,
		"topic":        topic,
	}
	reviewResp, err := cloud.ContentReview("final", reviewInput, options, cardKey, timeout)
	if err != nil {
		return "", nil, nil, output.ErrNetwork("final review failed: %v", err)
	}
	state.ReviewRounds = 1
	state.StageUsage["final_review_v1"] = reviewResp.Usage
	if err := writeJSONFile(filepath.Join(outputAbs, "05_final_review_v1.json"), reviewResp.Result); err != nil {
		return "", nil, nil, output.ErrSystem("write final review v1 failed: %v", err)
	}
	routeResp, err := cloud.ContentStage("final_review_route", map[string]any{"payload": map[string]any{
		"profile":      profile,
		"topic":        topic,
		"article":      article,
		"final_review": reviewResp.Result,
	}}, options, cardKey, timeout)
	if err != nil {
		state.Warnings = append(state.Warnings, fmt.Sprintf("final review route skipped: %v", err))
		return article, reviewResp.Result, nil, nil
	}
	state.StageUsage["final_review_route_v1"] = routeResp.Usage
	if err := writeJSONFile(filepath.Join(outputAbs, "06_review_route_v1.json"), routeResp.Result); err != nil {
		return "", nil, nil, output.ErrSystem("write review route failed: %v", err)
	}
	nextStep := strings.TrimSpace(strAny(routeResp.Result["next_step"]))
	if nextStep != "article_rewrite" {
		return article, reviewResp.Result, nil, nil
	}
	rewriteResp, err := cloud.ContentStage("article_rewrite", map[string]any{"payload": map[string]any{
		"profile":      profile,
		"topic":        topic,
		"memory":       memory,
		"article":      article,
		"final_review": reviewResp.Result,
		"route":        routeResp.Result,
	}}, options, cardKey, timeout)
	if err != nil {
		state.Warnings = append(state.Warnings, fmt.Sprintf("article rewrite skipped: %v", err))
		return article, reviewResp.Result, nil, nil
	}
	if err := writeJSONFile(filepath.Join(outputAbs, "07_article_v2.json"), rewriteResp.Result); err != nil {
		return "", nil, nil, output.ErrSystem("write article v2 json failed: %v", err)
	}
	rewritten := originalScriptArticleText(rewriteResp.Result)
	if strings.TrimSpace(rewritten) == "" {
		state.Warnings = append(state.Warnings, "article_rewrite returned empty article; kept v1")
		return article, reviewResp.Result, rewriteResp, nil
	}
	if err := os.WriteFile(filepath.Join(outputAbs, "07_article_v2.md"), []byte(rewritten), 0644); err != nil {
		return "", nil, nil, output.ErrSystem("write article v2 failed: %v", err)
	}
	review2Input := cloneMap(reviewInput)
	review2Input["article"] = rewritten
	review2Resp, err := cloud.ContentReview("final", review2Input, options, cardKey, timeout)
	if err != nil {
		state.Warnings = append(state.Warnings, fmt.Sprintf("final review v2 skipped: %v", err))
		return rewritten, reviewResp.Result, rewriteResp, nil
	}
	state.ReviewRounds = 2
	state.StageUsage["final_review_v2"] = review2Resp.Usage
	if err := writeJSONFile(filepath.Join(outputAbs, "08_final_review_v2.json"), review2Resp.Result); err != nil {
		return "", nil, nil, output.ErrSystem("write final review v2 failed: %v", err)
	}
	return rewritten, review2Resp.Result, rewriteResp, nil
}

func originalScriptArticleText(result map[string]any) string {
	for _, key := range []string{"article", "full_script", "revise_version", "script", "text"} {
		if text := strings.TrimSpace(strAny(result[key])); text != "" {
			return text
		}
	}
	return ""
}

func saveOriginalScriptCloudArtifacts(profileID, runID string, topic map[string]any, finalArticle string, finalReview map[string]any, cardKey string, state *originalScriptRunState) {
	artifacts := []struct {
		key          string
		artifactType string
		value        any
	}{
		{"topic_selection", "topic_selection", topic},
		{"final_article", "final_article", finalArticle},
		{"final_review", "final_review", finalReview},
	}
	for _, artifact := range artifacts {
		item, err := cloud.ContentArtifactSave(profileID, runID, artifact.artifactType, artifact.value, cardKey)
		if err != nil {
			state.Warnings = append(state.Warnings, fmt.Sprintf("cloud artifact %s save failed: %v", artifact.artifactType, err))
			continue
		}
		state.CloudAssets[artifact.key] = item.AssetID
	}
}

func updateOriginalScriptHistory(profileID, runID, topicHint string, topic map[string]any, finalArticle string, finalReview map[string]any, cardKey string, state *originalScriptRunState) {
	history := map[string]any{"items": []any{}}
	data, _, err := cloud.ContentMemoryRead(profileID, "content_history", cardKey)
	if err == nil {
		_ = json.Unmarshal(data, &history)
	}
	items, _ := history["items"].([]any)
	items = append([]any{map[string]any{
		"run_id":       runID,
		"topic_hint":   topicHint,
		"topic":        topic,
		"article_head": firstRunes(finalArticle, 160),
		"final_review": finalReview,
		"created_at":   time.Now().Format(time.RFC3339),
	}}, items...)
	if len(items) > 30 {
		items = items[:30]
	}
	history["items"] = items
	item, err := cloud.ContentMemorySave(profileID, "content_history", history, cardKey)
	if err != nil {
		state.Warnings = append(state.Warnings, fmt.Sprintf("content_history save failed: %v", err))
		return
	}
	state.CloudAssets["content_history"] = item.AssetID
}

func saveOriginalScriptState(outputAbs string, state originalScriptRunState) {
	_ = writeJSONFile(filepath.Join(outputAbs, "run_state.json"), state)
}

func firstRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}

func printContentOriginalScriptUsage() {
	fmt.Println("luma-cli content original-script <subcommand>")
	fmt.Println("  run --profile <profile_id> [--topic-hint <hint>] [--output runs/<run_id>]")
}
