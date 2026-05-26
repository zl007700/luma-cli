package clientruntime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeJoin_NormalPath(t *testing.T) {
	dir := t.TempDir()
	got, err := safeJoin(dir, "test.txt")
	if err != nil {
		t.Fatalf("safeJoin: %v", err)
	}
	if got != filepath.Join(dir, "test.txt") {
		t.Errorf("safeJoin = %q", got)
	}
}

func TestSafeJoin_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	_, err := safeJoin(dir, "../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestSafeJoin_NestedDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	os.MkdirAll(sub, 0755)
	got, err := safeJoin(dir, "subdir/file.txt")
	if err != nil {
		t.Fatalf("safeJoin: %v", err)
	}
	if got != filepath.Join(dir, "subdir", "file.txt") {
		t.Errorf("safeJoin = %q, want subdir/file.txt", got)
	}
}

func TestRuntimeRoot(t *testing.T) {
	root, err := RuntimeRoot()
	if err != nil {
		t.Fatal(err)
	}
	if root == "" {
		t.Error("RuntimeRoot should not be empty")
	}
}

func TestResourceRoot(t *testing.T) {
	root, err := ResourceRoot()
	if err != nil {
		t.Fatal(err)
	}
	if root == "" {
		t.Error("ResourceRoot should not be empty")
	}
}
