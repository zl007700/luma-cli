package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

type contentTopicMineResult struct {
	ProfileID         string                    `json:"profile_id,omitempty"`
	DateRange         string                    `json:"date_range"`
	SocialKeywords    []string                  `json:"social_keywords,omitempty"`
	WebQueries        []string                  `json:"web_queries,omitempty"`
	BenchmarkSource   *contentBenchmarkSource   `json:"benchmark_source,omitempty"`
	BenchmarkAccounts []contentBenchmarkAccount `json:"benchmark_accounts,omitempty"`
	SignalGroups      contentSignalGroups       `json:"signal_groups,omitempty"`
	EvergreenThemes   []contentEvergreenTheme   `json:"evergreen_themes,omitempty"`
	RawSignals        []map[string]any          `json:"raw_signals"`
	Counts            contentTopicMineCounts    `json:"counts"`
	Notes             []string                  `json:"notes,omitempty"`
	GeneratedAt       time.Time                 `json:"generated_at"`
}

type contentTopicMineCounts struct {
	SocialRaw        int `json:"social_raw"`
	SocialAccountRaw int `json:"social_account_raw"`
	WebRaw           int `json:"web_raw"`
	EvergreenThemes  int `json:"evergreen_themes"`
	MergedRaw        int `json:"merged_raw"`
	DroppedDupes     int `json:"dropped_dupes"`
	DroppedEmpty     int `json:"dropped_empty"`
	TruncatedFrom    int `json:"truncated_from,omitempty"`
}

