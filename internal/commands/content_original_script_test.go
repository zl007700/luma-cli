package commands

import (
	"strings"
	"testing"
)

func TestOriginalScriptArticleTextPrefersArticleField(t *testing.T) {
	got := originalScriptArticleText(map[string]any{
		"article":     " final article ",
		"full_script": "fallback",
	})
	if got != "final article" {
		t.Fatalf("article text = %q", got)
	}
}

func TestCompactOriginalScriptSourcesDropsRawSignal(t *testing.T) {
	got := compactOriginalScriptSources([]map[string]any{
		{
			"id":          "web_1",
			"source_type": "web",
			"title":       "Example",
			"summary":     "Useful summary",
			"url":         "https://example.com",
			"raw_signal":  map[string]any{"huge": "payload"},
		},
	}, 1)

	if len(got) != 1 {
		t.Fatalf("sources len = %d", len(got))
	}
	if _, ok := got[0]["raw_signal"]; ok {
		t.Fatalf("raw_signal should be dropped: %#v", got[0])
	}
}

func TestFallbackOriginalScriptDetailPlanEnforcesBudget(t *testing.T) {
	sources := []map[string]any{
		{"id": "web_1", "source_type": "web", "url": "https://example.com/1"},
		{"id": "web_2", "source_type": "web", "url": "https://example.com/2"},
		{"id": "web_3", "source_type": "web", "url": "https://example.com/3"},
		{"id": "dy_1", "source_type": "social", "url": "https://www.douyin.com/video/1"},
		{"id": "dy_2", "source_type": "social", "url": "https://www.douyin.com/video/2"},
	}

	got := fallbackOriginalScriptDetailPlan(sources, nil)

	selected := got["selected"].([]any)
	if len(selected) != 3 {
		t.Fatalf("selected len = %d", len(selected))
	}
	if selected[0].(map[string]any)["action"] != "url_read" || selected[1].(map[string]any)["action"] != "url_read" {
		t.Fatalf("web actions = %#v", selected)
	}
	if selected[2].(map[string]any)["action"] != "download_asr" {
		t.Fatalf("douyin action = %#v", selected)
	}
}

func TestLikelyReadableOriginalScriptURLFiltersBlockedHosts(t *testing.T) {
	if likelyReadableOriginalScriptURL("https://x.com/example/article/1") {
		t.Fatal("x.com should be filtered for fallback url-read")
	}
	if likelyReadableOriginalScriptURL("https://zhuanlan.zhihu.com/p/1") {
		t.Fatal("zhihu should be filtered for fallback url-read")
	}
	if !likelyReadableOriginalScriptURL("https://example.com/article") {
		t.Fatal("ordinary https URL should be readable candidate")
	}
}

func TestCompactOriginalScriptMemoryTrimsHistoryReview(t *testing.T) {
	memory := map[string]any{
		"content_history": map[string]any{
			"value": map[string]any{
				"items": []any{
					map[string]any{
						"run_id":       "run_1",
						"topic_hint":   "hint",
						"article_head": "article head",
						"topic":        map[string]any{"public_entry": "entry", "thesis": "thesis"},
						"final_review": map[string]any{"decision": "pass", "total_score": 8.2, "review_text": strings.Repeat("review ", 200)},
					},
				},
			},
		},
	}

	got := compactOriginalScriptMemory(memory, 1)

	history := got["content_history"].(map[string]any)
	items := history["items"].([]any)
	review := items[0].(map[string]any)["final_review"].(map[string]any)
	if len(strAny(review["summary"])) > 220 {
		t.Fatalf("review summary not compacted: %d", len(strAny(review["summary"])))
	}
}

func TestResolvedOriginalScriptTopicRequiresWritableTopic(t *testing.T) {
	got := resolvedOriginalScriptTopic(map[string]any{
		"decision":           "reject",
		"public_entry":       "none",
		"revision_direction": "try another angle",
		"selected_topic":     map[string]any{},
	})
	if got != nil {
		t.Fatalf("expected nil topic, got %#v", got)
	}
}

func TestResolvedOriginalScriptTopicUsesSelectedTopic(t *testing.T) {
	got := resolvedOriginalScriptTopic(map[string]any{
		"decision":       "pass",
		"duplicate_risk": "low",
		"selected_topic": map[string]any{
			"public_entry": "A concrete scene",
			"thesis":       "A clear thesis",
		},
	})
	if got == nil || got["public_entry"] != "A concrete scene" {
		t.Fatalf("topic = %#v", got)
	}
	if _, ok := got["topic_review"].(map[string]any); !ok {
		t.Fatalf("topic_review missing: %#v", got)
	}
}

