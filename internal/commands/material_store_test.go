package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportMaterialGroup(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())
	source := filepath.Join(t.TempDir(), "vlm_ai")
	if err := os.MkdirAll(filepath.Join(source, "items"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "group.json"), []byte(`{"name":"AI"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "items", "a.jpg"), []byte("image"), 0644); err != nil {
		t.Fatal(err)
	}

	imported, err := importMaterialGroup(source, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(imported, "items", "a.jpg")); err != nil {
		t.Fatalf("expected imported file: %v", err)
	}
	if got := resolveMaterialGroupPath("vlm_ai"); got != imported {
		t.Fatalf("resolve material group path\nwant: %s\n got: %s", imported, got)
	}
}

func TestImportMaterialGroupRejectsTraversalName(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())
	source := t.TempDir()
	if _, err := importMaterialGroup(source, "../bad", false); err == nil {
		t.Fatal("expected invalid group name error")
	}
}
