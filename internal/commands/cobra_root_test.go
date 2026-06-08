package commands

import "testing"

func TestNormalizeGlobalArgsStripsJSONFlag(t *testing.T) {
	runtimeOpts = runtimeOptions{}
	got := normalizeGlobalArgs([]string{"--json", "profile", "get", "ai_biz", "--json=true"})
	want := []string{"profile", "get", "ai_biz"}
	if !runtimeOpts.JSON {
		t.Fatal("runtimeOpts.JSON = false, want true")
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
