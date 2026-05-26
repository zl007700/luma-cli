package atom

import (
	"testing"
)

func TestNormalizeDouyinURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"//example.com/video.mp4", "https://example.com/video.mp4"},
		{"https://www.douyin.com/video/123", "https://www.douyin.com/video/123"},
		{"http://example.com", "http://example.com"},
		{"invalid", ""},
		{"", ""},
		{"  https://example.com  ", "https://example.com"},
	}
	for _, tt := range tests {
		got := normalizeDouyinURL(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeDouyinURL(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestNumberValue(t *testing.T) {
	if got := numberValue(3.14); got != 3.14 {
		t.Errorf("numberValue(3.14) = %f", got)
	}
	if got := numberValue(42); got != 42 {
		t.Errorf("numberValue(42) = %f", got)
	}
	if got := numberValue("invalid"); got != 0 {
		t.Errorf("numberValue(invalid) = %f", got)
	}
}

func TestTruncateUTF8(t *testing.T) {
	if got := truncateUTF8("hello", 3); got != "hel" {
		t.Errorf("truncateUTF8(hello, 3) = %q", got)
	}
	if got := truncateUTF8("你好世界", 2); got != "你好" {
		t.Errorf("truncateUTF8(你好世界, 2) = %q", got)
	}
	if got := truncateUTF8("short", 10); got != "short" {
		t.Errorf("truncateUTF8(short, 10) = %q, want short", got)
	}
	if got := truncateUTF8("text", 0); got != "text" {
		t.Errorf("truncateUTF8(text, 0) should return unchanged, got %q", got)
	}
}

