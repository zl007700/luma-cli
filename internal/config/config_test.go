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
	t.Setenv("LUMA_API_URL", "")

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

func TestSaveEnvironmentPreservesCardKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LUMA_CONFIG_DIR", dir)
	t.Setenv("LUMA_CARD_KEY", "")
	t.Setenv("LUMA_API_URL", "")

	if err := SaveCardKey("test-card-key"); err != nil {
		t.Fatalf("SaveCardKey failed: %v", err)
	}
	if err := SaveEnvironment("test", "https://backend.example.invalid/"); err != nil {
		t.Fatalf("SaveEnvironment failed: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.CardKey != "test-card-key" {
		t.Fatalf("expected card key to be preserved, got %q", cfg.CardKey)
	}
	if cfg.APIURL != "https://backend.example.invalid" {
		t.Fatalf("expected normalized api url, got %q", cfg.APIURL)
	}
	if cfg.Environment != "test" {
		t.Fatalf("expected test environment, got %q", cfg.Environment)
	}
}

func TestPendingDeviceAuthLifecycle(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LUMA_CONFIG_DIR", dir)
	t.Setenv("LUMA_CARD_KEY", "")
	t.Setenv("LUMA_API_URL", "")

	if err := SavePendingDeviceAuth("device-123", "ABCD", "https://app.example/authorize?code=ABCD"); err != nil {
		t.Fatalf("SavePendingDeviceAuth failed: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.PendingAuthDeviceCode != "device-123" {
		t.Fatalf("expected pending device code, got %q", cfg.PendingAuthDeviceCode)
	}
	if cfg.PendingAuthUserCode != "ABCD" {
		t.Fatalf("expected pending user code, got %q", cfg.PendingAuthUserCode)
	}
	if cfg.PendingAuthVerifyURL != "https://app.example/authorize?code=ABCD" {
		t.Fatalf("expected pending verify URL, got %q", cfg.PendingAuthVerifyURL)
	}

	if err := SaveCardKey("test-card-key"); err != nil {
		t.Fatalf("SaveCardKey failed: %v", err)
	}
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.CardKey != "test-card-key" {
		t.Fatalf("expected card key, got %q", cfg.CardKey)
	}
	if cfg.PendingAuthDeviceCode != "" || cfg.PendingAuthUserCode != "" || cfg.PendingAuthVerifyURL != "" {
		t.Fatalf("expected pending authorization to be cleared, got %#v", cfg)
	}
}

func TestAPIBaseURLUsesConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LUMA_CONFIG_DIR", dir)
	t.Setenv("LUMA_CARD_KEY", "")
	t.Setenv("LUMA_API_URL", "")

	if err := SaveEnvironment("test", "https://backend.example.invalid"); err != nil {
		t.Fatalf("SaveEnvironment failed: %v", err)
	}
	if got := APIBaseURL("https://api.pikgeo.com"); got != "https://backend.example.invalid" {
		t.Fatalf("expected config api url, got %q", got)
	}
}

func TestAPIBaseURLEnvTakesPrecedence(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())
	t.Setenv("LUMA_API_URL", "https://override.example.com/")

	if err := SaveEnvironment("test", "https://backend.example.invalid"); err != nil {
		t.Fatalf("SaveEnvironment failed: %v", err)
	}
	if got := APIBaseURL("https://api.pikgeo.com"); got != "https://override.example.com" {
		t.Fatalf("expected env api url, got %q", got)
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
