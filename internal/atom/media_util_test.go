package atom

import (
	"testing"
)

func TestListFromAny(t *testing.T) {
	if got := listFromAny(nil); got != nil {
		t.Error("listFromAny(nil) should return nil")
	}
	items := listFromAny([]any{"a", "b"})
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
	if got := listFromAny("not a slice"); got != nil {
		t.Error("listFromAny(string) should return nil")
	}
}

func TestFirstPresent(t *testing.T) {
	m := map[string]any{"name": "test", "id": "123"}
	if got := firstPresent(m, "missing", "name"); got != "test" {
		t.Errorf("firstPresent = %v", got)
	}
	if got := firstPresent(m, "missing1", "missing2"); got != nil {
		t.Error("firstPresent should return nil for missing keys")
	}
}

func TestFloatFromAny(t *testing.T) {
	if got := floatFromAny(3.14); got != 3.14 {
		t.Errorf("floatFromAny(3.14) = %f", got)
	}
	if got := floatFromAny(42); got != 42 {
		t.Errorf("floatFromAny(42) = %f", got)
	}
	if got := floatFromAny("invalid"); got != 0 {
		t.Errorf("floatFromAny(invalid) = %f", got)
	}
}

func TestIntFromAny(t *testing.T) {
	if got := intFromAny(42, -1); got != 42 {
		t.Errorf("intFromAny(42) = %d", got)
	}
	if got := intFromAny(3.14, -1); got != 3 {
		t.Errorf("intFromAny(3.14) = %d", got)
	}
	if got := intFromAny("invalid", -1); got != -1 {
		t.Errorf("intFromAny(invalid, -1) = %d", got)
	}
}

func TestStringFromAny(t *testing.T) {
	if got := stringFromAny(nil); got != "" {
		t.Errorf("stringFromAny(nil) = %q", got)
	}
	if got := stringFromAny("hello"); got != "hello" {
		t.Errorf("stringFromAny(hello) = %q", got)
	}
	if got := stringFromAny(42); got != "42" {
		t.Errorf("stringFromAny(42) = %q", got)
	}
}
