package subtitle

import (
	"testing"
)

func TestHanziOnly(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"你好世界", "你好世界"},
		{"hello世界", "世界"},
		{"ABC123测试DEF", "测试"},
		{"   ", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := hanziOnly(tt.input)
		if got != tt.expected {
			t.Errorf("hanziOnly(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMergedSegmentsText(t *testing.T) {
	segs := []Segment{
		{Text: "第一句"},
		{Text: "第二句"},
		{Text: "第三句"},
	}
	got := mergedSegmentsText(segs)
	if got != "第一句第二句第三句" {
		t.Errorf("mergedSegmentsText = %q", got)
	}
}

func TestSplitLongLine(t *testing.T) {
	parts := splitLongLine("这是一句测试文本", 6)
	if len(parts) < 2 {
		t.Errorf("expected at least 2 parts for 6-char max, got %d", len(parts))
	}
	for _, p := range parts {
		if len([]rune(p)) > 6 {
			t.Errorf("part %q exceeds max 6 chars", p)
		}
	}

	// Short text should not be split
	parts = splitLongLine("短文本", 10)
	if len(parts) != 1 {
		t.Errorf("expected 1 part, got %d", len(parts))
	}
}

func TestEnforceMaxChars(t *testing.T) {
	segs := []Segment{
		{Text: "这是一个非常长的句子需要被拆分"},
	}
	result := enforceMaxChars(segs, 6)
	for _, s := range result {
		if len([]rune(s.Text)) > 6 {
			t.Errorf("segment %q exceeds max 6 chars", s.Text)
		}
	}
}

func TestNormalizeSegmentsTrimsTrailingPunctuation(t *testing.T) {
	segs := []Segment{
		{SegID: 7, Text: "还是觉得闷？"},
		{SegID: 8, Text: "空气不流动。"},
		{SegID: 9, Text: "三件事：占不占地方、"},
		{SegID: 10, Text: "、办公室"},
		{SegID: 11, Text: "风速能不能调"},
	}
	result := normalizeSegments(segs)
	expected := []string{
		"还是觉得闷",
		"空气不流动",
		"三件事：占不占地方",
		"办公室",
		"风速能不能调",
	}
	if len(result) != len(expected) {
		t.Fatalf("expected %d segments, got %d", len(expected), len(result))
	}
	for i, want := range expected {
		if result[i].SegID != i {
			t.Errorf("segment %d got SegID %d, want %d", i, result[i].SegID, i)
		}
		if result[i].Text != want {
			t.Errorf("segment %d text = %q, want %q", i, result[i].Text, want)
		}
	}
}

func TestSplitByPunctuation(t *testing.T) {
	segs := splitByPunctuation("第一句。第二句！第三句？", 20)
	if len(segs) < 3 {
		t.Errorf("expected at least 3 segments, got %d", len(segs))
	}
}

func TestBuildSentenceGroups(t *testing.T) {
	segs := []Segment{
		{Text: "第一句", Start: 0, End: 1},
		{Text: "第二句", Start: 1, End: 2},
		{Text: "第三句", Start: 2, End: 3},
	}
	groups := buildSentenceGroups(segs)
	if len(groups) == 0 {
		t.Error("expected at least 1 sentence group")
	}
	for _, g := range groups {
		if g.Text == "" {
			t.Error("sentence group text should not be empty")
		}
	}
}
