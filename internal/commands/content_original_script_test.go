package commands

import "testing"

func TestOriginalScriptArticleTextPrefersArticleField(t *testing.T) {
	got := originalScriptArticleText(map[string]any{
		"article":     " final article ",
		"full_script": "fallback",
	})
	if got != "final article" {
		t.Fatalf("article text = %q", got)
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
