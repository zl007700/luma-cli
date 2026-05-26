package cloud

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGuessMimeType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"video.mp4", "video/mp4"},
		{"audio.mp3", "audio/mpeg"},
		{"sound.wav", "audio/wav"},
		{"movie.avi", "video/x-msvideo"},
		{"clip.mov", "video/quicktime"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"image.png", "image/png"},
		{"file.unknown", "application/octet-stream"},
		{"noext", "application/octet-stream"},
	}
	for _, tt := range tests {
		got := guessMimeType(tt.path)
		if got != tt.expected {
			t.Errorf("guessMimeType(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}

func TestGuessResourceType(t *testing.T) {
	tests := []struct {
		mime     string
		expected string
	}{
		{"video/mp4", "video"},
		{"video/quicktime", "video"},
		{"video/x-msvideo", "video"},
		{"audio/mpeg", "audio"},
		{"audio/wav", "audio"},
		{"audio/mp3", "audio"},
		{"image/jpeg", "image"},
		{"image/png", "image"},
		{"text/plain", "other"},
		{"application/octet-stream", "other"},
	}
	for _, tt := range tests {
		got := guessResourceType(tt.mime)
		if got != tt.expected {
			t.Errorf("guessResourceType(%q) = %q, want %q", tt.mime, got, tt.expected)
		}
	}
}

func TestFileContentHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := FileContentHash(path)
	if err != nil {
		t.Fatalf("FileContentHash: %v", err)
	}
	if len(hash) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars: %s", len(hash), hash)
	}

	// Same content = same hash
	path2 := filepath.Join(dir, "test2.txt")
	os.WriteFile(path2, []byte("hello world"), 0644)
	hash2, _ := FileContentHash(path2)
	if hash != hash2 {
		t.Error("same content should produce same hash")
	}

	// Different content = different hash
	os.WriteFile(path2, []byte("different"), 0644)
	hash3, _ := FileContentHash(path2)
	if hash == hash3 {
		t.Error("different content should produce different hash")
	}
}

func TestFileContentHash_NotFound(t *testing.T) {
	_, err := FileContentHash("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
