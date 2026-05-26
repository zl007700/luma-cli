package cloud

import (
	"testing"
)

func TestBaseURL_Default(t *testing.T) {
	t.Setenv("LUMA_API_URL", "")
	got := BaseURL()
	if got != DefaultBaseURL {
		t.Errorf("BaseURL() = %q, want %q", got, DefaultBaseURL)
	}
}

func TestBaseURL_EnvOverride(t *testing.T) {
	t.Setenv("LUMA_API_URL", "https://custom.example.com")
	got := BaseURL()
	if got != "https://custom.example.com" {
		t.Errorf("BaseURL() = %q, want custom URL", got)
	}
}

func TestDefaultBaseURL_NotEmpty(t *testing.T) {
	if DefaultBaseURL == "" {
		t.Error("DefaultBaseURL should not be empty")
	}
}