type contentBenchmarkSource struct {
	ProfileID string    `json:"profile_id"`
	AssetID   string    `json:"asset_id,omitempty"`
	Kind      string    `json:"kind,omitempty"`
	Name      string    `json:"name,omitempty"`
	ObjectKey string    `json:"object_key,omitempty"`
	LocalPath string    `json:"local_path,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type contentBenchmarkAccount struct {
	AccountInput string  `json:"account_input"`
	AccountID    string  `json:"account_id,omitempty"`
	SecUserID    string  `json:"sec_user_id,omitempty"`
	UniqueID     string  `json:"unique_id,omitempty"`
	Nickname     string  `json:"nickname,omitempty"`
	Tier         string  `json:"tier,omitempty"`
	Score        float64 `json:"score,omitempty"`
}

type contentSignalGroups struct {
	HotSignals       []map[string]any `json:"hot_signals,omitempty"`
	BenchmarkSignals []map[string]any `json:"benchmark_signals,omitempty"`
	WebSignals       []map[string]any `json:"web_signals,omitempty"`
}

type contentEvergreenTheme struct {
	ThemeID         string   `json:"theme_id"`
	Theme           string   `json:"theme"`
	Audience        string   `json:"audience,omitempty"`
	Pain            string   `json:"pain,omitempty"`
	Opinion         string   `json:"opinion,omitempty"`
	Keywords        []string `json:"keywords,omitempty"`
	Score           float64  `json:"score"`
	EvidenceCount   int      `json:"evidence_count"`
	SourceSignalIDs []string `json:"source_signal_ids,omitempty"`
	SourceTitles    []string `json:"source_titles,omitempty"`
	LastSeenAt      string   `json:"last_seen_at,omitempty"`
}

type contentTopicReviewResult struct {
	ProfileID        string             `json:"profile_id,omitempty"`
	InputPath        string             `json:"input_path,omitempty"`
	TargetDuration   string             `json:"target_duration,omitempty"`
	ProfileFormatFit contentFormatFit   `json:"profile_format_fit,omitempty"`
	TopicCards       []contentTopicCard `json:"topic_cards"`
	Counts           contentReviewCount `json:"counts"`
	GeneratedAt      time.Time          `json:"generated_at"`
}

type contentReviewCount struct {
	RawSignals       int `json:"raw_signals"`
	EvergreenThemes  int `json:"evergreen_themes"`
	GeneratedCards   int `json:"generated_cards"`
	RejectedSignals  int `json:"rejected_signals"`
	SelectedTopCards int `json:"selected_top_cards"`
}

type contentTopicCard struct {
	TopicID                 string                 `json:"topic_id"`
	Title                   string                 `json:"title"`
	Angle                   string                 `json:"angle"`
	CoreOpinion             string                 `json:"core_opinion"`
	CommonMisunderstand     string                 `json:"common_misunderstanding,omitempty"`
	AudienceValue           string                 `json:"audience_value,omitempty"`
	ObjectiveFacts          []string               `json:"objective_facts,omitempty"`
	PublicEntry             string                 `json:"public_entry,omitempty"`
	WhyGeneralAudienceCares string                 `json:"why_general_audience_cares,omitempty"`
	AudienceFilterTurn      string                 `json:"audience_filter_turn,omitempty"`
	ReviewThinking          string                 `json:"review_thinking,omitempty"`
	SourceType              string                 `json:"source_type"`
	ThemeID                 string                 `json:"theme_id,omitempty"`
	Theme                   string                 `json:"theme,omitempty"`
	EvidenceSignals         []contentEvidenceBrief `json:"evidence_signals,omitempty"`
	MaterialHypothesis      []string               `json:"material_hypothesis,omitempty"`
	Risks                   []string               `json:"risks,omitempty"`
	FormatFit               contentFormatFit       `json:"format_fit"`
	LongformPlan            *contentLongformPlan   `json:"longform_plan,omitempty"`
	Scores                  contentTopicScores     `json:"scores"`
	TotalScore              float64                `json:"total_score"`
	Status                  string                 `json:"status"`
	Recommendation          string                 `json:"recommendation"`
	Debug                   map[string]any         `json:"debug,omitempty"`
}

type contentEvidenceBrief struct {
	Source      string  `json:"source,omitempty"`
	Title       string  `json:"title,omitempty"`
	AuthorName  string  `json:"author_name,omitempty"`
	URL         string  `json:"url,omitempty"`
	Likes       float64 `json:"likes,omitempty"`
	PublishedAt string  `json:"published_at,omitempty"`
}

type contentTopicScores struct {
	Freshness            float64 `json:"freshness"`
	AudienceRelevance    float64 `json:"audience_relevance"`
	ProfileFit           float64 `json:"profile_fit"`
	Conflict             float64 `json:"conflict"`
	MaterialAvailability float64 `json:"material_availability"`
	ConversionPotential  float64 `json:"conversion_potential"`
}

type contentFormatFit struct {
	Recommended    string   `json:"recommended,omitempty"`
	DurationSec    int      `json:"duration_sec,omitempty"`
	Confidence     float64  `json:"confidence,omitempty"`
	Fallback       string   `json:"fallback,omitempty"`
	ProfileDefault string   `json:"profile_default,omitempty"`
	TopicCapacity  string   `json:"topic_capacity,omitempty"`
	Reason         []string `json:"reason,omitempty"`
}

type contentLongformPlan struct {
	Suitable               bool                     `json:"suitable"`
	DurationTargetSec      int                      `json:"duration_target_sec,omitempty"`
	CoreThesis             string                   `json:"core_thesis,omitempty"`
	ChapterCount           int                      `json:"chapter_count,omitempty"`
	EstimatedMaterialCount int                      `json:"estimated_material_count,omitempty"`
	LogicChain             []contentLongformSection `json:"logic_chain,omitempty"`
	WhySuitable            []string                 `json:"why_suitable,omitempty"`
	Risks                  []string                 `json:"risks,omitempty"`
}

type contentLongformSection struct {
	Section    string   `json:"section"`
	Claim      string   `json:"claim"`
	Evidence   []string `json:"evidence,omitempty"`
	VisualNeed string   `json:"visual_need,omitempty"`
}

func cmdContent(args []string) error {
	if len(args) < 1 {
		printContentUsage()
		return nil
	}
	switch args[0] {
	case "search":
		return cmdContentSearch(args[1:])
	case "topic":
		return cmdContentTopic(args[1:])
	case "history":
		return cmdContentHistory(args[1:])
	case "artifact":
		return cmdContentArtifact(args[1:])
	default:
		printContentUsage()
		return nil
	}
}

func cmdContentSearch(args []string) error {
	if len(args) < 1 {
		printContentSearchUsage()
		return nil
	}
	switch args[0] {
	case "social":
		return cmdContentSearchSocial(args[1:])
	case "social-account", "account":
		return cmdContentSearchSocialAccount(args[1:])
	case "websearch":
		return cmdContentSearchWebsearch(args[1:])
	case "image", "image-search":
		return cmdContentSearchImage(args[1:])
	default:
		printContentSearchUsage()
		return nil
	}
}

func cmdContentTopic(args []string) error {
	if len(args) < 1 {
		printContentTopicUsage()
		return nil
	}
	switch args[0] {
	case "mine":
		return cmdContentTopicMine(args[1:])
	case "review":
		return cmdContentTopicReview(args[1:])
	default:
		printContentTopicUsage()
		return nil
	}
}

func cmdContentArtifact(args []string) error {
	if len(args) < 1 {
		printContentArtifactUsage()
		return nil
	}
	switch args[0] {
	case "upload":
		return cmdContentArtifactUpload(args[1:])
	default:
		printContentArtifactUsage()
		return nil
	}
}

func cmdContentArtifactUpload(raw []string) error {
	args := cmdutil.Parse(raw)
	inputPath := strings.TrimSpace(args.String("input", args.Pos(0)))
	if inputPath == "" {
		return output.ErrValidation("--input is required")
	}
	artifactType := strings.TrimSpace(args.String("type", args.String("artifact-type", "")))
	if artifactType == "" {
		return output.ErrValidation("--type is required")
	}
	profileID := contentProfileID(args, "")
	if profileID == "" {
		return output.ErrValidation("--profile is required or run luma-cli profile use <id>")
	}
	absInput, err := absoluteOutputPath(inputPath)
	if err != nil {
		return output.ErrValidation("bad input path: %v", err)
	}
	resourceName := strings.TrimSpace(args.String("name", artifactType+".current.json"))
	extraMeta := map[string]any{}
	if value := strings.TrimSpace(args.String("topic-id", "")); value != "" {
		extraMeta["topic_id"] = value
	}
	if value := strings.TrimSpace(args.String("topic-title", args.String("title", ""))); value != "" {
		extraMeta["topic_title"] = value
	}
	if value := strings.TrimSpace(args.String("content-fingerprint", "")); value != "" {
		extraMeta["content_fingerprint"] = value
	}
	cloudArtifact, err := uploadContentJSONArtifact(absInput, nil, profileID, artifactType, resourceName, "content.artifact.upload", extraMeta)
	if err != nil {
		return output.ErrNetwork("artifact upload failed: %v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: cloudArtifact})
		return nil
	}
	fmt.Printf("Uploaded content artifact: %s\n", artifactType)
	fmt.Printf("Group: %s\n", cloudArtifact["group_name"])
	fmt.Printf("Object key: %s\n", cloudArtifact["object_key"])
	return nil
}

func cmdContentSearchSocial(raw []string) error {
	args := cmdutil.Parse(raw)
	keywords := splitContentList(args.String("keywords", args.Pos(0)))
	if len(keywords) == 0 {
		return output.ErrValidation("--keywords is required")
	}
	limit, err := args.Int("limit-per-keyword", 10)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	cfg, err := requireConfig()
	if err != nil {
		return output.ErrAuth("%v", err)
	}
	result, err := cloud.SearchSocial(cloud.SocialSearchRequest{
		Platform:        strings.TrimSpace(args.String("platform", "douyin")),
		Keywords:        keywords,
		DateRange:       strings.TrimSpace(args.String("date-range", "7d")),
		LimitPerKeyword: limit,
	}, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("social search failed: %v", err)
	}
	return saveContentSearchResult(result, strings.TrimSpace(args.String("output", "content_social_signals.json")), "content.search.social", contentProfileID(args, ""))
}

func cmdContentSearchSocialAccount(raw []string) error {
	args := cmdutil.Parse(raw)
	accounts := splitContentList(args.String("accounts", args.Pos(0)))
	if len(accounts) == 0 {
		return output.ErrValidation("--accounts is required")
	}
	maxPages, err := args.Int("max-pages", 1)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	count, err := args.Int("count", 20)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	includeRaw, err := args.Bool("include-raw", false)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	cfg, err := requireConfig()
	if err != nil {
		return output.ErrAuth("%v", err)
	}
	result, err := cloud.SearchSocialAccount(cloud.SocialAccountSearchRequest{
		Platform:   strings.TrimSpace(args.String("platform", "douyin")),
		Accounts:   accounts,
		MaxPages:   maxPages,
		Count:      count,
		IncludeRaw: includeRaw,
	}, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("social account search failed: %v", err)
	}
	return saveContentSearchResult(result, strings.TrimSpace(args.String("output", "content_social_account_signals.json")), "content.search.social-account", contentProfileID(args, ""))
}

func cmdContentSearchWebsearch(raw []string) error {
	args := cmdutil.Parse(raw)
	queries := contentSearchQueries(args)
	if len(queries) == 0 {
		return output.ErrValidation("--query or --queries is required")
	}
	num, err := args.Int("num", 6)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	cfg, err := requireConfig()
	if err != nil {
		return output.ErrAuth("%v", err)
	}
	result, err := cloud.SearchWebsearch(cloud.WebSearchRequest{
		Queries:   queries,
		DateRange: strings.TrimSpace(args.String("date-range", "7d")),
		Num:       num,
	}, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("websearch failed: %v", err)
	}
	return saveContentSearchResult(result, strings.TrimSpace(args.String("output", "content_websearch_signals.json")), "content.search.websearch", contentProfileID(args, ""))
}

func cmdContentSearchImage(raw []string) error {
	args := cmdutil.Parse(raw)
	queries := contentSearchQueries(args)
	if len(queries) == 0 {
		return output.ErrValidation("--query or --queries is required")
	}
	count, err := args.Int("count", 10)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	cfg, err := requireConfig()
	if err != nil {
		return output.ErrAuth("%v", err)
	}
	result, err := cloud.SearchImage(cloud.ImageSearchRequest{
		Queries: queries,
		Count:   count,
	}, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("image search failed: %v", err)
	}
	return saveContentSearchResult(result, strings.TrimSpace(args.String("output", "content_image_signals.json")), "content.search.image", contentProfileID(args, ""))
}

func contentSearchQueries(args cmdutil.Args) []string {
	if args.Has("query") {
		query := strings.TrimSpace(args.String("query", ""))
		if query == "" {
			return nil
		}
		return []string{query}
	}
	return splitContentList(args.String("queries", args.Pos(0)))
}

func cmdContentHistory(raw []string) error {
	args := cmdutil.Parse(raw)
	profileID := contentProfileID(args, "")
	if profileID == "" {
		return output.ErrValidation("--profile is required or run luma-cli profile use <id>")
	}
	artifactType := strings.TrimSpace(args.String("type", args.String("artifact-type", "")))
	group := contentGroupName(profileID)
	cfg, err := requireConfig()
	if err != nil {
		return output.ErrAuth("%v", err)
	}
	items, err := cloud.AssetList(group, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("list content history failed: %v", err)
	}
	filtered := []map[string]any{}
	for _, rawItem := range items {
		item := mapFromAny(rawItem)
		meta := mapFromAny(item["meta"])
		if artifactType != "" && strAny(meta["artifact_type"]) != artifactType {
			continue
		}
		filtered = append(filtered, item)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"profile_id": profileID, "group_name": group, "items": filtered}})
		return nil
	}
	if len(filtered) == 0 {
		fmt.Printf("No content history found for %s.\n", profileID)
		return nil
	}
	fmt.Printf("Content history: %s\n", group)
	fmt.Printf("%-28s %-28s %s\n", "TYPE", "FILENAME", "OBJECT KEY")
	for _, item := range filtered {
		meta := mapFromAny(item["meta"])
		fmt.Printf("%-28s %-28s %s\n", firstNonEmpty(strAny(meta["artifact_type"]), strAny(meta["luma_resource_kind"])), strAny(item["filename"]), strAny(item["object_key"]))
	}
	return nil
}

func cmdContentTopicMine(raw []string) error {
	args := cmdutil.Parse(raw)
	socialKeywords := splitContentList(args.String("social-keywords", ""))
	webQueries := splitContentList(args.String("web-queries", ""))
	limit, err := args.Int("limit-per-keyword", 10)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	webNum, err := args.Int("web-num", 6)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	maxRaw, err := args.Int("max-raw", 100)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	dateRange := strings.TrimSpace(args.String("date-range", "7d"))
	useBenchmark := !args.Has("no-benchmark")
	benchmarkAccountLimit, err := args.Int("benchmark-account-limit", 10)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	benchmarkVideoCount, err := args.Int("benchmark-video-count", 5)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	evergreenLimit, err := args.Int("evergreen-limit", 8)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	cfg, err := requireConfig()
	if err != nil {
		return output.ErrAuth("%v", err)
	}

	profileID := strings.TrimSpace(args.String("profile", ""))
	if profileID == "" {
		state, _ := loadProfileState()
		profileID = state.CurrentProfile
	}
	var benchmarkSource *contentBenchmarkSource
	var benchmarkAccounts []contentBenchmarkAccount
	var notes []string
	if useBenchmark && profileID != "" && benchmarkAccountLimit > 0 {
		source, accounts, sourceNotes, err := loadContentBenchmarkAccounts(profileID, benchmarkAccountLimit)
		if err != nil {
			notes = append(notes, "benchmark skipped: "+err.Error())
		} else {
			benchmarkSource = source
			benchmarkAccounts = accounts
			notes = append(notes, sourceNotes...)
		}
	}
	if len(socialKeywords) == 0 && len(webQueries) == 0 && len(benchmarkAccounts) == 0 {
		return output.ErrValidation("pass --social-keywords, --web-queries, or have a profile benchmark account pool")
	}

	var searchResults []map[string]any
	if len(socialKeywords) > 0 {
		result, err := cloud.SearchSocial(cloud.SocialSearchRequest{
			Platform:        strings.TrimSpace(args.String("platform", "douyin")),
			Keywords:        socialKeywords,
			DateRange:       dateRange,
			LimitPerKeyword: limit,
		}, cfg.CardKey)
		if err != nil {
			return output.ErrNetwork("social search failed: %v", err)
		}
		searchResults = append(searchResults, result)
	}
	if len(webQueries) > 0 {
		result, err := cloud.SearchWebsearch(cloud.WebSearchRequest{
			Queries:   webQueries,
			DateRange: dateRange,
			Num:       webNum,
		}, cfg.CardKey)
		if err != nil {
			return output.ErrNetwork("websearch failed: %v", err)
		}
		searchResults = append(searchResults, result)
	}
	if len(benchmarkAccounts) > 0 {
		benchmarkSuccesses := 0
		for _, account := range benchmarkAccounts {
			result, err := cloud.SearchSocialAccount(cloud.SocialAccountSearchRequest{
				Platform: strings.TrimSpace(args.String("platform", "douyin")),
				Accounts: []string{account.AccountInput},
				MaxPages: 1,
				Count:    benchmarkVideoCount,
			}, cfg.CardKey)
			if err != nil {
				label := firstNonEmpty(account.Nickname, account.AccountInput)
				notes = append(notes, fmt.Sprintf("benchmark account skipped: %s: %v", label, err))
				continue
			}
			benchmarkSuccesses++
			searchResults = append(searchResults, result)
		}
		if benchmarkSuccesses == 0 && len(searchResults) == 0 {
			return output.ErrNetwork("all benchmark account searches failed")
		}
	}

	mineResult := buildContentTopicMineResult(socialKeywords, webQueries, dateRange, maxRaw, evergreenLimit, searchResults...)
	mineResult.ProfileID = profileID
	mineResult.BenchmarkSource = benchmarkSource
	mineResult.BenchmarkAccounts = benchmarkAccounts
	mineResult.Notes = notes
	outputPath := strings.TrimSpace(args.String("output", "02_raw_signals.json"))
	abs, err := absoluteOutputPath(outputPath)
	if err != nil {
		return output.ErrValidation("bad output path: %v", err)
	}
	if err := writeJSONFile(abs, mineResult); err != nil {
		return output.ErrSystem("write output failed: %v", err)
	}
	recordProjectArtifact("content_raw_signals", abs, "content.discovery")
	cloudArtifact, uploadErr := uploadContentJSONArtifact(abs, mineResult, profileID, "content_raw_signals", "raw_signals.current.json", "content.discovery", nil)
	if uploadErr != nil {
		notes = append(notes, "cloud upload skipped: "+uploadErr.Error())
		mineResult.Notes = notes
	}

	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"result": mineResult, "output_path": abs, "cloud_artifact": cloudArtifact}})
		return nil
	}
	fmt.Printf("Raw signals: %d\n", mineResult.Counts.MergedRaw)
	fmt.Printf("Social raw: %d, benchmark raw: %d, web raw: %d, evergreen: %d, dupes: %d, empty: %d\n", mineResult.Counts.SocialRaw, mineResult.Counts.SocialAccountRaw, mineResult.Counts.WebRaw, mineResult.Counts.EvergreenThemes, mineResult.Counts.DroppedDupes, mineResult.Counts.DroppedEmpty)
	if benchmarkSource != nil {
		fmt.Printf("Benchmark accounts: %d from %s (%s)\n", len(benchmarkAccounts), benchmarkSource.Name, benchmarkSource.Kind)
	}
	fmt.Printf("Saved to: %s\n", abs)
	if cloudArtifact != nil {
		fmt.Printf("Uploaded to cloud group: %s\n", cloudArtifact["group_name"])
	}
	return nil
}

func contentProfileID(args cmdutil.Args, fallback string) string {
	profileID := strings.TrimSpace(args.String("profile", fallback))
	if profileID != "" {
		return profileID
	}
	state, _ := loadProfileState()
	return strings.TrimSpace(state.CurrentProfile)
}

func contentGroupName(profileID string) string {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return "content_global"
	}
	return "content_" + slugForID(profileID)
}

func uploadContentJSONArtifact(localPath string, value any, profileID, artifactType, resourceName, step string, extraMeta map[string]any) (map[string]any, error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return nil, fmt.Errorf("profile id is required for cloud content artifact")
	}
	cfg, err := requireConfig()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(resourceName) == "" {
		resourceName = artifactType + ".current.json"
	}
	group := contentGroupName(profileID)
	if strings.TrimSpace(localPath) == "" {
		tmp, err := os.CreateTemp("", "luma-content-artifact-*.json")
		if err != nil {
			return nil, err
		}
		localPath = tmp.Name()
		tmp.Close()
		defer os.Remove(localPath)
		if err := writeJSONFile(localPath, value); err != nil {
			return nil, err
		}
	}
	meta := map[string]any{
		"luma_resource_kind": "content_artifact",
		"artifact_type":      artifactType,
		"profile_id":         profileID,
		"step":               step,
		"is_current":         strings.Contains(resourceName, ".current."),
	}
	for key, value := range extraMeta {
		meta[key] = value
	}
	objectKey, err := cloud.UploadFileWithNameAndMeta(localPath, cfg.CardKey, group, resourceName, meta)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"group_name":     group,
		"object_key":     objectKey,
		"artifact_type":  artifactType,
		"resource_name":  resourceName,
		"profile_id":     profileID,
		"canonical_path": "cloud:" + group + "/" + resourceName,
	}, nil
}

func cmdContentTopicReview(raw []string) error {
	args := cmdutil.Parse(raw)
	inputPath := strings.TrimSpace(args.String("input", args.Pos(0)))
	if inputPath == "" {
		return output.ErrValidation("--input is required")
	}
	top, err := args.Int("top", 10)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	targetDuration := strings.TrimSpace(args.String("target-duration", "auto"))
	modelTier := strings.TrimSpace(args.String("model", args.String("model-tier", "")))
	absInput, err := absoluteOutputPath(inputPath)
	if err != nil {
		return output.ErrValidation("bad input path: %v", err)
	}
	data, err := os.ReadFile(absInput)
	if err != nil {
		return output.ErrSystem("read input failed: %v", err)
	}
	var mineResult contentTopicMineResult
	if err := json.Unmarshal(data, &mineResult); err != nil {
		return output.ErrValidation("parse mine result failed: %v", err)
	}
	profileID := strings.TrimSpace(args.String("profile", mineResult.ProfileID))
	var profile *lumaProfile
	if profileID != "" {
		if loaded, err := loadProfile(profileID); err == nil {
			profile = &loaded
		}
	}
	review := buildContentTopicReviewResult(mineResult, profile, absInput, top, targetDuration)
	outputPath := strings.TrimSpace(args.String("output", "03_topic_review.json"))
	absOutput, err := absoluteOutputPath(outputPath)
	if err != nil {
		return output.ErrValidation("bad output path: %v", err)
	}
	var outputValue any = review
	if modelTier != "" {
		cfg, err := requireConfig()
		if err != nil {
			return output.ErrAuth("%v", err)
		}
		input, err := structToMap(map[string]any{
			"local_review": review,
			"profile":      profile,
			"top":          top,
		})
		if err != nil {
			return output.ErrSystem("build review payload failed: %v", err)
		}
		resp, err := cloud.RunAgentAbility(
			"/v1/agent/topic/review",
			input,
			map[string]any{"model_tier": modelTier, "target_duration": targetDuration},
			cfg.CardKey,
		)
		if err != nil {
			return output.ErrNetwork("topic review failed: %v", err)
		}
		outputValue = resp.Result
	}
	if err := writeJSONFile(absOutput, outputValue); err != nil {
		return output.ErrSystem("write output failed: %v", err)
	}
	recordProjectArtifact("content_topic_review", absOutput, "topic.review")
	cloudArtifact, uploadErr := uploadContentJSONArtifact(absOutput, outputValue, profileID, "content_topic_review", "topic_review.current.json", "topic.review", nil)
	if uploadErr != nil && !runtimeOpts.JSON {
		fmt.Printf("Warning: cloud upload skipped: %v\n", uploadErr)
	}
	if runtimeOpts.JSON {
		data := map[string]any{"result": outputValue, "output_path": absOutput, "cloud_artifact": cloudArtifact}
		if uploadErr != nil {
			data["cloud_upload_error"] = uploadErr.Error()
		}
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: data})
		return nil
	}
	fmt.Printf("Topic cards: %d\n", len(review.TopicCards))
	fmt.Printf("Raw signals: %d, evergreen: %d, rejected: %d\n", review.Counts.RawSignals, review.Counts.EvergreenThemes, review.Counts.RejectedSignals)
	if len(review.TopicCards) > 0 {
		fmt.Println("Top topics:")
		for i, card := range review.TopicCards {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d. %.1f %s\n", i+1, card.TotalScore, card.Title)
		}
	}
	fmt.Printf("Saved to: %s\n", absOutput)
	if cloudArtifact != nil {
		fmt.Printf("Uploaded to cloud group: %s\n", cloudArtifact["group_name"])
	}
	return nil
}

func saveContentSearchResult(result map[string]any, outputPath string, step string, profileID string) error {
	abs, err := absoluteOutputPath(outputPath)
	if err != nil {
		return output.ErrValidation("bad output path: %v", err)
	}
	if err := writeJSONFile(abs, result); err != nil {
		return output.ErrSystem("write output failed: %v", err)
	}
	recordProjectArtifact("content_signals", abs, step)
	artifactType := "content_signals"
	logicalName := strings.ReplaceAll(step, ".", "_") + ".current.json"
	cloudArtifact, uploadErr := uploadContentJSONArtifact(abs, result, profileID, artifactType, logicalName, step, nil)
	signals := rawSignalsFromContentSearchResult(result)
	if runtimeOpts.JSON {
		data := map[string]any{"result": result, "output_path": abs, "count": len(signals), "cloud_artifact": cloudArtifact}
		if uploadErr != nil {
			data["cloud_upload_error"] = uploadErr.Error()
		}
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: data})
		return nil
	}
	fmt.Printf("Signals: %d\n", len(signals))
	fmt.Printf("Saved to: %s\n", abs)
	if uploadErr != nil {
		fmt.Printf("Warning: cloud upload skipped: %v\n", uploadErr)
	} else if cloudArtifact != nil {
		fmt.Printf("Uploaded to cloud group: %s\n", cloudArtifact["group_name"])
	}
	return nil
}

func buildContentTopicMineResult(socialKeywords []string, webQueries []string, dateRange string, maxRaw int, evergreenLimit int, searchResults ...map[string]any) contentTopicMineResult {
	var all []map[string]any
	groups := contentSignalGroups{}
	counts := contentTopicMineCounts{}
	for _, result := range searchResults {
		source := strings.TrimSpace(strAny(result["source"]))
		signals := rawSignalsFromContentSearchResult(result)
		switch source {
		case "social":
			counts.SocialRaw += len(signals)
			groups.HotSignals = append(groups.HotSignals, signals...)
		case "social_account":
			counts.SocialAccountRaw += len(signals)
			groups.BenchmarkSignals = append(groups.BenchmarkSignals, signals...)
		case "websearch":
			counts.WebRaw += len(signals)
			groups.WebSignals = append(groups.WebSignals, signals...)
		}
		all = append(all, signals...)
	}
	merged, droppedDupes, droppedEmpty := mergeContentRawSignals(all, maxRaw)
	themes := mineEvergreenThemesFromSignals(merged, evergreenLimit)
	counts.MergedRaw = len(merged)
	counts.DroppedDupes = droppedDupes
	counts.DroppedEmpty = droppedEmpty
	counts.EvergreenThemes = len(themes)
	if maxRaw > 0 && len(all)-droppedDupes-droppedEmpty > maxRaw {
		counts.TruncatedFrom = len(all) - droppedDupes - droppedEmpty
	}
	return contentTopicMineResult{
		DateRange:       dateRange,
		SocialKeywords:  socialKeywords,
		WebQueries:      webQueries,
		SignalGroups:    groups,
		EvergreenThemes: themes,
		RawSignals:      merged,
		Counts:          counts,
		GeneratedAt:     time.Now(),
	}
}

func buildContentTopicReviewResult(mine contentTopicMineResult, profile *lumaProfile, inputPath string, top int, targetDuration string) contentTopicReviewResult {
	if top <= 0 {
		top = 10
	}
	rejected := 0
	filteredSignals := make([]map[string]any, 0, len(mine.RawSignals))
	for _, signal := range mine.RawSignals {
		if rejectWeakTopicSignal(signal, profile) {
			rejected++
			continue
		}
		filteredSignals = append(filteredSignals, signal)
	}
	var cards []contentTopicCard
	for _, signal := range topSignalsForReview(filteredSignals, top*3) {
		theme, hasTheme := bestThemeForSignal(signal, mine.EvergreenThemes)
		cards = append(cards, buildSignalTopicCard(signal, profile, theme, hasTheme))
	}
	if len(cards) < minInt(top, 5) {
		for _, theme := range mine.EvergreenThemes {
			evidence := evidenceForTheme(theme, filteredSignals, 4)
			if len(evidence) == 0 && len(theme.SourceTitles) == 0 {
				continue
			}
			if themeAlreadyUsed(theme.ThemeID, cards) {
				continue
			}
			cards = append(cards, buildThemeTopicCard(theme, evidence, profile))
			if len(cards) >= top {
				break
			}
		}
	}
	if len(cards) == 0 {
		for _, theme := range mine.EvergreenThemes {
			evidence := evidenceForTheme(theme, filteredSignals, 4)
			if len(evidence) == 0 && len(theme.SourceTitles) == 0 {
				continue
			}
			cards = append(cards, buildThemeTopicCard(theme, evidence, profile))
			if len(cards) >= top {
				break
			}
		}
	}
	cards = dedupeTopicCards(cards)
	profileFit := inferProfileFormatFit(profile, targetDuration)
	for i := range cards {
		enrichTopicCardReviewFields(&cards[i], profile)
		applyFormatFitToTopicCard(&cards[i], profileFit, targetDuration)
	}
	sort.SliceStable(cards, func(i, j int) bool {
		if cards[i].TotalScore == cards[j].TotalScore {
			return cards[i].Scores.Freshness > cards[j].Scores.Freshness
		}
		return cards[i].TotalScore > cards[j].TotalScore
	})
	if len(cards) > top {
		cards = cards[:top]
	}
	for i := range cards {
		cards[i].TopicID = fmt.Sprintf("topic_%03d", i+1)
		if i < 3 {
			cards[i].Status = "shortlisted"
			cards[i].Recommendation = "进入 material.plan 前可先做轻量事实核验"
		} else {
			cards[i].Status = "ranked"
			cards[i].Recommendation = "保留为候选选题"
		}
	}
	profileID := mine.ProfileID
	if profile != nil {
		profileID = profile.ID
	}
	return contentTopicReviewResult{
		ProfileID:        profileID,
		InputPath:        inputPath,
		TargetDuration:   targetDuration,
		ProfileFormatFit: profileFit,
		TopicCards:       cards,
		Counts: contentReviewCount{
			RawSignals:       len(mine.RawSignals),
			EvergreenThemes:  len(mine.EvergreenThemes),
			GeneratedCards:   len(cards),
			RejectedSignals:  rejected,
			SelectedTopCards: minInt(len(cards), 3),
		},
		GeneratedAt: time.Now(),
	}
}

func dedupeTopicCards(cards []contentTopicCard) []contentTopicCard {
	seen := map[string]bool{}
	out := make([]contentTopicCard, 0, len(cards))
	for _, card := range cards {
		key := compactTopicTitle(card.Title)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, card)
	}
	return out
}

func themeAlreadyUsed(themeID string, cards []contentTopicCard) bool {
	for _, card := range cards {
		if card.ThemeID == themeID {
			return true
		}
	}
	return false
}

func enrichTopicCardReviewFields(card *contentTopicCard, profile *lumaProfile) {
	if card == nil {
		return
	}
	card.ObjectiveFacts = objectiveFactsForTopicCard(*card)
	card.PublicEntry = publicEntryForTopicCard(*card)
	card.WhyGeneralAudienceCares = whyGeneralAudienceCaresForTopicCard(*card)
	card.AudienceFilterTurn = audienceFilterTurnForTopicCard(*card, profile)
	card.ReviewThinking = "Topic card keeps factual signals separate from the creative opening: use public_entry to attract a broader audience, then use audience_filter_turn to narrow back to the profile."
}

func objectiveFactsForTopicCard(card contentTopicCard) []string {
	var facts []string
	if card.SourceType != "" {
		facts = append(facts, "source_type: "+card.SourceType)
	}
	if card.Theme != "" {
		facts = append(facts, "matched_theme: "+card.Theme)
	}
	for _, item := range card.EvidenceSignals {
		if item.Title != "" {
			facts = append(facts, "evidence: "+item.Title)
		}
		if len(facts) >= 5 {
			break
		}
	}
	return facts
}

func publicEntryForTopicCard(card contentTopicCard) string {
	text := card.Title + " " + card.Angle + " " + card.CoreOpinion + " " + card.Theme
	if strings.Contains(text, "AI") || strings.Contains(text, "Agent") || strings.Contains(text, "agent") {
		return "A lot of people get excited about new tools, but very few actually change how work gets done."
	}
	if strings.Contains(text, "鑾峰") || strings.Contains(text, "钀ラ攢") {
		return "Everyone wants more traffic, but the harder question is why attention does not turn into customers."
	}
	return "Start from the broad phenomenon behind this topic, not from the narrow target audience label."
}

func whyGeneralAudienceCaresForTopicCard(card contentTopicCard) string {
	text := card.Title + " " + card.Angle + " " + card.CoreOpinion
	if strings.Contains(text, "AI") || strings.Contains(text, "Agent") || strings.Contains(text, "agent") {
		return "It connects to a general anxiety: tools are getting stronger, but most people's work habits and results are not changing at the same speed."
	}
	return "It can be framed as a common decision problem before narrowing into the profile audience."
}

func audienceFilterTurnForTopicCard(card contentTopicCard, profile *lumaProfile) string {
	audience := defaultAudience(profile)
	return fmt.Sprintf("After the public entry, narrow the topic back to %s and explain why this objective signal matters to their business decisions.", audience)
}

func inferProfileFormatFit(profile *lumaProfile, targetDuration string) contentFormatFit {
	text := ""
	if profile != nil {
		text = profile.Identity + " " + strings.Join(profile.Audience, " ") + " " + strings.Join(profile.Stance, " ")
	}
	format := "short_explainer"
	duration := 90
	reasons := []string{"profile 默认适合轻量解释视频"}
	if len(matchedKeywords(text, []string{"老板", "企业", "SaaS", "saas", "Agent", "智能体", "业务", "流程", "营销", "获客", "创业"})) >= 3 {
		format = "midform_commentary"
		duration = 240
		reasons = []string{"profile 面向老板/企业/业务流程，默认更适合结构化中场观点视频"}
	}
	if targetDuration != "" && targetDuration != "auto" {
		if parsed := parseDurationTarget(targetDuration); parsed > 0 {
			duration = parsed
			format = formatForDuration(parsed)
			reasons = append(reasons, "命令显式指定目标时长")
		}
	}
	return contentFormatFit{
		Recommended:    format,
		DurationSec:    duration,
		Confidence:     0.75,
		Fallback:       "short_explainer",
		ProfileDefault: format,
		TopicCapacity:  "profile_default",
		Reason:         reasons,
	}
}

func applyFormatFitToTopicCard(card *contentTopicCard, profileFit contentFormatFit, targetDuration string) {
	capacity, reasons := topicCapacity(card)
	format := profileFit.Recommended
	duration := profileFit.DurationSec
	confidence := 0.72
	fallback := "short_explainer"
	if targetDuration == "" || targetDuration == "auto" {
		switch capacity {
		case "midform":
			if profileFit.Recommended == "midform_commentary" || card.TotalScore >= 6.8 {
				format = "midform_commentary"
				duration = 240
				confidence = 0.82
			} else {
				format = "short_explainer"
				duration = 90
				confidence = 0.68
			}
		case "short":
			format = "short_explainer"
			duration = 90
			confidence = 0.76
		default:
			format = firstNonEmpty(profileFit.Recommended, "short_explainer")
			duration = profileFit.DurationSec
		}
	} else if parsed := parseDurationTarget(targetDuration); parsed > 0 {
		duration = parsed
		format = formatForDuration(parsed)
		confidence = 0.7
		if parsed >= 180 && capacity == "short" {
			format = "short_explainer"
			duration = 90
			confidence = 0.64
			reasons = append(reasons, "topic 承载力不足，降级为短解释")
		}
	}
	card.FormatFit = contentFormatFit{
		Recommended:    format,
		DurationSec:    duration,
		Confidence:     round1(confidence),
		Fallback:       fallback,
		ProfileDefault: profileFit.ProfileDefault,
		TopicCapacity:  capacity,
		Reason:         append(profileFit.Reason, reasons...),
	}
	if duration >= 180 && capacity == "midform" {
		plan := buildLongformPlan(*card, duration)
		card.LongformPlan = &plan
		card.TotalScore = round1(card.TotalScore + 0.4)
	} else if duration < 180 {
		card.LongformPlan = nil
	}
}

func topicCapacity(card *contentTopicCard) (string, []string) {
	reasons := []string{}
	points := 0
	if card.ThemeID != "" {
		points++
		reasons = append(reasons, "命中常青主题，可展开结构性观点")
	}
	if len(card.EvidenceSignals) >= 2 {
		points++
		reasons = append(reasons, "有多个证据信号支撑章节化表达")
	}
	if len(card.EvidenceSignals) >= 4 {
		points++
		reasons = append(reasons, "证据信号较充足")
	}
	text := card.Title + " " + card.Angle + " " + card.CoreOpinion + " " + card.CommonMisunderstand
	if len(matchedKeywords(text, []string{"不是", "而是", "问题", "误区", "关键", "难点", "改变", "重构"})) >= 2 {
		points++
		reasons = append(reasons, "有误区/反常识/转折表达")
	}
	if len(matchedKeywords(text, []string{"流程", "工作流", "业务", "组织", "获客", "营销", "交付", "执行", "闭环"})) >= 2 {
		points++
		reasons = append(reasons, "能展开成业务逻辑链")
	}
	if points >= 3 {
		return "midform", reasons
	}
	return "short", append(reasons, "更适合短解释或短评")
}

func buildLongformPlan(card contentTopicCard, duration int) contentLongformPlan {
	evidenceTitles := make([]string, 0, len(card.EvidenceSignals))
	for _, item := range card.EvidenceSignals {
		if item.Title != "" {
			evidenceTitles = append(evidenceTitles, item.Title)
		}
	}
	sections := []contentLongformSection{
		{
			Section:    "开头结论",
			Claim:      card.Title,
			Evidence:   takeStrings(evidenceTitles, 1),
			VisualNeed: "结论大字卡 + 数字人口播",
		},
		{
			Section:    "现象背景",
			Claim:      firstNonEmpty(firstEvidenceTitle(card), "最近相关信号正在变多，但大多数人只看到表层热闹"),
			Evidence:   takeStrings(evidenceTitles, 2),
			VisualNeed: "信号列表卡 / 对标账号标题卡",
		},
		{
			Section:    "常见误区",
			Claim:      firstNonEmpty(card.CommonMisunderstand, "大多数人会把它理解成工具更新，而不是流程变化"),
			VisualNeed: "误区 vs 真相对比卡",
		},
		{
			Section:    "核心判断",
			Claim:      card.CoreOpinion,
			Evidence:   takeStrings(evidenceTitles, 3),
			VisualNeed: "流程卡 / 逻辑链条卡",
		},
		{
			Section:    "对受众的影响",
			Claim:      card.AudienceValue,
			VisualNeed: "影响清单卡",
		},
		{
			Section:    "行动建议",
			Claim:      "先不要从工具清单开始，而是选一个高频业务流程，判断 AI 能否稳定嵌进去",
			VisualNeed: "行动建议卡",
		},
	}
	return contentLongformPlan{
		Suitable:               true,
		DurationTargetSec:      duration,
		CoreThesis:             card.CoreOpinion,
		ChapterCount:           len(sections),
		EstimatedMaterialCount: maxInt(4, minInt(8, len(card.EvidenceSignals)+4)),
		LogicChain:             sections,
		WhySuitable:            card.FormatFit.Reason,
		Risks:                  card.Risks,
	}
}

func parseDurationTarget(value string) int {
	value = strings.TrimSpace(strings.TrimSuffix(value, "s"))
	switch value {
	case "short":
		return 90
	case "mid", "midform":
		return 240
	case "long":
		return 300
	}
	var n int
	if _, err := fmt.Sscanf(value, "%d", &n); err == nil {
		return n
	}
	return 0
}

func formatForDuration(duration int) string {
	if duration >= 180 {
		return "midform_commentary"
	}
	if duration >= 120 {
		return "structured_explainer"
	}
	return "short_explainer"
}

func takeStrings(values []string, limit int) []string {
	if limit > 0 && len(values) > limit {
		return values[:limit]
	}
	return values
}

func firstEvidenceTitle(card contentTopicCard) string {
	if len(card.EvidenceSignals) == 0 {
		return ""
	}
	return card.EvidenceSignals[0].Title
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func buildThemeTopicCard(theme contentEvergreenTheme, evidence []contentEvidenceBrief, profile *lumaProfile) contentTopicCard {
	lead := ""
	if len(evidence) > 0 {
		lead = evidence[0].Title
	} else if len(theme.SourceTitles) > 0 {
		lead = theme.SourceTitles[0]
	}
	title := theme.Theme
	if lead != "" {
		title = titleFromThemeAndSignal(theme, lead)
	}
	scores := scoreTopic(theme.Theme+" "+strings.Join(theme.SourceTitles, " "), theme.Score, evidence, profile)
	return contentTopicCard{
		Title:               title,
		Angle:               theme.Pain,
		CoreOpinion:         theme.Opinion,
		CommonMisunderstand: commonMisunderstandingForTheme(theme.Theme),
		AudienceValue:       audienceValueForTheme(theme, profile),
		SourceType:          "evergreen_x_signals",
		ThemeID:             theme.ThemeID,
		Theme:               theme.Theme,
		EvidenceSignals:     evidence,
		MaterialHypothesis:  materialHypothesisForEvidence(evidence),
		Risks:               risksForEvidence(evidence),
		Scores:              scores,
		TotalScore:          topicTotalScore(scores),
		Debug: map[string]any{
			"theme_score":    theme.Score,
			"evidence_count": theme.EvidenceCount,
		},
	}
}

func buildSignalTopicCard(signal map[string]any, profile *lumaProfile, theme contentEvergreenTheme, hasTheme bool) contentTopicCard {
	title := compactTopicTitle(strAny(signal["title"]))
	evidence := []contentEvidenceBrief{evidenceBriefFromSignal(signal)}
	scores := scoreTopic(title, signalEngagement(signal), evidence, profile)
	card := contentTopicCard{
		Title:               title,
		Angle:               signalAngle(title),
		CoreOpinion:         signalOpinion(title, profile),
		CommonMisunderstand: "大多数人会只看事件表面，而忽略它背后的流程变化",
		AudienceValue:       defaultAudienceValue(profile),
		SourceType:          firstNonEmpty(strAny(signal["source"]), "signal"),
		EvidenceSignals:     evidence,
		MaterialHypothesis:  materialHypothesisForEvidence(evidence),
		Risks:               risksForEvidence(evidence),
		Scores:              scores,
		TotalScore:          topicTotalScore(scores),
	}
	if hasTheme {
		card.ThemeID = theme.ThemeID
		card.Theme = theme.Theme
		card.Title = titleFromSignalAndTheme(title, theme)
		card.Angle = firstNonEmpty(theme.Pain, card.Angle)
		card.CoreOpinion = firstNonEmpty(theme.Opinion, card.CoreOpinion)
		card.CommonMisunderstand = commonMisunderstandingForTheme(theme.Theme)
		card.AudienceValue = audienceValueForTheme(theme, profile)
		card.Debug = map[string]any{"matched_evergreen_theme": theme.Theme}
	}
	return card
}

func titleFromSignalAndTheme(signalTitle string, theme contentEvergreenTheme) string {
	switch theme.ThemeID {
	case "evg_ai_marketing_acquisition":
		if strings.Contains(signalTitle, "同城") || strings.Contains(signalTitle, "GEO") {
			return "AI 同城获客改变的不是流量入口，而是商家被推荐的规则"
		}
		return "AI 获客真正改变的不是内容数量，而是营销工作流"
	case "evg_agent_vs_tool":
		if strings.Contains(signalTitle, "变现") {
			return "智能体开始变现后，真正关键的不是工具，而是交付闭环"
		}
		return "Agent 和普通 AI 工具的区别，不在会不会聊天，而在能不能执行"
	case "evg_ai_business_workflow":
		return "老板用 AI 没效果，问题通常不在工具，而在业务流程"
	case "evg_ai_native_org":
		return "企业 AI 落地的关键，不是人人会用，而是组织会沉淀"
	case "evg_enterprise_data_agent":
		return "数据 Agent 的难点不是写 SQL，而是给出可信业务答案"
	case "evg_vertical_saas_agent":
		return "垂直 SaaS 的 Agent 机会，不在模型，而在行业场景"
	case "evg_ai_personal_opportunity":
		return "普通人用 AI 赚钱的核心不是工具清单，而是可交付服务"
	case "evg_ai_video_content_system":
		return "AI 视频真正改变的不是剪辑成本，而是内容测试成本"
	default:
		return compactTopicTitle(signalTitle)
	}
}

func rejectWeakTopicSignal(signal map[string]any, profile *lumaProfile) bool {
	title := strings.TrimSpace(strAny(signal["title"]))
	if title == "" {
		return true
	}
	noiseWords := []string{"NBA", "机器人", "股票", "A股", "基金", "房产", "娱乐", "情感"}
	for _, word := range noiseWords {
		if strings.Contains(title, word) && strongBusinessTopicHits(title) == 0 {
			return true
		}
	}
	adWords := []string{"实名承诺", "评论666", "报价", "欢迎嘀我", "点击领取", "感兴趣", "留言", "私信"}
	adHits := matchedKeywords(title, adWords)
	if len(adHits) >= 2 {
		return true
	}
	return false
}

func businessTopicHits(text string) int {
	return len(matchedKeywords(text, []string{"Agent", "agent", "智能体", "获客", "营销", "销售", "企业", "老板", "业务", "流程", "工作流", "落地", "SaaS", "saas", "数据", "CRM", "私域"}))
}

func strongBusinessTopicHits(text string) int {
	return len(matchedKeywords(text, []string{"Agent", "agent", "智能体", "获客", "营销", "销售", "企业", "老板", "业务流程", "工作流", "落地", "SaaS", "saas", "CRM", "私域"}))
}

func evidenceForTheme(theme contentEvergreenTheme, signals []map[string]any, limit int) []contentEvidenceBrief {
	type scoredEvidence struct {
		score float64
		item  contentEvidenceBrief
	}
	var items []scoredEvidence
	for _, signal := range signals {
		title := strAny(signal["title"])
		if title == "" {
			continue
		}
		if len(theme.Keywords) > 0 && strings.HasPrefix(theme.ThemeID, "evg_") {
			rule := evergreenRuleForThemeID(theme.ThemeID)
			if len(rule.Required) > 0 && len(matchedKeywords(title, rule.Required)) == 0 {
				continue
			}
		}
		matches := matchedKeywords(title, theme.Keywords)
		if len(matches) == 0 {
			continue
		}
		score := float64(len(matches))*1.5 + signalEngagement(signal)
		if strAny(signal["source"]) == "social_account" {
			score += 2
		}
		items = append(items, scoredEvidence{score: score, item: evidenceBriefFromSignal(signal)})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].score > items[j].score })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	out := make([]contentEvidenceBrief, 0, len(items))
	for _, item := range items {
		out = append(out, item.item)
	}
	return out
}

func evergreenRuleForThemeID(themeID string) evergreenRule {
	id := strings.TrimPrefix(themeID, "evg_")
	for _, rule := range defaultEvergreenRules {
		if rule.ID == id {
			return rule
		}
	}
	return evergreenRule{}
}

func topSignalsForReview(signals []map[string]any, limit int) []map[string]any {
	items := append([]map[string]any(nil), signals...)
	sort.SliceStable(items, func(i, j int) bool {
		return signalReviewScore(items[i]) > signalReviewScore(items[j])
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func signalReviewScore(signal map[string]any) float64 {
	title := strAny(signal["title"])
	score := signalEngagement(signal)
	if strAny(signal["source"]) == "social_account" {
		score += 2
	}
	score += float64(len(matchedKeywords(title, []string{"Agent", "agent", "智能体", "获客", "营销", "企业", "老板", "业务", "流程", "落地", "SaaS", "数据"}))) * 0.8
	score -= float64(len(matchedKeywords(title, []string{"报价", "评论666", "NBA", "机器人", "股票"}))) * 1.2
	return score
}

func bestThemeForSignal(signal map[string]any, themes []contentEvergreenTheme) (contentEvergreenTheme, bool) {
	title := strAny(signal["title"])
	bestScore := 0.0
	var best contentEvergreenTheme
	for _, theme := range themes {
		rule := evergreenRuleForThemeID(theme.ThemeID)
		if len(rule.Required) > 0 && len(matchedKeywords(title, rule.Required)) == 0 {
			continue
		}
		matches := matchedKeywords(title, theme.Keywords)
		if len(matches) < 2 {
			continue
		}
		score := float64(len(matches)) + theme.Score/100
		if score > bestScore {
			bestScore = score
			best = theme
		}
	}
	return best, bestScore > 0
}

func titleFromThemeAndSignal(theme contentEvergreenTheme, lead string) string {
	lead = compactTopicTitle(lead)
	switch theme.ThemeID {
	case "evg_agent_vs_tool":
		return "智能体开始变现后，真正关键的不是工具，而是执行闭环"
	case "evg_ai_business_workflow":
		return "老板用 AI 没效果，问题通常不在工具，而在业务流程"
	case "evg_ai_marketing_acquisition":
		return "AI 获客真正改变的不是内容数量，而是营销工作流"
	case "evg_ai_native_org":
		return "企业 AI 落地的关键，不是人人会用，而是组织会沉淀"
	case "evg_enterprise_data_agent":
		return "数据 Agent 的难点不是写 SQL，而是给出可信业务答案"
	case "evg_vertical_saas_agent":
		return "垂直 SaaS 的 Agent 机会，不在模型，而在行业场景"
	default:
		return compactTopicTitle(firstNonEmpty(theme.Theme, lead))
	}
}

func compactTopicTitle(title string) string {
	title = strings.TrimSpace(strings.ReplaceAll(title, "\n", " "))
	for strings.Contains(title, "  ") {
		title = strings.ReplaceAll(title, "  ", " ")
	}
	cutWords := []string{" #", "。#", " 原文", " 评论区", " 欢迎"}
	for _, cut := range cutWords {
		if idx := strings.Index(title, cut); idx > 0 {
			title = strings.TrimSpace(title[:idx])
		}
	}
	runes := []rune(title)
	if len(runes) > 42 {
		return string(runes[:42])
	}
	return title
}

func commonMisunderstandingForTheme(theme string) string {
	if strings.Contains(theme, "工具") {
		return "大多数人会把问题理解成该买哪个工具，而不是该改哪条流程"
	}
	if strings.Contains(theme, "获客") || strings.Contains(theme, "营销") {
		return "大多数人会把 AI 获客理解成自动发内容，而不是线索和转化链路重构"
	}
	if strings.Contains(theme, "数据") {
		return "大多数人会以为会写 SQL 就等于懂业务数据"
	}
	return "大多数人会只看热闹和概念，而忽略真正能落地的结构变化"
}

func audienceValueForTheme(theme contentEvergreenTheme, profile *lumaProfile) string {
	audience := firstNonEmpty(theme.Audience, defaultAudience(profile))
	return fmt.Sprintf("对%s来说，这个选题能把概念翻译成可执行的业务判断", audience)
}

func defaultAudienceValue(profile *lumaProfile) string {
	return fmt.Sprintf("对%s来说，价值在于判断这件事是否会影响获客、销售或交付流程", defaultAudience(profile))
}

func defaultAudience(profile *lumaProfile) string {
	if profile != nil && len(profile.Audience) > 0 {
		return profile.Audience[0]
	}
	return "目标受众"
}

func signalAngle(title string) string {
	if strings.Contains(title, "获客") || strings.Contains(title, "营销") {
		return "把单点工具玩法放回获客和转化流程里看"
	}
	if strings.Contains(title, "Agent") || strings.Contains(title, "智能体") {
		return "从概念热度转向真实执行能力和业务闭环"
	}
	return "从事件表面抽出对业务流程的影响"
}

func signalOpinion(title string, profile *lumaProfile) string {
	if strings.Contains(title, "获客") || strings.Contains(title, "营销") {
		return "真正值得关注的不是 AI 能不能生成内容，而是能不能重做线索、触达、跟进和复盘链路"
	}
	if strings.Contains(title, "Agent") || strings.Contains(title, "智能体") {
		return "Agent 的价值不在炫技，而在能否把重复决策和执行稳定跑起来"
	}
	if profile != nil && len(profile.Stance) > 0 {
		return profile.Stance[0]
	}
	return "这件事的重点不在新闻本身，而在它改变了什么生产方式"
}

func scoreTopic(text string, baseScore float64, evidence []contentEvidenceBrief, profile *lumaProfile) contentTopicScores {
	profileHits := profileKeywordHits(text, profile)
	engagement := 0.0
	for _, item := range evidence {
		if item.Likes > engagement {
			engagement = item.Likes
		}
	}
	return contentTopicScores{
		Freshness:            clampScore(5 + sourceFreshnessBoost(evidence)),
		AudienceRelevance:    clampScore(4 + float64(profileHits)*0.8 + float64(len(matchedKeywords(text, []string{"老板", "企业", "获客", "营销", "SaaS"})))*0.6),
		ProfileFit:           clampScore(4 + float64(profileHits)*0.9),
		Conflict:             clampScore(4 + float64(len(matchedKeywords(text, []string{"不是", "而是", "难点", "问题", "变现", "替代", "重构"})))*0.8),
		MaterialAvailability: clampScore(4.5 + float64(len(evidence))*0.8),
		ConversionPotential:  clampScore(4 + float64(len(matchedKeywords(text, []string{"获客", "营销", "销售", "老板", "业务", "落地", "工具"})))*0.7 + engagementScoreFromLikes(engagement)),
	}
}

func topicTotalScore(scores contentTopicScores) float64 {
	return round1(scores.Freshness*0.15 + scores.AudienceRelevance*0.2 + scores.ProfileFit*0.2 + scores.Conflict*0.15 + scores.MaterialAvailability*0.15 + scores.ConversionPotential*0.15)
}

func profileKeywordHits(text string, profile *lumaProfile) int {
	if profile == nil {
		return len(matchedKeywords(text, []string{"AI", "Agent", "智能体", "获客", "营销", "企业", "老板", "SaaS"}))
	}
	values := []string{profile.Identity}
	values = append(values, profile.Audience...)
	values = append(values, profile.Stance...)
	hits := 0
	for _, value := range values {
		for _, token := range splitLooseKeywords(value) {
			if strings.Contains(text, token) {
				hits++
			}
		}
	}
	return hits
}

func splitLooseKeywords(value string) []string {
	value = strings.ReplaceAll(value, "不是", " ")
	value = strings.ReplaceAll(value, "解决", " ")
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == ' ' || r == '\n' || r == '\t'
	})
	out := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len([]rune(part)) >= 2 {
			out = append(out, part)
		}
	}
	return out
}

func sourceFreshnessBoost(evidence []contentEvidenceBrief) float64 {
	if len(evidence) == 0 {
		return 0
	}
	for _, item := range evidence {
		if strings.HasPrefix(item.PublishedAt, "2026-06") || item.PublishedAt == "" {
			return 2
		}
	}
	return 1
}

func engagementScoreFromLikes(likes float64) float64 {
	if likes > 10000 {
		return 2
	}
	if likes > 1000 {
		return 1.5
	}
	if likes > 100 {
		return 1
	}
	return 0
}

func clampScore(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 10 {
		return 10
	}
	return round1(value)
}

func evidenceBriefFromSignal(signal map[string]any) contentEvidenceBrief {
	stats, _ := signal["stats"].(map[string]any)
	likes, _ := numberAny(stats["likes"])
	return contentEvidenceBrief{
		Source:      strAny(signal["source"]),
		Title:       compactTopicTitle(strAny(signal["title"])),
		AuthorName:  strAny(signal["author_name"]),
		URL:         strAny(signal["url"]),
		Likes:       likes,
		PublishedAt: strAny(signal["published_at"]),
	}
}

func materialHypothesisForEvidence(evidence []contentEvidenceBrief) []string {
	items := []string{"观点卡/对比卡", "口播关键词字幕"}
	hasWeb := false
	hasSocial := false
	for _, item := range evidence {
		if item.Source == "websearch" {
			hasWeb = true
		}
		if item.Source == "social" || item.Source == "social_account" {
			hasSocial = true
		}
	}
	if hasWeb {
		items = append([]string{"网页/新闻页面截图"}, items...)
	}
	if hasSocial {
		items = append(items, "对标账号标题截图或信息卡")
	}
	return uniqueProfileStrings(items)
}

func risksForEvidence(evidence []contentEvidenceBrief) []string {
	risks := []string{}
	if len(evidence) == 0 {
		return []string{"缺少证据信号，进入 material.plan 前需要补充搜索"}
	}
	hasWeb := false
	for _, item := range evidence {
		if item.Source == "websearch" {
			hasWeb = true
		}
	}
	if !hasWeb {
		risks = append(risks, "目前主要来自短视频信号，事实性表达需要再做 websearch/官方来源核验")
	}
	return risks
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func loadContentBenchmarkAccounts(profileID string, limit int) (*contentBenchmarkSource, []contentBenchmarkAccount, []string, error) {
	profile, err := loadProfile(profileID)
	if err != nil {
		return nil, nil, nil, err
	}
	asset, ok := selectBenchmarkAsset(profile.Assets, "benchmark")
	if !ok {
		asset, ok = selectBenchmarkAsset(profile.Assets, "benchmark_candidate")
	}
	if !ok {
		return nil, nil, nil, fmt.Errorf("no kind=benchmark or kind=benchmark_candidate asset found for profile %s", profileID)
	}
	var data []byte
	if strings.TrimSpace(asset.LocalPath) != "" {
		data, err = os.ReadFile(asset.LocalPath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("read benchmark asset failed: %w", err)
		}
	} else {
		cfg, err := requireConfig()
		if err != nil {
			return nil, nil, nil, err
		}
		item, err := resourceByObjectKey(asset.Group, asset.ObjectKey, cfg.CardKey)
		if err != nil {
			return nil, nil, nil, err
		}
		text, err := downloadResourceText(strAny(item["resource_url"]))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("download benchmark asset failed: %w", err)
		}
		data = []byte(text)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, nil, nil, fmt.Errorf("parse benchmark asset failed: %w", err)
	}
	accounts := extractBenchmarkAccounts(payload, limit)
	if len(accounts) == 0 {
		return nil, nil, nil, fmt.Errorf("benchmark asset %s contains no usable accounts", asset.Name)
	}
	source := &contentBenchmarkSource{
		ProfileID: profileID,
		AssetID:   asset.ID,
		Kind:      asset.Kind,
		Name:      asset.Name,
		ObjectKey: asset.ObjectKey,
		LocalPath: asset.LocalPath,
		CreatedAt: asset.CreatedAt,
	}
	notes := []string{}
	if asset.Kind == "benchmark_candidate" {
		notes = append(notes, "using latest benchmark_candidate because no approved kind=benchmark asset exists")
	}
	return source, accounts, notes, nil
}

func selectBenchmarkAsset(assets []profileAsset, kind string) (profileAsset, bool) {
	candidates := make([]profileAsset, 0, len(assets))
	for _, asset := range assets {
		if strings.EqualFold(strings.TrimSpace(asset.Kind), kind) {
			candidates = append(candidates, asset)
		}
	}
	if len(candidates) == 0 {
		return profileAsset{}, false
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].CreatedAt.After(candidates[j].CreatedAt)
	})
	return candidates[0], true
}

func extractBenchmarkAccounts(payload map[string]any, limit int) []contentBenchmarkAccount {
	var out []contentBenchmarkAccount
	seen := map[string]bool{}
	pools := []struct {
		key  string
		tier string
	}{
		{key: "recommended_seed_accounts", tier: "A"},
		{key: "observation_accounts", tier: "B"},
		{key: "weekly_candidates", tier: "C"},
	}
	for _, pool := range pools {
		values, _ := payload[pool.key].([]any)
		for _, value := range values {
			item, ok := value.(map[string]any)
			if !ok {
				continue
			}
			account := benchmarkAccountFromMap(item, pool.tier)
			if account.AccountInput == "" || seen[account.AccountInput] {
				continue
			}
			seen[account.AccountInput] = true
			out = append(out, account)
			if limit > 0 && len(out) >= limit {
				return out
			}
		}
	}
	return out
}

func benchmarkAccountFromMap(item map[string]any, fallbackTier string) contentBenchmarkAccount {
	accountID := strings.TrimSpace(strAny(item["account_id"]))
	secUserID := strings.TrimSpace(strAny(item["sec_user_id"]))
	uniqueID := strings.TrimSpace(strAny(item["unique_id"]))
	profileURL := strings.TrimSpace(strAny(item["profile_url"]))
	accountInput := firstNonEmpty(accountID, secUserID, uniqueID, profileURL)
	score, _ := numberAny(item["benchmark_value_score"])
	if score == 0 {
		score, _ = numberAny(item["score"])
	}
	return contentBenchmarkAccount{
		AccountInput: accountInput,
		AccountID:    accountID,
		SecUserID:    secUserID,
		UniqueID:     uniqueID,
		Nickname:     strings.TrimSpace(strAny(item["nickname"])),
		Tier:         firstNonEmpty(strings.TrimSpace(strAny(item["tier"])), fallbackTier),
		Score:        score,
	}
}

type evergreenRule struct {
	ID       string
	Theme    string
	Audience string
	Pain     string
	Opinion  string
	Required []string
	Keywords []string
}

var defaultEvergreenRules = []evergreenRule{
	{
		ID:       "ai_business_workflow",
		Theme:    "老板买了 AI 工具但业务流程没有改变",
		Audience: "中小企业老板",
		Pain:     "工具很多，但获客、销售、交付流程没有承接，所以看不到结果",
		Opinion:  "AI 的价值不是多买一个工具，而是把它嵌进业务流程，形成可重复的生产系统",
		Required: []string{"老板", "企业", "业务", "流程", "工作流", "落地"},
		Keywords: []string{"AI", "ai", "工具", "老板", "企业", "业务", "流程", "工作流", "落地", "效率"},
	},
	{
		ID:       "agent_vs_tool",
		Theme:    "Agent 和普通 AI 工具的区别",
		Audience: "中小企业老板",
		Pain:     "很多人把 Agent 当成更会聊天的工具，忽略了执行、协同和流程闭环",
		Opinion:  "Agent 的关键不是回答问题，而是能在明确目标下持续调用工具、处理上下文并推进任务",
		Required: []string{"Agent", "agent", "智能体"},
		Keywords: []string{"Agent", "agent", "智能体", "工具", "执行", "协同", "平台", "架构"},
	},
	{
		ID:       "ai_marketing_acquisition",
		Theme:    "AI 时代获客和营销工作流会被重构",
		Audience: "小B老板",
		Pain:     "传统投流、私域和内容获客越来越贵，老板不知道 AI 应该接在哪个环节",
		Opinion:  "AI 获客不是自动发内容，而是从线索识别、内容测试、销售跟进到复盘的整条链路重做",
		Required: []string{"获客", "营销", "销售", "私域", "GEO", "同城", "转化"},
		Keywords: []string{"获客", "营销", "销售", "私域", "GEO", "同城", "门店", "客户", "转化"},
	},
	{
		ID:       "ai_native_org",
		Theme:    "AI-native 组织不是人人各用各的 AI",
		Audience: "企业经营者",
		Pain:     "员工各自使用 AI，组织却没有沉淀知识、权限、日志和复盘能力",
		Opinion:  "真正的 AI-native 是把 AI 工作公开化、结构化、可复用，让组织从每一次 AI 执行中学习",
		Required: []string{"组织", "内部", "权限", "日志", "Slack", "Shopify", "平台", "治理", "协同"},
		Keywords: []string{"组织", "内部", "权限", "日志", "Slack", "Shopify", "平台", "治理", "协同"},
	},
	{
		ID:       "ai_video_content_system",
		Theme:    "AI 短视频不是自动剪辑，而是内容测试系统",
		Audience: "内容创业者",
		Pain:     "很多人只关注 AI 视频能不能省剪辑成本，却没有建立选题、脚本、测试和复盘链路",
		Opinion:  "AI 视频真正改变的是内容测试成本，让小团队可以更快验证角度、标题和表达",
		Required: []string{"视频", "短视频", "短剧", "剪辑", "带货"},
		Keywords: []string{"视频", "短视频", "短剧", "剪辑", "带货", "内容", "脚本", "测试"},
	},
	{
		ID:       "ai_personal_opportunity",
		Theme:    "普通人用 AI 赚钱的核心不是工具清单",
		Audience: "副业人群",
		Pain:     "普通人容易被工具玩法和变现噱头带偏，缺少可持续交付能力",
		Opinion:  "AI 副业能跑通，关键是找到具体人群的真实问题，并把 AI 包成可交付的服务或流程",
		Required: []string{"普通人", "副业", "赚钱", "变现", "创业", "搞钱"},
		Keywords: []string{"普通人", "副业", "赚钱", "变现", "创业", "玩法", "搞钱", "交付"},
	},
	{
		ID:       "enterprise_data_agent",
		Theme:    "企业数据 Agent 难点不在写 SQL，而在业务答案可信",
		Audience: "中小企业老板",
		Pain:     "数据工具能生成查询，但常常给不出老板真正信任的业务判断",
		Opinion:  "数据 Agent 要有价值，必须理解指标口径、业务上下文和可追溯证据，而不是只会生成报表",
		Required: []string{"SQL", "Data Agent", "dataagent", "数据Agent", "数据 Agent"},
		Keywords: []string{"数据", "SQL", "指标", "报表", "分析", "答案", "可信", "Data Agent", "dataagent"},
	},
	{
		ID:       "vertical_saas_agent",
		Theme:    "垂直 SaaS 的 Agent 机会在行业场景，不在通用模型",
		Audience: "SaaS 创业者",
		Pain:     "很多团队担心通用大模型吃掉软件，但忽略了行业流程和客户现场的复杂性",
		Opinion:  "大厂做通用能力，垂直团队的机会是把行业 know-how 做成能落地的 Agent 工作流",
		Required: []string{"SaaS", "saas", "垂直", "软件", "行业", "场景", "大厂", "模型"},
		Keywords: []string{"SaaS", "saas", "垂直", "软件", "行业", "场景", "know", "大厂", "模型"},
	},
}

func mineEvergreenThemesFromSignals(signals []map[string]any, limit int) []contentEvergreenTheme {
	if limit == 0 {
		return nil
	}
	type themeHit struct {
		rule   evergreenRule
		score  float64
		count  int
		ids    []string
		titles []string
		last   string
	}
	hits := map[string]*themeHit{}
	for _, signal := range signals {
		title := strings.TrimSpace(strAny(signal["title"]))
		if title == "" {
			continue
		}
		sourceBoost := 1.0
		switch strings.TrimSpace(strAny(signal["source"])) {
		case "social_account":
			sourceBoost = 1.4
		case "websearch":
			sourceBoost = 1.2
		}
		for _, rule := range defaultEvergreenRules {
			if len(rule.Required) > 0 && len(matchedKeywords(title, rule.Required)) == 0 {
				continue
			}
			matches := matchedKeywords(title, rule.Keywords)
			if len(matches) < 2 {
				continue
			}
			hit := hits[rule.ID]
			if hit == nil {
				hit = &themeHit{rule: rule}
				hits[rule.ID] = hit
			}
			engagement := signalEngagement(signal)
			hit.score += (float64(len(matches))*1.2 + engagement) * sourceBoost
			hit.count++
			if len(hit.titles) < 5 {
				hit.titles = append(hit.titles, title)
			}
			if id := firstNonEmpty(strAny(signal["signal_id"]), strAny(signal["url"])); id != "" && len(hit.ids) < 8 {
				hit.ids = append(hit.ids, id)
			}
			if seenAt := firstNonEmpty(strAny(signal["published_at"]), strAny(signal["created_at"])); seenAt > hit.last {
				hit.last = seenAt
			}
		}
	}
	items := make([]*themeHit, 0, len(hits))
	for _, hit := range hits {
		if len(hit.titles) == 0 {
			continue
		}
		items = append(items, hit)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	out := make([]contentEvergreenTheme, 0, len(items))
	for _, hit := range items {
		out = append(out, contentEvergreenTheme{
			ThemeID:         "evg_" + hit.rule.ID,
			Theme:           hit.rule.Theme,
			Audience:        hit.rule.Audience,
			Pain:            hit.rule.Pain,
			Opinion:         hit.rule.Opinion,
			Keywords:        hit.rule.Keywords,
			Score:           round1(hit.score),
			EvidenceCount:   hit.count,
			SourceSignalIDs: hit.ids,
			SourceTitles:    hit.titles,
			LastSeenAt:      hit.last,
		})
	}
	return out
}

func matchedKeywords(text string, keywords []string) []string {
	var out []string
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}
		if strings.Contains(text, keyword) {
			out = append(out, keyword)
		}
	}
	return out
}

func signalEngagement(signal map[string]any) float64 {
	stats, _ := signal["stats"].(map[string]any)
	if len(stats) == 0 {
		return 0
	}
	likes, _ := numberAny(stats["likes"])
	comments, _ := numberAny(stats["comments"])
	shares, _ := numberAny(stats["shares"])
	collects, _ := numberAny(stats["collects"])
	value := likes + comments*2 + shares*2 + collects
	if value <= 0 {
		return 0
	}
	if value > 10000 {
		return 4
	}
	if value > 1000 {
		return 3
	}
	if value > 100 {
		return 2
	}
	return 1
}

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}

func rawSignalsFromContentSearchResult(result map[string]any) []map[string]any {
	values, ok := result["raw_signals"].([]any)
	if !ok {
		if nested, ok := result["result"].(map[string]any); ok {
			values, _ = nested["raw_signals"].([]any)
		}
	}
	out := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if item, ok := value.(map[string]any); ok {
			out = append(out, item)
		}
	}
	return out
}

func mergeContentRawSignals(signals []map[string]any, maxRaw int) ([]map[string]any, int, int) {
	seen := map[string]bool{}
	out := make([]map[string]any, 0, len(signals))
	droppedDupes := 0
	droppedEmpty := 0
	for _, signal := range signals {
		if !hasContentSignal(signal) {
			droppedEmpty++
			continue
		}
		key := contentSignalStableKey(signal)
		if key != "" && seen[key] {
			droppedDupes++
			continue
		}
		if key != "" {
			seen[key] = true
		}
		out = append(out, signal)
		if maxRaw > 0 && len(out) >= maxRaw {
			break
		}
	}
	return out, droppedDupes, droppedEmpty
}

func hasContentSignal(signal map[string]any) bool {
	return strings.TrimSpace(strAny(signal["title"])) != "" || strings.TrimSpace(strAny(signal["url"])) != ""
}

func contentSignalStableKey(signal map[string]any) string {
	for _, key := range []string{"signal_id", "url"} {
		value := strings.TrimSpace(strAny(signal[key]))
		if value != "" {
			return key + ":" + value
		}
	}
	title := strings.TrimSpace(strAny(signal["title"]))
	source := strings.TrimSpace(strAny(signal["source"]))
	if title == "" {
		return ""
	}
	return "title:" + source + ":" + title
}

func splitContentList(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，' || r == '\n' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func printContentUsage() {
	fmt.Println("luma-cli content <subcommand>")
	fmt.Println("  search       Search social or web sources")
	fmt.Println("  topic        Mine topic raw signals")
	fmt.Println("  history      List cloud-stored content artifacts for one profile")
}

func printContentSearchUsage() {
	fmt.Println("luma-cli content search <source>")
	fmt.Println("  social       Search platform signals, e.g. --keywords \"AI工具,AI智能体\"")
	fmt.Println("  social-account")
	fmt.Println("               Fetch recent videos for accounts, e.g. --accounts \"sec_uid_or_profile_url\"")
	fmt.Println("  websearch    Search web signals, e.g. --queries \"MiniMax M3 official\"")
	fmt.Println("  image        Search image candidates, e.g. --queries \"AI Agent workflow\"")
}

func printContentArtifactUsage() {
	fmt.Println("luma-cli content artifact <subcommand>")
	fmt.Println("  upload --input <artifact.json> --type <artifact_type> --profile <profile_id> [--name script.current.json] [--topic-id <id>] [--topic-title <title>]")
}

func printContentTopicUsage() {
	fmt.Println("luma-cli content topic <subcommand>")
	fmt.Println("  mine         Mine and merge raw topic signals")
	fmt.Println("  review       Review mined signals into ranked topic cards")
}
