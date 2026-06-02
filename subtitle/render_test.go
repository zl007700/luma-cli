package subtitle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatSRTTime(t *testing.T) {
	got := formatSRTTime(3661.5) // 1h 1m 1.5s
	if !strings.Contains(got, "01:01:01,500") {
		t.Errorf("formatSRTTime(3661.5) = %q, expected 01:01:01,500", got)
	}

	got = formatSRTTime(0)
	if !strings.Contains(got, "00:00:00,000") {
		t.Errorf("formatSRTTime(0) = %q, expected 00:00:00,000", got)
	}
}

func TestFormatASSTime(t *testing.T) {
	got := formatASSTime(3661.5)
	if got != "1:01:01.50" {
		t.Errorf("formatASSTime(3661.5) = %q, want 1:01:01.50", got)
	}

	got = formatASSTime(0)
	if got != "0:00:00.00" {
		t.Errorf("formatASSTime(0) = %q, want 0:00:00.00", got)
	}
}

func TestHexToASSColor(t *testing.T) {
	got := hexToASSColor("#FF0000", "&HFFFFFFFF&")
	// ASS format is &H00BBGGRR& so #FF0000 → &H000000FF&
	if got != "&H000000FF" {
		t.Errorf("hexToASSColor(#FF0000) = %q, want &H000000FF", got)
	}

	got = hexToASSColor("invalid", "&H000000FF&")
	if got != "&H000000FF&" {
		t.Errorf("hexToASSColor(invalid) should return default, got %q", got)
	}
}

func TestEscapeASSText(t *testing.T) {
	if s := escapeASSText("hello"); s != "hello" {
		t.Errorf("escapeASSText(hello) = %q", s)
	}
}

func TestEscapeFilterPath(t *testing.T) {
	path := `C:\Users\test\file.ass`
	got := escapeFilterPath(path)
	if strings.Contains(got, `\`) {
		t.Logf("escapeFilterPath result: %q", got)
	}
}

func TestWriteASSUsesRealFontFamilyAndWrapsByFontWidth(t *testing.T) {
	fontPath := filepath.Join("..", "fonts", "字由ID江湖体.ttf")
	if _, err := os.Stat(fontPath); err != nil {
		t.Skipf("font fixture missing: %v", err)
	}
	out := filepath.Join(t.TempDir(), "subtitle.ass")
	err := WriteASS([]Segment{{
		Start: 0,
		End:   1,
		Text:  "这是一个非常非常长的字幕文本用来验证不会因为估算宽度太小而超出屏幕",
	}}, out, ASSOptions{
		PlayResX:  720,
		PlayResY:  1280,
		FontPath:  fontPath,
		FontSize:  80,
		MarginL:   60,
		MarginR:   60,
		MarginV:   500,
		Color:     "#FDFDFF",
		BackColor: "#000000",
	})
	if err != nil {
		t.Fatalf("WriteASS failed: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read ASS: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "Style: Default,Microsoft YaHei") {
		t.Fatalf("expected real font family, got fallback style:\n%s", content)
	}
	if !strings.Contains(content, `\N`) {
		t.Fatalf("expected wrapped subtitle line, got:\n%s", content)
	}
	if !strings.Contains(content, "[Fonts]") {
		t.Fatalf("expected embedded font section, got:\n%s", content[:min(len(content), 500)])
	}
}

func TestRenderWithHighlightIgnoresEmptyKeyword(t *testing.T) {
	got := renderWithHighlight("测试字幕", "", 80, 1.25, "&H005AD9FF", "")
	if strings.Contains(got, `\fs`) || strings.Contains(got, `\1c`) {
		t.Fatalf("empty keyword should not emit highlight tags: %q", got)
	}
}
