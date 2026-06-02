package commands

import "testing"

func TestResolveFontSizeScalesDefaultLayoutToVideoHeight(t *testing.T) {
	fontSize, marginV := resolveFontSize(80, 500, 1280, false, false)
	if fontSize != 53 {
		t.Fatalf("fontSize = %d, want 53", fontSize)
	}
	if marginV != 333 {
		t.Fatalf("marginV = %d, want 333", marginV)
	}
}

func TestResolveFontSizeKeepsExplicitLayout(t *testing.T) {
	fontSize, marginV := resolveFontSize(80, 500, 1280, true, true)
	if fontSize != 80 {
		t.Fatalf("fontSize = %d, want 80", fontSize)
	}
	if marginV != 500 {
		t.Fatalf("marginV = %d, want 500", marginV)
	}
}

func TestResolveSideMarginScalesDefaultLayoutToVideoWidth(t *testing.T) {
	got := resolveSideMargin(60, 720, false)
	if got != 40 {
		t.Fatalf("sideMargin = %d, want 40", got)
	}
}

func TestResolveSideMarginKeepsExplicitLayout(t *testing.T) {
	got := resolveSideMargin(60, 720, true)
	if got != 60 {
		t.Fatalf("sideMargin = %d, want 60", got)
	}
}
