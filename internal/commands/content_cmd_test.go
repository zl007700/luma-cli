package commands

import (
	"strings"
	"testing"

	"github.com/luma-cli/lumer-cli/internal/cmdutil"
)

func TestBuildContentTopicMineResultMergesAndCountsSignals(t *testing.T) {
	social := map[string]any{
		"source": "social",
		"raw_signals": []any{
			map[string]any{"signal_id": "sig_1", "title": "AI工具爆了", "source": "douyin"},
			map[string]any{"signal_id": "sig_1", "title": "AI工具爆了", "source": "douyin"},
			map[string]any{},
		},
	}
	websearch := map[string]any{
		"source": "websearch",
		"raw_signals": []any{
			map[string]any{"url": "https://example.com/m3", "title": "MiniMax M3 发布", "source": "websearch"},
		},
	}
	account := map[string]any{
		"source": "social_account",
		"raw_signals": []any{
			map[string]any{"signal_id": "acct_1", "title": "企业 AI Agent 落地", "source": "social_account"},
		},
	}

	result := buildContentTopicMineResult(
		[]string{"AI工具"},
		[]string{"MiniMax M3"},
		"7d",
		100,
		8,
		social,
		account,
		websearch,
	)

	if result.Counts.SocialRaw != 3 {
		t.Fatalf("SocialRaw = %d, want 3", result.Counts.SocialRaw)
	}
	if result.Counts.WebRaw != 1 {
		t.Fatalf("WebRaw = %d, want 1", result.Counts.WebRaw)
	}
	if result.Counts.SocialAccountRaw != 1 {
		t.Fatalf("SocialAccountRaw = %d, want 1", result.Counts.SocialAccountRaw)
	}
	if result.Counts.MergedRaw != 3 {
		t.Fatalf("MergedRaw = %d, want 3", result.Counts.MergedRaw)
	}
	if result.Counts.DroppedDupes != 1 {
		t.Fatalf("DroppedDupes = %d, want 1", result.Counts.DroppedDupes)
	}
	if result.Counts.DroppedEmpty != 1 {
		t.Fatalf("DroppedEmpty = %d, want 1", result.Counts.DroppedEmpty)
	}
	if result.Counts.EvergreenThemes == 0 || len(result.EvergreenThemes) == 0 {
		t.Fatal("expected evergreen themes to be mined")
	}
}

func TestExtractBenchmarkAccountsPrefersSeedAccountsAndDedupes(t *testing.T) {
	payload := map[string]any{
		"recommended_seed_accounts": []any{
			map[string]any{"account_id": "uid_a", "sec_user_id": "sec_a", "nickname": "A", "tier": "A", "benchmark_value_score": 8.8},
		},
		"observation_accounts": []any{
			map[string]any{"account_id": "uid_a", "sec_user_id": "sec_a", "nickname": "Duplicate", "tier": "B"},
			map[string]any{"account_id": "uid_b", "nickname": "B", "tier": "B", "score": 6.1},
		},
	}

	got := extractBenchmarkAccounts(payload, 10)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %#v", len(got), got)
	}
	if got[0].AccountInput != "uid_a" || got[0].Tier != "A" || got[0].Score != 8.8 {
		t.Fatalf("unexpected first account: %#v", got[0])
	}
	if got[1].AccountInput != "uid_b" || got[1].Tier != "B" || got[1].Score != 6.1 {
		t.Fatalf("unexpected second account: %#v", got[1])
	}
}

func TestMineEvergreenThemesFromSignalsBuildsReusableThemes(t *testing.T) {
	signals := []map[string]any{
		{
			"signal_id": "s1",
			"source":    "social_account",
			"title":     "企业 AI Agent 落地，老板真正缺的是业务工作流",
			"stats":     map[string]any{"likes": 1200},
		},
		{
			"signal_id": "s2",
			"source":    "social",
			"title":     "AI 获客和营销自动化正在改变本地商家的销售转化",
			"stats":     map[string]any{"likes": 300},
		},
	}

	themes := mineEvergreenThemesFromSignals(signals, 5)
	if len(themes) == 0 {
		t.Fatal("expected evergreen themes")
	}
	found := false
	for _, theme := range themes {
		if theme.ThemeID == "evg_ai_business_workflow" {
			found = true
			if theme.EvidenceCount == 0 || theme.Score <= 0 {
				t.Fatalf("bad workflow theme: %#v", theme)
			}
		}
	}
	if !found {
		t.Fatalf("workflow theme not found: %#v", themes)
	}
}

