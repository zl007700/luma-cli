package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/atom"
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
	state.CurrentStep = "research"
	saveOriginalScriptState(outputAbs, state)
	research := runOriginalScriptResearch(profileMap, topicHint, cfg.CardKey, &state)
	if err := writeJSONFile(filepath.Join(outputAbs, "03_research_rounds.json"), research); err != nil {
		return output.ErrSystem("write research failed: %v", err)
	}

	state.CurrentStep = "detail_expansion"
	saveOriginalScriptState(outputAbs, state)
	detailPlan, expandedDetails := runOriginalScriptDetailExpansion(research, options, cfg.CardKey, timeout, outputAbs, &state)
	if err := writeJSONFile(filepath.Join(outputAbs, "04_detail_expansion_plan.json"), detailPlan); err != nil {
		return output.ErrSystem("write detail expansion plan failed: %v", err)
	}
	if err := writeJSONFile(filepath.Join(outputAbs, "05_expanded_details.json"), expandedDetails); err != nil {
		return output.ErrSystem("write expanded details failed: %v", err)
	}

	state.CurrentStep = "topic_refine"
	saveOriginalScriptState(outputAbs, state)
	llmMemory := compactOriginalScriptMemory(memory, 8)
	llmResearch := compactOriginalScriptResearch(research, 18)
	llmExpandedDetails := compactOriginalScriptExpandedDetails(expandedDetails, 5000)
	if err := writeJSONFile(filepath.Join(outputAbs, "05_compact_context_topic.json"), map[string]any{"research": llmResearch, "expanded_details": llmExpandedDetails}); err != nil {
		return output.ErrSystem("write compact topic context failed: %v", err)
	}
	topicInput := map[string]any{
		"profile":             profileMap,
		"topic_hint":          topicHint,
		"memory":              llmMemory,
		"research":            llmResearch,
		"expanded_details":    llmExpandedDetails,
		"avoid_recent_topics": originalScriptAvoidRecent(memory, 8),
		"instruction":         "Select or refine one original video-script topic. Avoid repeating recent history even if topic_hint is similar; choose a meaningfully different angle, public entry, and concrete scene.",
	}
	topicResp, err := cloud.ContentStage("topic_refine", map[string]any{"payload": topicInput}, options, cfg.CardKey, timeout)
	if err != nil {
		return output.ErrNetwork("topic refine failed: %v", err)
	}
	state.StageUsage["topic_refine"] = topicResp.Usage
	if err := writeJSONFile(filepath.Join(outputAbs, "06_topic_selection.json"), topicResp.Result); err != nil {
		return output.ErrSystem("write topic selection failed: %v", err)
	}
	topicResult := topicResp.Result
	writableTopic := resolvedOriginalScriptTopic(topicResult)
	if writableTopic == nil {
		retryInput := cloneMap(topicInput)
		retryInput["previous_topic_result"] = topicResult
		retryInput["instruction"] = "The previous topic result rejected the hint but did not produce a writable selected_topic. Convert the rejection/revision_direction into exactly one concrete writable topic. Return selected_topic with public_entry, concrete_scene, core_conflict, thesis, audience_reward, and evidence_to_use. Do not write the article."
		retryResp, err := cloud.ContentStage("topic_refine", map[string]any{"payload": retryInput}, options, cfg.CardKey, timeout)
		if err != nil {
			return output.ErrNetwork("topic refine retry failed: %v", err)
		}
		state.StageUsage["topic_refine_retry"] = retryResp.Usage
		topicResult = retryResp.Result
		if err := writeJSONFile(filepath.Join(outputAbs, "06_topic_selection_retry.json"), topicResult); err != nil {
			return output.ErrSystem("write topic selection retry failed: %v", err)
		}
		writableTopic = resolvedOriginalScriptTopic(topicResult)
	}
	if writableTopic == nil {
		state.Status = "needs_topic"
		state.CurrentStep = "topic_refine"
		state.Warnings = append(state.Warnings, "topic_refine did not produce a writable selected_topic; article_write was blocked")
		saveOriginalScriptState(outputAbs, state)
		return output.ErrSystem("topic refine did not produce a writable selected_topic")
	}

	state.CurrentStep = "article_write"
	saveOriginalScriptState(outputAbs, state)
	writeMemory := compactOriginalScriptMemory(memory, 5)
	writeResearch := compactOriginalScriptResearch(research, 10)
	writeExpandedDetails := compactOriginalScriptExpandedDetails(expandedDetails, 2500)
	if err := writeJSONFile(filepath.Join(outputAbs, "06_compact_context_write.json"), map[string]any{"research": writeResearch, "expanded_details": writeExpandedDetails}); err != nil {
		return output.ErrSystem("write compact write context failed: %v", err)
	}
	writeInput := map[string]any{
		"profile":          profileMap,
		"topic":            writableTopic,
		"topic_hint":       topicHint,
		"memory":           writeMemory,
		"research":         writeResearch,
		"expanded_details": writeExpandedDetails,
		"material_context": map[string]any{
			"research":         writeResearch,
			"expanded_details": writeExpandedDetails,
			"rule":             "Use research and expanded_details as evidence boundaries. Do not invent facts beyond provided context.",
		},
	}
	writeResp, err := cloud.ContentStage("article_write", map[string]any{"payload": writeInput}, options, cfg.CardKey, timeout)
	if err != nil {
		state.Status = "failed"
		state.CurrentStep = "article_write"
		state.Warnings = append(state.Warnings, fmt.Sprintf("article_write failed: %v", err))
		saveOriginalScriptState(outputAbs, state)
		return output.ErrNetwork("article write failed: %v", err)
	}
	state.StageUsage["article_write"] = writeResp.Usage
	if err := writeJSONFile(filepath.Join(outputAbs, "07_article_v1.json"), writeResp.Result); err != nil {
		return output.ErrSystem("write article json failed: %v", err)
	}
	article := originalScriptArticleText(writeResp.Result)
	if strings.TrimSpace(article) == "" {
		return output.ErrSystem("article_write returned empty article")
	}
	if err := os.WriteFile(filepath.Join(outputAbs, "07_article_v1.md"), []byte(article), 0644); err != nil {
		return output.ErrSystem("write article draft failed: %v", err)
	}

	finalArticle, finalReview, rewriteResp, err := reviewAndMaybeRewriteOriginalScript(
		profileMap,
		writableTopic,
		writeMemory,
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
	passed, blockReason := originalScriptFinalReviewPassed(finalReview)
	if passed {
		saveOriginalScriptCloudArtifacts(profileID, runID, writableTopic, finalArticle, finalReview, cfg.CardKey, &state)
		updateOriginalScriptHistory(profileID, runID, topicHint, writableTopic, finalArticle, finalReview, cfg.CardKey, &state)
		state.Status = "done"
	} else {
		saveOriginalScriptReviewArtifact(profileID, runID, "needs_revision_article", finalArticle, cfg.CardKey, &state)
		saveOriginalScriptReviewArtifact(profileID, runID, "needs_revision_review", finalReview, cfg.CardKey, &state)
		state.Status = "needs_revision"
		state.Warnings = append(state.Warnings, fmt.Sprintf("final review did not pass promotion gate: %s; final was saved locally but not promoted to content_history", blockReason))
	}
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

func runOriginalScriptResearch(profile map[string]any, topicHint, cardKey string, state *originalScriptRunState) map[string]any {
	webQueries := originalScriptWebQueries(profile, topicHint)
	socialKeywords := originalScriptSocialKeywords(topicHint)
	result := map[string]any{
		"web_queries":     webQueries,
		"social_keywords": socialKeywords,
		"started_at":      time.Now().Format(time.RFC3339),
	}
	if len(webQueries) > 0 {
		webResult, err := cloud.SearchWebsearch(cloud.WebSearchRequest{
			Queries:   webQueries,
			DateRange: "7d",
			Num:       5,
		}, cardKey)
		if err != nil {
			state.Warnings = append(state.Warnings, fmt.Sprintf("websearch failed: %v", err))
		} else {
			result["websearch"] = webResult
			result["web_sources"] = originalScriptSignals(webResult, "web")
		}
	}
	if len(socialKeywords) > 0 {
		socialResult, err := cloud.SearchSocial(cloud.SocialSearchRequest{
			Platform:        "douyin",
			Keywords:        socialKeywords,
			DateRange:       "7d",
			LimitPerKeyword: 5,
		}, cardKey)
		if err != nil {
			state.Warnings = append(state.Warnings, fmt.Sprintf("social search failed: %v", err))
		} else {
			result["social"] = socialResult
			result["social_sources"] = originalScriptSignals(socialResult, "social")
		}
	}
	result["finished_at"] = time.Now().Format(time.RFC3339)
	return result
}

func runOriginalScriptDetailExpansion(research map[string]any, options map[string]any, cardKey string, timeout time.Duration, outputAbs string, state *originalScriptRunState) (map[string]any, map[string]any) {
	sources := []map[string]any{}
	sources = append(sources, originalScriptSourceList(research["web_sources"])...)
	sources = append(sources, originalScriptSourceList(research["social_sources"])...)
	if len(sources) == 0 {
		state.Warnings = append(state.Warnings, "detail expansion skipped: no research sources")
		return map[string]any{"selected": []any{}, "skipped": []any{}, "reason": "no research sources"}, map[string]any{"items": []any{}}
	}
	plannerInput := map[string]any{
		"research_goal": "Select high-value sources for original video script evidence expansion.",
		"budget":        map[string]any{"max_items": 3, "max_url_reads": 2, "max_video_asr": 1},
		"sources":       compactOriginalScriptSources(sources, 12),
	}
	if err := writeJSONFile(filepath.Join(outputAbs, "04_detail_expansion_input.json"), plannerInput); err != nil {
		state.Warnings = append(state.Warnings, fmt.Sprintf("detail expansion input save failed: %v", err))
	}
	planResp, err := cloud.ContentStage("detail_expansion_plan", map[string]any{"payload": plannerInput}, options, cardKey, timeout)
	var planResult map[string]any
	if err != nil {
		state.Warnings = append(state.Warnings, fmt.Sprintf("detail expansion plan failed: %v", err))
		planResult = fallbackOriginalScriptDetailPlan(sources, err)
	} else {
		state.StageUsage["detail_expansion_plan"] = planResp.Usage
		planResult = planResp.Result
	}
	expanded := map[string]any{"items": []any{}}
	selected := listMapFromAny(anyListFromPayload(planResult, "selected"))
	items := []any{}
	tried := map[string]bool{}
	urlReadAttempts := 0
	videoASRAttempts := 0
	successes := 0
	for index, selectedItem := range selected {
		if index >= 3 {
			break
		}
		sourceID := strings.TrimSpace(strAny(selectedItem["id"]))
		tried[sourceID] = true
		source := originalScriptFindSource(sources, sourceID)
		if source == nil {
			items = append(items, map[string]any{"id": sourceID, "status": "skipped", "reason": "source not found"})
			continue
		}
		action := strings.TrimSpace(strAny(selectedItem["action"]))
		if action == "" {
			action = originalScriptDefaultExpansionAction(source)
		}
		var item map[string]any
		switch action {
		case "url_read":
			urlReadAttempts++
			item = originalScriptExpandURL(source, cardKey)
		case "download_asr":
			videoASRAttempts++
			item = originalScriptExpandVideoASR(source, cardKey, outputAbs)
		default:
			item = map[string]any{"id": sourceID, "status": "skipped", "reason": "unsupported action: " + action, "source": source}
		}
		if originalScriptExpansionSucceeded(item) {
			successes++
		}
		items = append(items, item)
	}
	if successes == 0 {
		fallbackItems := runOriginalScriptExpansionFallback(sources, tried, cardKey, outputAbs, &urlReadAttempts, &videoASRAttempts)
		if len(fallbackItems) > 0 {
			planResult["runtime_fallback"] = map[string]any{
				"reason":             "selected expansions produced no usable content",
				"max_url_read_tries": 4,
				"max_video_asr":      1,
			}
			items = append(items, fallbackItems...)
		}
	}
	expanded["items"] = items
	return planResult, expanded
}

func runOriginalScriptExpansionFallback(sources []map[string]any, tried map[string]bool, cardKey, outputAbs string, urlReadAttempts, videoASRAttempts *int) []any {
	items := []any{}
	for _, source := range sources {
		if *urlReadAttempts >= 4 || len(items) >= 2 {
			break
		}
		id := strAny(source["id"])
		if tried[id] || originalScriptDefaultExpansionAction(source) != "url_read" || !likelyReadableOriginalScriptURL(strAny(source["url"])) {
			continue
		}
		tried[id] = true
		*urlReadAttempts++
		item := originalScriptExpandURL(source, cardKey)
		item["runtime_fallback"] = true
		items = append(items, item)
		if originalScriptExpansionSucceeded(item) {
			break
		}
	}
	if *videoASRAttempts < 1 {
		for _, source := range sources {
			id := strAny(source["id"])
			if tried[id] || originalScriptDefaultExpansionAction(source) != "download_asr" {
				continue
			}
			tried[id] = true
			*videoASRAttempts++
			item := originalScriptExpandVideoASR(source, cardKey, outputAbs)
			item["runtime_fallback"] = true
			items = append(items, item)
			break
		}
	}
	return items
}

func fallbackOriginalScriptDetailPlan(sources []map[string]any, cause error) map[string]any {
	selected := []any{}
	urlReads := 0
	videoASR := 0
	for _, source := range sources {
		if len(selected) >= 3 {
			break
		}
		action := originalScriptDefaultExpansionAction(source)
		if action == "url_read" {
			if urlReads >= 2 || strings.TrimSpace(strAny(source["url"])) == "" {
				continue
			}
			urlReads++
		} else if action == "download_asr" {
			if videoASR >= 1 {
				continue
			}
			videoASR++
		} else {
			continue
		}
		selected = append(selected, map[string]any{
			"id":     source["id"],
			"action": action,
			"reason": "fallback selected after detail planner failed",
		})
	}
	errorText := ""
	if cause != nil {
		errorText = cause.Error()
	}
	return map[string]any{
		"selected":         selected,
		"fallback":         true,
		"fallback_reason":  "detail_expansion_plan_failed",
		"planner_error":    errorText,
		"budget_enforced":  map[string]any{"max_items": 3, "max_url_reads": 2, "max_video_asr": 1},
		"selection_policy": "prefer up to two readable web sources plus one Douyin source for ASR",
	}
}

func originalScriptWebQueries(profile map[string]any, topicHint string) []string {
	queries := []string{}
	if strings.TrimSpace(topicHint) != "" {
		queries = append(queries, strings.TrimSpace(topicHint))
	}
	identity := strings.TrimSpace(strAny(profile["identity"]))
	if identity != "" {
		queries = append(queries, identity+" AI工具 工作流 案例")
	}
	queries = append(queries,
		"AI Agent 自动跟进 客户 询盘 工作流",
		"AI工具 工作流 自动化 中小企业 案例",
	)
	return dedupeStringsLimit(queries, 4)
}

func originalScriptSocialKeywords(topicHint string) []string {
	keywords := []string{"AI工具", "AI工作流"}
	for _, item := range splitContentList(topicHint) {
		if strings.Contains(item, "AI") || strings.Contains(item, "工具") || strings.Contains(item, "工作流") {
			keywords = append(keywords, item)
		}
	}
	return dedupeStringsLimit(keywords, 3)
}

func originalScriptSignals(result map[string]any, sourceType string) []map[string]any {
	rawSignals := listMapFromAny(anyListFromPayload(result, "raw_signals"))
	out := make([]map[string]any, 0, len(rawSignals))
	for index, signal := range rawSignals {
		id := firstNonEmpty(strAny(signal["signal_id"]), fmt.Sprintf("%s_%d", sourceType, index+1))
		raw := mapFromAny(signal["raw"])
		url := firstNonEmpty(strAny(signal["url"]), strAny(signal["link"]), strAny(raw["url"]), strAny(raw["link"]))
		out = append(out, map[string]any{
			"id":             id,
			"source_type":    sourceType,
			"title":          firstNonEmpty(strAny(signal["title"]), strAny(raw["title"])),
			"summary":        firstNonEmpty(strAny(signal["summary"]), strAny(raw["summary"]), strAny(raw["note"])),
			"url":            url,
			"published_at":   firstNonEmpty(strAny(signal["published_at"]), strAny(raw["publish_time"]), strAny(raw["date"])),
			"author_name":    firstNonEmpty(strAny(signal["author_name"]), strAny(raw["author"])),
			"stats":          signal["stats"],
			"matched_query":  firstNonEmpty(strAny(signal["matched_query"]), strAny(signal["matched_keyword"])),
			"raw_signal":     signal,
			"expansion_hint": originalScriptDefaultExpansionAction(map[string]any{"source_type": sourceType, "url": url}),
		})
	}
	return out
}

func originalScriptSourceList(value any) []map[string]any {
	if items, ok := value.([]map[string]any); ok {
		return items
	}
	if anyItems, ok := value.([]any); ok {
		return listMapFromAny(anyItems)
	}
	return nil
}

func originalScriptFindSource(sources []map[string]any, id string) map[string]any {
	for _, source := range sources {
		if strAny(source["id"]) == id {
			return source
		}
	}
	return nil
}

func resolvedOriginalScriptTopic(result map[string]any) map[string]any {
	selected := mapFromAny(result["selected_topic"])
	if len(selected) > 0 {
		out := cloneMap(selected)
		out["topic_review"] = map[string]any{
			"decision":           result["decision"],
			"duplicate_risk":     result["duplicate_risk"],
			"reason":             result["reason"],
			"research_gap":       result["research_gap"],
			"revision_direction": result["revision_direction"],
		}
		return out
	}
	if _, hasSelectedTopic := result["selected_topic"]; hasSelectedTopic {
		decision := strings.TrimSpace(strAny(result["decision"]))
		if decision == "reject" || decision == "rerun_research" || decision == "revise" {
			return nil
		}
	}
	if strings.TrimSpace(strAny(result["public_entry"])) != "" ||
		strings.TrimSpace(strAny(result["concrete_scene"])) != "" ||
		strings.TrimSpace(strAny(result["thesis"])) != "" {
		return result
	}
	return nil
}

func originalScriptDefaultExpansionAction(source map[string]any) string {
	sourceType := strings.TrimSpace(strAny(source["source_type"]))
	url := strings.TrimSpace(strAny(source["url"]))
	if sourceType == "web" && url != "" {
		return "url_read"
	}
	if strings.Contains(url, "douyin.com") {
		return "download_asr"
	}
	return "url_read"
}

func originalScriptExpansionSucceeded(item map[string]any) bool {
	return strings.TrimSpace(strAny(item["status"])) == "expanded"
}

func likelyReadableOriginalScriptURL(rawURL string) bool {
	value := strings.ToLower(strings.TrimSpace(rawURL))
	if value == "" {
		return false
	}
	blockedHosts := []string{
		"instagram.com",
		"x.com/",
		"twitter.com",
		"zhihu.com",
		"reddit.com",
	}
	for _, host := range blockedHosts {
		if strings.Contains(value, host) {
			return false
		}
	}
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func originalScriptExpandURL(source map[string]any, cardKey string) map[string]any {
	url := strings.TrimSpace(strAny(source["url"]))
	if url == "" {
		return map[string]any{"id": source["id"], "status": "skipped", "reason": "missing url", "source": source}
	}
	result, err := cloud.URLRead(cloud.URLReadRequest{URL: url, MaxChars: 6000}, cardKey)
	if err != nil {
		return map[string]any{"id": source["id"], "status": "failed", "action": "url_read", "error": err.Error(), "source": source}
	}
	return map[string]any{"id": source["id"], "status": "expanded", "action": "url_read", "source": source, "content": result}
}

func originalScriptExpandVideoASR(source map[string]any, cardKey, outputAbs string) map[string]any {
	url := strings.TrimSpace(strAny(source["url"]))
	if url == "" || !strings.Contains(url, "douyin.com") {
		return map[string]any{"id": source["id"], "status": "skipped", "action": "download_asr", "reason": "missing douyin url", "source": source}
	}
	videoDir := filepath.Join(outputAbs, "video_asr")
	_ = os.MkdirAll(videoDir, 0755)
	videoPath := filepath.Join(videoDir, strings.TrimSpace(strAny(source["id"]))+".mp4")
	downloaded, err := atom.DownloadSocialVideo(url, videoPath, cardKey)
	if err != nil {
		return map[string]any{"id": source["id"], "status": "failed", "action": "download_asr", "stage": "download", "error": err.Error(), "source": source}
	}
	asr, err := atom.RunASR(atom.ASROptions{FilePath: downloaded.VideoPath, Language: "zh", CardKey: cardKey, TimeoutSec: 300})
	if err != nil {
		return map[string]any{"id": source["id"], "status": "failed", "action": "download_asr", "stage": "asr", "error": err.Error(), "video_path": downloaded.VideoPath, "source": source}
	}
	return map[string]any{"id": source["id"], "status": "expanded", "action": "download_asr", "source": source, "video_path": downloaded.VideoPath, "asr_text": asr.Text, "segments": asr.Segments}
}

func compactOriginalScriptMemory(memory map[string]any, maxHistory int) map[string]any {
	if maxHistory <= 0 {
		maxHistory = 5
	}
	out := map[string]any{}
	if history := originalScriptMemoryValue(memory, "content_history"); len(history) > 0 {
		items := []any{}
		for _, raw := range anyListFromPayload(history, "items") {
			item := mapFromAny(raw)
			if len(item) == 0 {
				continue
			}
			topic := mapFromAny(item["topic"])
			items = append(items, map[string]any{
				"run_id":       item["run_id"],
				"created_at":   item["created_at"],
				"topic_hint":   firstRunes(strAny(item["topic_hint"]), 120),
				"article_head": firstRunes(strAny(item["article_head"]), 220),
				"topic": map[string]any{
					"title":          firstRunes(strAny(topic["title"]), 120),
					"public_entry":   firstRunes(strAny(topic["public_entry"]), 160),
					"concrete_scene": firstRunes(strAny(firstNonEmpty(strAny(topic["concrete_scene"]), strAny(topic["scene"]))), 160),
					"thesis":         firstRunes(strAny(topic["thesis"]), 180),
				},
				"final_review": compactOriginalScriptReview(mapFromAny(item["final_review"])),
			})
			if len(items) >= maxHistory {
				break
			}
		}
		out["content_history"] = map[string]any{"items": items}
	}
	for _, key := range []string{"evergreen_topics", "topic_bank"} {
		value := originalScriptMemoryValue(memory, key)
		if len(value) == 0 {
			continue
		}
		items := []any{}
		for index, raw := range anyListFromPayload(value, "items") {
			if index >= 20 {
				break
			}
			items = append(items, raw)
		}
		out[key] = map[string]any{
			"schema_version": value["schema_version"],
			"note":           firstRunes(strAny(value["note"]), 160),
			"items":          items,
		}
	}
	return out
}

func compactOriginalScriptReview(review map[string]any) map[string]any {
	if len(review) == 0 {
		return map[string]any{}
	}
	return map[string]any{
		"decision":    review["decision"],
		"total_score": review["total_score"],
		"summary":     firstRunes(firstNonEmpty(strAny(review["summary"]), strAny(review["review_text"])), 220),
	}
}

func compactOriginalScriptResearch(research map[string]any, maxSources int) map[string]any {
	if maxSources <= 0 {
		maxSources = 12
	}
	web := compactOriginalScriptSources(originalScriptSourceList(research["web_sources"]), maxSources/2)
	social := compactOriginalScriptSources(originalScriptSourceList(research["social_sources"]), maxSources-len(web))
	return map[string]any{
		"web_queries":     research["web_queries"],
		"social_keywords": research["social_keywords"],
		"web_sources":     web,
		"social_sources":  social,
		"source_counts": map[string]any{
			"web":    len(originalScriptSourceList(research["web_sources"])),
			"social": len(originalScriptSourceList(research["social_sources"])),
		},
	}
}

func compactOriginalScriptSources(sources []map[string]any, limit int) []map[string]any {
	if limit <= 0 || len(sources) == 0 {
		return nil
	}
	out := []map[string]any{}
	for _, source := range sources {
		out = append(out, map[string]any{
			"id":            source["id"],
			"source_type":   source["source_type"],
			"title":         firstRunes(strAny(source["title"]), 120),
			"summary":       firstRunes(strAny(source["summary"]), 240),
			"url":           source["url"],
			"published_at":  source["published_at"],
			"author_name":   source["author_name"],
			"stats":         source["stats"],
			"matched_query": source["matched_query"],
		})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func compactOriginalScriptExpandedDetails(expanded map[string]any, maxContentChars int) map[string]any {
	if maxContentChars <= 0 {
		maxContentChars = 6000
	}
	items := []map[string]any{}
	rawItems, _ := expanded["items"].([]any)
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		compact := map[string]any{
			"id":     item["id"],
			"status": item["status"],
			"action": item["action"],
			"source": compactOriginalScriptSource(mapFromAny(item["source"])),
		}
		if content := mapFromAny(item["content"]); len(content) > 0 {
			compact["content"] = compactURLReadContent(content, maxContentChars)
		}
		if asr := strings.TrimSpace(strAny(item["asr_text"])); asr != "" {
			compact["asr_text"] = firstRunes(asr, maxContentChars)
		}
		if errText := strAny(item["error"]); errText != "" {
			compact["error"] = errText
		}
		if reason := strAny(item["reason"]); reason != "" {
			compact["reason"] = reason
		}
		items = append(items, compact)
	}
	return map[string]any{"items": items}
}

func compactOriginalScriptSource(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	return map[string]any{
		"id":          source["id"],
		"source_type": source["source_type"],
		"title":       firstRunes(strAny(source["title"]), 120),
		"summary":     firstRunes(strAny(source["summary"]), 240),
		"url":         source["url"],
	}
}

func compactURLReadContent(content map[string]any, maxChars int) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"url", "title", "provider", "source", "status"} {
		if value := content[key]; value != nil {
			out[key] = value
		}
	}
	for _, key := range []string{"content", "text", "markdown"} {
		if text := strings.TrimSpace(strAny(content[key])); text != "" {
			out[key] = firstRunes(text, maxChars)
			break
		}
	}
	if chars := content["content_chars"]; chars != nil {
		out["content_chars"] = chars
	}
	return out
}

func dedupeStringsLimit(values []string, limit int) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		clean := strings.TrimSpace(value)
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		out = append(out, clean)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
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
	reviewResp.Result["review_route"] = routeResp.Result
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

func originalScriptFinalReviewPassed(review map[string]any) (bool, string) {
	decision := strings.ToLower(strings.TrimSpace(strAny(review["decision"])))
	score := floatFromAny(review["total_score"])
	credibility := floatFromAny(mapFromAny(mapFromAny(review["scores"])["credibility"])["score"])
	route := mapFromAny(review["review_route"])
	routeNext := strings.ToLower(strings.TrimSpace(strAny(route["next_step"])))
	reviewText := strings.ToLower(strAny(review["review_text"]) + " " + strAny(review["summary"]) + " " + strAny(route["reason"]) + " " + strAny(route["minimum_fix"]))

	if routeNext != "" && routeNext != "finish" {
		return false, fmt.Sprintf("review route requested %s", routeNext)
	}
	if !strings.HasPrefix(decision, "pass") {
		return false, fmt.Sprintf("decision=%s", decision)
	}
	if score < 8 {
		return false, fmt.Sprintf("score=%.1f < 8.0", score)
	}
	if credibility > 0 && credibility < 7 {
		return false, fmt.Sprintf("credibility=%.1f < 7.0", credibility)
	}
	if strings.Contains(reviewText, "fact_check") || strings.Contains(reviewText, "事实核查") || strings.Contains(reviewText, "查证") {
		return false, "review requested fact check"
	}
	return true, "pass"
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

func saveOriginalScriptReviewArtifact(profileID, runID, artifactType string, value any, cardKey string, state *originalScriptRunState) {
	item, err := cloud.ContentArtifactSave(profileID, runID, artifactType, value, cardKey)
	if err != nil {
		state.Warnings = append(state.Warnings, fmt.Sprintf("cloud artifact %s save failed: %v", artifactType, err))
		return
	}
	state.CloudAssets[artifactType] = item.AssetID
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
		"final_review": compactOriginalScriptReview(finalReview),
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
