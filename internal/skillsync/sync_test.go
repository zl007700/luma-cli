package skillsync

import (
	"reflect"
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
	got := BuildSkillsAddArgs("zl007700/luma-cli", "luma-content-research", true, true)
	want := []string{"-y", "skills", "add", "zl007700/luma-cli", "-s", "luma-content-research", "-g", "-y"}
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
