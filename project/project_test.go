package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAndLoad(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())
	tmpDir := t.TempDir()

	p, err := Create("test-video", tmpDir)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify directory structure
	for _, sub := range subDirs {
		dir := filepath.Join(p.Path, sub)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("subdirectory not created: %s", sub)
		}
	}

	// Verify project.json exists
	metaPath := filepath.Join(p.Path, "project.json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Fatal("project.json not created")
	}

	// Load and verify
	loaded, err := Load(p.Path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Name != "test-video" {
		t.Errorf("expected name 'test-video', got '%s'", loaded.Name)
	}
	if loaded.Path != p.Path {
		t.Errorf("expected path '%s', got '%s'", p.Path, loaded.Path)
	}
	if loaded.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestCreateDuplicate(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())
	tmpDir := t.TempDir()

	_, err := Create("dup", tmpDir)
	if err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	_, err = Create("dup", tmpDir)
	if err == nil {
		t.Error("expected error for duplicate project name")
	}
}

func TestCreateEmptyName(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())
	tmpDir := t.TempDir()
	_, err := Create("", tmpDir)
	if err == nil {
		t.Error("expected error for empty project name")
	}
}

func TestCreateDefaultBasePath(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("LUMA_CONFIG_DIR", configDir)

	// Create with empty basePath should use ~/.luma/projects/
	p, err := Create("auto-path", "")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	expected := filepath.Join(configDir, "projects", "auto-path")
	if p.Path != expected {
		t.Errorf("expected path '%s', got '%s'", expected, p.Path)
	}

	// Cleanup
	os.RemoveAll(p.Path)
}

func TestSaveAndAddStep(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())
	tmpDir := t.TempDir()

	p, err := Create("step-test", tmpDir)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = p.AddStep(Step{
		Command:   "subtitle",
		Timestamp: p.CreatedAt,
		Input:     "input.mp4",
		Output:    "output.mp4",
	})
	if err != nil {
		t.Fatalf("AddStep failed: %v", err)
	}

	loaded, err := Load(p.Path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded.History) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(loaded.History))
	}
	if loaded.History[0].Command != "subtitle" {
		t.Errorf("expected command 'subtitle', got '%s'", loaded.History[0].Command)
	}
	if loaded.History[0].Input != "input.mp4" {
		t.Errorf("expected input 'input.mp4', got '%s'", loaded.History[0].Input)
	}
}

func TestSubDir(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())
	tmpDir := t.TempDir()
	p, err := Create("subdir-test", tmpDir)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	expected := filepath.Join(p.Path, DirSubtitles)
	if p.SubDir(DirSubtitles) != expected {
		t.Errorf("expected '%s', got '%s'", expected, p.SubDir(DirSubtitles))
	}
}

func TestCleanTmp(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())
	tmpDir := t.TempDir()
	p, err := Create("clean-test", tmpDir)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Create a temp file in tmp/
	tmpFile := filepath.Join(p.SubDir(DirTmp), "test.tmp")
	os.WriteFile(tmpFile, []byte("hello"), 0644)

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("temp file should exist before clean")
	}

	if err := p.CleanTmp(); err != nil {
		t.Fatalf("CleanTmp failed: %v", err)
	}

	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("temp file should be removed after clean")
	}
}

func TestSetActiveAndGetActive(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())

	tmpDir := t.TempDir()

	p1, err := Create("active1", tmpDir)
	if err != nil {
		t.Fatalf("Create active1 failed: %v", err)
	}

	// First project should be auto-activated
	active, err := GetActiveProject()
	if err != nil {
		t.Fatalf("GetActiveProject failed: %v", err)
	}
	if active == nil || active.Name != "active1" {
		t.Errorf("expected active project 'active1', got '%v'", active)
	}

	// Create second project
	_, err = Create("active2", tmpDir)
	if err != nil {
		t.Fatalf("Create active2 failed: %v", err)
	}

	// Switch active project
	if err := SetActiveProject("active2"); err != nil {
		t.Fatalf("SetActiveProject failed: %v", err)
	}

	active, err = GetActiveProject()
	if err != nil {
		t.Fatalf("GetActiveProject failed: %v", err)
	}
	if active == nil || active.Name != "active2" {
		t.Errorf("expected active project 'active2', got '%v'", active)
	}

	// Original project should still be loadable
	loaded1, err := Load(p1.Path)
	if err != nil {
		t.Fatalf("Load active1 failed: %v", err)
	}
	if loaded1.Name != "active1" {
		t.Errorf("expected 'active1', got '%s'", loaded1.Name)
	}

}

func TestListProjects(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())

	tmpDir := t.TempDir()

	_, err := Create("list1", tmpDir)
	if err != nil {
		t.Fatalf("Create list1 failed: %v", err)
	}
	_, err = Create("list2", tmpDir)
	if err != nil {
		t.Fatalf("Create list2 failed: %v", err)
	}

	projects, err := ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}

	names := map[string]bool{}
	for _, p := range projects {
		names[p.Name] = true
	}
	if !names["list1"] || !names["list2"] {
		t.Errorf("expected both list1 and list2, got %v", names)
	}
}

func TestFindByName(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())

	tmpDir := t.TempDir()

	_, err := Create("findme", tmpDir)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	p, err := FindByName("findme")
	if err != nil {
		t.Fatalf("FindByName failed: %v", err)
	}
	if p.Name != "findme" {
		t.Errorf("expected 'findme', got '%s'", p.Name)
	}

	_, err = FindByName("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent project")
	}
}

func TestActiveProjectName(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())

	// No active project initially
	name, err := ActiveProjectName()
	if err != nil {
		t.Fatalf("ActiveProjectName failed: %v", err)
	}
	if name != "" {
		t.Errorf("expected empty name, got '%s'", name)
	}

	tmpDir := t.TempDir()
	_, err = Create("name-test", tmpDir)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	name, err = ActiveProjectName()
	if err != nil {
		t.Fatalf("ActiveProjectName failed: %v", err)
	}
	if name != "name-test" {
		t.Errorf("expected 'name-test', got '%s'", name)
	}
}

func TestGetActiveProjectNone(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())

	active, err := GetActiveProject()
	if err != nil {
		t.Fatalf("GetActiveProject failed: %v", err)
	}
	if active != nil {
		t.Error("expected nil when no active project")
	}
}
