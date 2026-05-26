package subtitle

import (
	"testing"
)

func TestFallbackEvenAlign(t *testing.T) {
	segs := []Segment{
		{Text: "第一句"},
		{Text: "第二句"},
		{Text: "第三句"},
	}
	result := FallbackEvenAlign(segs, 9.0)
	if len(result) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(result))
	}
	for i, s := range result {
		if s.End <= s.Start {
			t.Errorf("segment %d: end (%f) <= start (%f)", i, s.End, s.Start)
		}
	}

	// Empty segments
	result = FallbackEvenAlign(nil, 10)
	if len(result) != 0 {
		t.Errorf("expected 0 segments for nil input, got %d", len(result))
	}
}

func TestAutoSizeParams(t *testing.T) {
	fontSize, marginL, marginR, marginV := AutoSizeParams(1080, 1920)
	if fontSize <= 0 {
		t.Errorf("fontSize should be positive, got %d", fontSize)
	}
	if marginL <= 0 || marginR <= 0 || marginV <= 0 {
		t.Errorf("margins should be positive: L=%d R=%d V=%d", marginL, marginR, marginV)
	}

	fontSize2, _, _, _ := AutoSizeParams(720, 1280)
	if fontSize2 >= fontSize {
		t.Errorf("smaller video should have smaller font: %d >= %d", fontSize2, fontSize)
	}
}
