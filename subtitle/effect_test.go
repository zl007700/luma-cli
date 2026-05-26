package subtitle

import (
	"testing"
)

func TestEase_BackOut(t *testing.T) {
	e := &Ease{}
	if got := e.BackOut(0); got != 0 {
		t.Errorf("BackOut(0) = %f, want 0", got)
	}
	if got := e.BackOut(1); got < 0.99 {
		t.Errorf("BackOut(1) = %f, want ~1", got)
	}
}

func TestEase_CubicOut(t *testing.T) {
	e := &Ease{}
	if got := e.CubicOut(0); got != 0 {
		t.Errorf("CubicOut(0) = %f, want 0", got)
	}
	if got := e.CubicOut(1); got != 1 {
		t.Errorf("CubicOut(1) = %f, want 1", got)
	}
}

func TestEase_Bounce(t *testing.T) {
	e := &Ease{}
	if got := e.Bounce(0); got != 0 {
		t.Errorf("Bounce(0) = %f, want 0", got)
	}
	if got := e.Bounce(1); got < 0.99 {
		t.Errorf("Bounce(1) = %f, want ~1", got)
	}
	if mid := e.Bounce(0.5); mid < 0 || mid > 1 {
		t.Errorf("Bounce(0.5) = %f, out of [0,1]", mid)
	}
}
