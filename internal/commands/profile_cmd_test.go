package commands

import "testing"

func TestSplitProfileListCleansAndDedupes(t *testing.T) {
	got := splitProfileList("小B老板,副业人群,小B老板\n内容创业者")
	want := []string{"小B老板", "副业人群", "内容创业者"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSplitProfileListKeepsChineseCommaInsidePhrases(t *testing.T) {
	got := splitProfileList("AI不是工具，是生产系统;会用AI的人更值钱")
	want := []string{"AI不是工具，是生产系统", "会用AI的人更值钱"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestValidateProfileRequiresCoreFields(t *testing.T) {
	profile := lumaProfile{
		ID:       "ai_biz",
		Identity: "AI行业动态点评者",
		Audience: []string{"小B老板"},
		Stance:   []string{"AI不是工具，是生产系统"},
		Avoid:    []string{"纯新闻搬运"},
	}
	if err := validateProfile(profile); err != nil {
		t.Fatalf("validateProfile returned error: %v", err)
	}
	profile.Avoid = nil
	if err := validateProfile(profile); err == nil {
		t.Fatal("validateProfile returned nil for missing avoid")
	}
}

func TestAppendProfileExtra(t *testing.T) {
	got := appendProfileExtra("已有信息\n", "新增信息")
	want := "已有信息\n\n新增信息\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestUpsertProfileAsset(t *testing.T) {
	first := profileAsset{ID: "asset_a", ObjectKey: "key_a", Name: "old"}
	second := profileAsset{ID: "asset_b", ObjectKey: "key_a", Name: "new"}
	items := upsertProfileAsset([]profileAsset{first}, second)
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}
	if items[0].Name != "new" || items[0].ID != "asset_b" {
		t.Fatalf("asset not replaced: %#v", items[0])
	}
}

func TestSplitAssetFileList(t *testing.T) {
	got := splitAssetFileList("a.md;b.png，c.mp4\na.md")
	want := []string{"a.md", "b.png", "c.mp4"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
