package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveAndLoadCardKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LUMA_CONFIG_DIR", dir)
	t.Setenv("LUMA_CARD_KEY", "")

	if err := SaveCardKey("test-card-key"); err != nil {
		t.Fatalf("SaveCardKey failed: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config")
	}
	if cfg.CardKey != "test-card-key" {
		t.Fatalf("expected card key, got %q", cfg.CardKey)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dir, "config.json"))
		if err != nil {
			t.Fatalf("stat config failed: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("expected config mode 0600, got %v", info.Mode().Perm())
		}
	}
}

func TestEnvCardKeyTakesPrecedence(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())
	t.Setenv("LUMA_CARD_KEY", "env-card-key")

	if err := SaveCardKey("file-card-key"); err != nil {
		t.Fatalf("SaveCardKey failed: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.CardKey != "env-card-key" {
		t.Fatalf("expected env key, got %q", cfg.CardKey)
	}
}

func TestMaskKeyHandlesShortKeys(t *testing.T) {
	if got := MaskKey("abc"); got != "***" {
		t.Fatalf("expected masked short key, got %q", got)
	}
	if got := MaskKey("1234567890"); got != "1234...7890" {
		t.Fatalf("expected masked long key, got %q", got)
	}
}
