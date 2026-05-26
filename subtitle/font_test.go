package subtitle

import (
	"testing"
)

func TestHexToRGBA(t *testing.T) {
	r, g, b, a := HexToRGBA("#FF0000")
	if r != 255 || g != 0 || b != 0 || a != 255 {
		t.Errorf("HexToRGBA(#FF0000) = (%d,%d,%d,%d), want (255,0,0,255)", r, g, b, a)
	}

	r, g, b, a = HexToRGBA("#00FF00")
	if r != 0 || g != 255 || b != 0 || a != 255 {
		t.Errorf("HexToRGBA(#00FF00) = (%d,%d,%d,%d), want (0,255,0,255)", r, g, b, a)
	}

	r, g, b, a = HexToRGBA("336699")
	if r != 0x33 || g != 0x66 || b != 0x99 || a != 255 {
		t.Errorf("HexToRGBA(336699) = (%d,%d,%d,%d)", r, g, b, a)
	}

	r, g, b, a = HexToRGBA("invalid")
	if r != 253 || g != 253 || b != 255 || a != 255 {
		t.Errorf("HexToRGBA(invalid) = (%d,%d,%d,%d), expected default subtitle color", r, g, b, a)
	}
}

func TestRGBAFromHex(t *testing.T) {
	c := RGBAFromHex("#FFFFFF")
	if c.R != 255 || c.G != 255 || c.B != 255 || c.A != 255 {
		t.Errorf("RGBAFromHex(#FFFFFF) = %+v", c)
	}

	c = RGBAFromHex("FFD95A")
	if c.R != 0xFF || c.G != 0xD9 || c.B != 0x5A {
		t.Errorf("RGBAFromHex(FFD95A) = %+v", c)
	}
}

func TestStripHash(t *testing.T) {
	if s := stripHash("#abc"); s != "abc" {
		t.Errorf("stripHash(#abc) = %q", s)
	}
	if s := stripHash("abc"); s != "abc" {
		t.Errorf("stripHash(abc) = %q", s)
	}
}
