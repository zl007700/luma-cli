package cloud

import "testing"

func TestFilterAssetItemsKeepsRequestedKindGroupAndScope(t *testing.T) {
	items := []AssetItem{
		{AssetID: "system_voice", Kind: "voice", GroupName: "common", Scope: "system"},
		{AssetID: "user_voice", Kind: "voice", GroupName: "common", Scope: "user"},
		{AssetID: "system_template", Kind: "template", GroupName: "common", Scope: "system"},
		{AssetID: "other_group", Kind: "voice", GroupName: "demo", Scope: "system"},
	}

	got := filterAssetItems(items, "voice", "common", "system")

	if len(got) != 1 {
		t.Fatalf("filtered length = %d, want 1: %#v", len(got), got)
	}
	if got[0].AssetID != "system_voice" {
		t.Fatalf("filtered asset = %q, want system_voice", got[0].AssetID)
	}
}
