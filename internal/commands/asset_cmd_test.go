package commands

import "testing"

func TestNormalizeAssetGroupName(t *testing.T) {
	cases := map[string]string{
		"avatar":  "roles",
		"avatars": "roles",
		"role":    "roles",
		"voices":  "voice",
		"roles":   "roles",
	}
	for input, want := range cases {
		if got := normalizeAssetGroupName(input); got != want {
			t.Fatalf("normalizeAssetGroupName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMergeAssetItemsDedupesCommonAssets(t *testing.T) {
	primary := []any{
		map[string]any{"object_key": "roles/a.mp4"},
	}
	common := []any{
		map[string]any{"object_key": "roles/a.mp4"},
		map[string]any{"object_key": "common/roles/b.mp4"},
	}
	got := mergeAssetItems(primary, common)
	if len(got) != 2 {
		t.Fatalf("expected 2 merged items, got %d", len(got))
	}
}