func TestOriginalScriptFinalReviewGateBlocksResearchRoute(t *testing.T) {
	ok, reason := originalScriptFinalReviewPassed(map[string]any{
		"decision":    "pass",
		"total_score": 8.5,
		"scores": map[string]any{
			"credibility": map[string]any{"score": 8.0},
		},
		"review_route": map[string]any{"next_step": "research"},
	})
	if ok || !strings.Contains(reason, "research") {
		t.Fatalf("gate = %v, %s", ok, reason)
	}
}

func TestOriginalScriptFinalReviewGateBlocksLowCredibility(t *testing.T) {
	ok, reason := originalScriptFinalReviewPassed(map[string]any{
		"decision":    "pass",
		"total_score": 8.0,
		"scores": map[string]any{
			"credibility": map[string]any{"score": 6.5},
		},
	})
	if ok || !strings.Contains(reason, "credibility") {
		t.Fatalf("gate = %v, %s", ok, reason)
	}
}

func TestOriginalScriptFinalReviewGateAllowsCleanPass(t *testing.T) {
	ok, reason := originalScriptFinalReviewPassed(map[string]any{
		"decision":    "pass",
		"total_score": 8.2,
		"scores": map[string]any{
			"credibility": map[string]any{"score": 7.2},
		},
		"review_route": map[string]any{"next_step": "finish"},
	})
	if !ok {
		t.Fatalf("gate blocked clean pass: %s", reason)
	}
}

func TestOriginalScriptArticleTextFallsBackToScriptFields(t *testing.T) {
	got := originalScriptArticleText(map[string]any{
		"revise_version": " revised ",
	})
	if got != "revised" {
		t.Fatalf("article text = %q", got)
	}
}

func TestFirstRunesDoesNotSplitChineseTextByBytes(t *testing.T) {
	got := firstRunes("光标在两个窗口之间疯狂跳跃", 4)
	if got != "光标在两" {
		t.Fatalf("firstRunes = %q", got)
	}
}

func TestOriginalScriptAvoidRecentExtractsHistory(t *testing.T) {
	memory := map[string]any{
		"content_history": map[string]any{
			"value": map[string]any{
				"items": []any{
					map[string]any{
						"run_id":       "run_1",
						"topic_hint":   "AI工具不是收藏夹",
						"article_head": "收藏夹塞了100个网址",
						"topic": map[string]any{
							"public_entry":   "收藏夹焦虑",
							"duplicate_risk": "high",
						},
					},
				},
			},
		},
	}

	got := originalScriptAvoidRecent(memory, 5)

	if len(got) != 1 {
		t.Fatalf("avoid len = %d", len(got))
	}
	if got[0]["article_head"] != "收藏夹塞了100个网址" {
		t.Fatalf("article_head = %#v", got[0])
	}
	if got[0]["public_entry"] != "收藏夹焦虑" {
		t.Fatalf("public_entry = %#v", got[0])
	}
}

func TestDefaultOriginalScriptMemoryValueHasItems(t *testing.T) {
	got := defaultOriginalScriptMemoryValue("evergreen_topics")
	if _, ok := got["items"].([]any); !ok {
		t.Fatalf("items missing: %#v", got)
	}
	if got["schema_version"] != 1 {
		t.Fatalf("schema_version = %#v", got["schema_version"])
	}
}

func TestOriginalScriptResearchQueriesIncludeTopicHint(t *testing.T) {
	got := originalScriptWebQueries(map[string]any{"identity": "AI工具实测"}, "AI工具不是收藏夹")
	if len(got) == 0 || got[0] != "AI工具不是收藏夹" {
		t.Fatalf("queries = %#v", got)
	}
}

func TestOriginalScriptSignalsNormalizeWebAndSocialResults(t *testing.T) {
	result := map[string]any{
		"raw_signals": []any{
			map[string]any{
				"signal_id": "web_1",
				"title":     "Demo",
				"url":       "https://example.com",
				"summary":   "summary",
			},
		},
	}
	got := originalScriptSignals(result, "web")
	if len(got) != 1 {
		t.Fatalf("signals len = %d", len(got))
	}
	if got[0]["id"] != "web_1" || got[0]["expansion_hint"] != "url_read" {
		t.Fatalf("signal = %#v", got[0])
	}
}

func TestCompactOriginalScriptExpandedDetailsTruncatesContent(t *testing.T) {
	expanded := map[string]any{
		"items": []any{
			map[string]any{
				"id":     "web_1",
				"status": "expanded",
				"action": "url_read",
				"source": map[string]any{"id": "web_1", "title": "Demo", "summary": "Summary", "url": "https://example.com"},
				"content": map[string]any{
					"content":       "一二三四五六七八九十",
					"content_chars": 10,
				},
			},
		},
	}

	got := compactOriginalScriptExpandedDetails(expanded, 4)

	items := got["items"].([]map[string]any)
	content := items[0]["content"].(map[string]any)
	if content["content"] != "一二三四" {
		t.Fatalf("content = %#v", content)
	}
}
