package subtitle

import (
	"image/color"
)

// FontFace is a simple font face interface for rendering
type FontFace struct {
	Size    int
	Ascent  int
	Descent int
	Advance func(r rune) int
}

// BasicFont returns a basic font face for rendering
func BasicFont() FontFace {
	return FontFace{
		Size:    13,
		Ascent:  11,
		Descent: 2,
		Advance: func(r rune) int {
			if r >= 0x4E00 && r <= 0x9FFF {
				return 13
			}
			return 7
		},
	}
}

// HexToRGBA converts hex color string to RGBA
func HexToRGBA(hex string) (r, g, b, a uint8) {
	hex = stripHash(hex)
	if len(hex) != 6 {
		return 253, 253, 255, 255
	}
	return hexToUint8(hex[0:2]), hexToUint8(hex[2:4]), hexToUint8(hex[4:6]), 255
}

func stripHash(s string) string {
	if len(s) > 0 && s[0] == '#' {
		return s[1:]
	}
	return s
}

func hexToUint8(s string) uint8 {
	var val uint8 = 0
	for _, c := range s {
		val <<= 4
		switch {
		case c >= '0' && c <= '9':
			val += uint8(c - '0')
		case c >= 'a' && c <= 'f':
			val += uint8(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			val += uint8(c - 'A' + 10)
		}
	}
	return val
}

// RGBAFromHex creates color.RGBA from hex string
func RGBAFromHex(hex string) color.RGBA {
	r, g, b, a := HexToRGBA(hex)
	return color.RGBA{R: r, G: g, B: b, A: a}
}