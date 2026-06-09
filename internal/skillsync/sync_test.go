package skillsync

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func TestBuildSkillsAddArgsDefaultGlobal(t *testing.T) {
	got := BuildSkillsAddArgs("zl007700/luma-cli", "", true, true)
	want := []string{"-y", "skills", "add", "zl007700/luma-cli", "-g", "-y"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestBuildSkillsAddArgsSelective(t *testing.T) {
	got := BuildSkillsAddArgs("zl007700/luma-cli", "luma-content-ip-writing", true, true)
	want := []string{"-y", "skills", "add", "zl007700/luma-cli", "-s", "luma-content-ip-writing", "-g", "-y"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestIsVersionDrift(t *testing.T) {
	if IsVersionDrift("dev", &Stamp{Version: "0.0.1"}) {
		t.Fatal("dev builds should not report skills drift")
	}
	if IsVersionDrift("0.0.1", &Stamp{Version: "0.0.1"}) {
		t.Fatal("matching versions should not report skills drift")
	}
	if !IsVersionDrift("0.0.2", &Stamp{Version: "0.0.1"}) {
		t.Fatal("different release versions should report skills drift")
	}
}

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.0.17", "0.0.15", 1},
		{"v0.0.17", "0.0.17", 0},
		{"0.0.17", "0.0.17-beta.1", 0},
		{"0.1.0", "0.0.99", 1},
		{"0.0.1", "0.0.2", -1},
	}
	for _, tc := range cases {
		got := CompareSemver(tc.a, tc.b)
		if got != tc.want {
			t.Fatalf("CompareSemver(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestCleanupDeprecatedSkills(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", "")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	codexSkills := filepath.Join(home, ".codex", "skills")
	agentsSkills := filepath.Join(home, ".agents", "skills")
	for _, root := range []string{codexSkills, agentsSkills} {
		if err := os.MkdirAll(filepath.Join(root, "luma-content-research"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, "custom-skill"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := CleanupDeprecatedSkills()
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 2 {
		t.Fatalf("removed %d deprecated skill dirs, want 2: %#v", len(removed), removed)
	}
	for _, root := range []string{codexSkills, agentsSkills} {
		if _, err := os.Stat(filepath.Join(root, "luma-content-research")); !os.IsNotExist(err) {
			t.Fatalf("deprecated skill still exists under %s", root)
		}
		if _, err := os.Stat(filepath.Join(root, "custom-skill")); err != nil {
			t.Fatalf("custom skill should remain under %s: %v", root, err)
		}
		if !slices.Contains(removed, filepath.Join(root, "luma-content-research")) {
			t.Fatalf("removed paths missing %s: %#v", root, removed)
		}
	}
}