func TestBuildContentTopicReviewResultCreatesRankedCards(t *testing.T) {
	mine := contentTopicMineResult{
		ProfileID: "ai_saas_agent_founder",
		EvergreenThemes: []contentEvergreenTheme{
			{
				ThemeID:       "evg_agent_vs_tool",
				Theme:         "Agent 和普通 AI 工具的区别",
				Pain:          "很多人把 Agent 当成更会聊天的工具",
				Opinion:       "Agent 的关键不是回答问题，而是执行闭环",
				Keywords:      []string{"Agent", "智能体", "工具", "执行"},
				Score:         20,
				EvidenceCount: 2,
				SourceTitles:  []string{"AI圈重要转折点！智能体开始变现了"},
			},
		},
		RawSignals: []map[string]any{
			{
				"signal_id":   "s1",
				"source":      "social_account",
				"title":       "AI圈重要转折点！智能体开始变现了",
				"author_name": "A",
				"stats":       map[string]any{"likes": 1200},
			},
			{
				"signal_id":   "s2",
				"source":      "social",
				"title":       "NBA首秀AI人形机器人",
				"author_name": "B",
			},
		},
	}
	profile := &lumaProfile{
		ID:       "ai_saas_agent_founder",
		Identity: "AI SaaS Agent 创业者",
		Audience: []string{"中小企业老板"},
		Stance:   []string{"Agent的价值不在炫技，而在替老板把重复决策和执行跑起来"},
	}

	review := buildContentTopicReviewResult(mine, profile, "mine.json", 5, "auto")
	if len(review.TopicCards) == 0 {
		t.Fatal("expected topic cards")
	}
	if review.TopicCards[0].Status != "shortlisted" {
		t.Fatalf("first card status = %q", review.TopicCards[0].Status)
	}
	if review.Counts.RejectedSignals != 1 {
		t.Fatalf("RejectedSignals = %d, want 1", review.Counts.RejectedSignals)
	}
	if review.TopicCards[0].TotalScore <= 0 {
		t.Fatalf("bad score: %#v", review.TopicCards[0])
	}
	if review.TopicCards[0].FormatFit.Recommended == "" {
		t.Fatalf("missing format fit: %#v", review.TopicCards[0])
	}
	if review.TopicCards[0].PublicEntry == "" {
		t.Fatalf("missing public entry: %#v", review.TopicCards[0])
	}
	if review.TopicCards[0].WhyGeneralAudienceCares == "" {
		t.Fatalf("missing general audience reason: %#v", review.TopicCards[0])
	}
	if review.TopicCards[0].AudienceFilterTurn == "" {
		t.Fatalf("missing audience filter turn: %#v", review.TopicCards[0])
	}
	if len(review.TopicCards[0].ObjectiveFacts) == 0 {
		t.Fatalf("missing objective facts: %#v", review.TopicCards[0])
	}
}

func TestSplitContentListDedupesCommaAndNewlineValues(t *testing.T) {
	got := splitContentList("AI工具，AI智能体,AI工具\nAI创业")
	want := []string{"AI工具", "AI智能体", "AI创业"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestContentSearchQueriesKeepsSingleQueryPunctuation(t *testing.T) {
	args := cmdutil.Parse([]string{"--query", "老板用 AI 没效果，问题不在工具，而在业务流程"})
	got := contentSearchQueries(args)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if got[0] != "老板用 AI 没效果，问题不在工具，而在业务流程" {
		t.Fatalf("query = %q", got[0])
	}
}

func TestValidateWebsearchDateRange(t *testing.T) {
	for _, value := range []string{"24h", "7d", " 7D "} {
		got, err := validateWebsearchDateRange(value)
		if err != nil {
			t.Fatalf("validateWebsearchDateRange(%q) returned error: %v", value, err)
		}
		if got != strings.ToLower(strings.TrimSpace(value)) {
			t.Fatalf("validateWebsearchDateRange(%q) = %q", value, got)
		}
	}

	if _, err := validateWebsearchDateRange("365d"); err == nil {
		t.Fatal("expected 365d to be rejected")
	}
}
