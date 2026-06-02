package subtitle

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// WriteSRT writes segments to an SRT subtitle file.
func WriteSRT(segments []Segment, outputPath string) error {
	var lines []string
	index := 1
	for _, seg := range segments {
		text := strings.TrimSpace(seg.Text)
		if text == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%d", index))
		lines = append(lines, fmt.Sprintf("%s --> %s", formatSRTTime(seg.Start), formatSRTTime(seg.End)))
		lines = append(lines, text)
		lines = append(lines, "")
		index++
	}
	return os.WriteFile(outputPath, []byte(strings.Join(lines, "\n")), 0644)
}

// WriteASS writes segments to an ASS subtitle file with optional keyword highlight.
// effectType is currently embedded as ASS override tags (simplified, no animated effects yet).
func WriteASS(segments []Segment, outputPath string, opts ASSOptions) error {
	if opts.FontSize <= 0 {
		opts.FontSize = 48
	}
	if opts.MarginL <= 0 {
		opts.MarginL = 60
	}
	if opts.MarginR <= 0 {
		opts.MarginR = 60
	}
	if opts.MarginV <= 0 {
		opts.MarginV = 100
	}
	if opts.Outline <= 0 {
		opts.Outline = 2.0
	}
	if opts.Spacing <= 0 {
		opts.Spacing = 2.0
	}
	resX := opts.PlayResX
	resY := opts.PlayResY
	if resX <= 0 {
		resX = 1080
	}
	if resY <= 0 {
		resY = 1920
	}
	fontName := opts.FontName
	fontInfo := loadASSFontInfo(opts.FontPath, opts.FontSize)
	if fontInfo.Family != "" {
		fontName = fontInfo.Family
		opts.Bold = fontInfo.Bold
		opts.Italic = fontInfo.Italic
	}
	if fontName == "" {
		fontName = "Microsoft YaHei"
	}

	var header []string
	header = append(header, "[Script Info]")
	header = append(header, "ScriptType: v4.00+")
	header = append(header, fmt.Sprintf("PlayResX: %d", resX))
	header = append(header, fmt.Sprintf("PlayResY: %d", resY))
	header = append(header, "WrapStyle: 2")
	header = append(header, "ScaledBorderAndShadow: yes")
	header = append(header, "YCbCr Matrix: TV.601")
	header = append(header, "")
	header = append(header, "[V4+ Styles]")

	primary := hexToASSColor(opts.Color, "&H00FFFFFF")
	outlineColor := hexToASSColor(opts.StrokeColor, "&H00000000")
	backColor := hexToASSColor(opts.BackColor, "&H64000000")
	highlightColor := hexToASSColor(opts.HighlightColor, "&H005AD9FF")

	styleLine := fmt.Sprintf(
		"Style: Default,%s,%d,%s,&H000000FF,%s,%s,%d,%d,0,0,100,100,%.1f,0,1,%.1f,%.1f,2,%d,%d,%d,1",
		fontName, opts.FontSize, primary, outlineColor, backColor,
		opts.Bold, opts.Italic, opts.Spacing, opts.Outline, opts.Shadow,
		opts.MarginL, opts.MarginR, opts.MarginV,
	)
	header = append(header, "Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding")
	header = append(header, styleLine)
	header = append(header, "")
	header = append(header, "[Events]")
	header = append(header, "Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text")

	var body []string
	for _, seg := range segments {
		text := strings.TrimSpace(seg.Text)
		if text == "" {
			continue
		}

		wrapped := wrapTextByPixelWidth(text, resX-opts.MarginL-opts.MarginR, opts.FontSize, fontInfo)
		displayText := escapeASSText(wrapped)

		// Apply keyword highlight (with effect if any)
		displayText = renderWithHighlight(wrapped, seg.HighlightWord, opts.FontSize, opts.HighlightScale, highlightColor, seg.EffectType)

		start := formatASSTime(seg.Start)
		end := formatASSTime(seg.End)
		body = append(body, fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s", start, end, displayText))
	}

	var allLines []string
	allLines = append(allLines, header...)
	allLines = append(allLines, body...)
	allLines = append(allLines, assFontSection(fontName, opts.FontPath)...)
	return os.WriteFile(outputPath, []byte(strings.Join(allLines, "\n")), 0644)
}

// BurnSubtitles burns ASS subtitles into video using ffmpeg libass filter.
func BurnSubtitles(videoPath, assPath, outputPath string, fontDir, ffmpegPath string) error {
	if ffmpegPath == "" {
		ffmpegPath = findFFmpeg()
	}
	escapedASS := escapeFilterPath(assPath)
	escapedFontDir := escapeFilterPath(fontDir)
	filterValue := fmt.Sprintf("subtitles='%s':fontsdir='%s'", escapedASS, escapedFontDir)

	cmd := exec.Command(ffmpegPath,
		"-y", "-i", videoPath,
		"-vf", filterValue,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "18",
		"-c:a", "copy",
		outputPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ASSOptions controls ASS subtitle generation.
type ASSOptions struct {
	PlayResX       int
	PlayResY       int
	FontName       string
	FontPath       string
	FontSize       int
	Color          string
	StrokeColor    string
	BackColor      string
	HighlightColor string
	HighlightScale float64
	MarginL        int
	MarginR        int
	MarginV        int
	Outline        float64
	Shadow         float64
	Spacing        float64
	Bold           int
	Italic         int
}

// ---- Internal helpers ----

func formatSRTTime(seconds float64) string {
	totalMs := int(seconds * 1000)
	hours := totalMs / 3600000
	minutes := (totalMs % 3600000) / 60000
	secs := (totalMs % 60000) / 1000
	ms := totalMs % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, ms)
}

func formatASSTime(seconds float64) string {
	centiseconds := int(seconds * 100)
	hours := centiseconds / 360000
	minutes := (centiseconds % 360000) / 6000
	secs := (centiseconds % 6000) / 100
	cs := centiseconds % 100
	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, secs, cs)
}

func hexToASSColor(hex string, defaultColor string) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return defaultColor
	}
	rr := hex[0:2]
	gg := hex[2:4]
	bb := hex[4:6]
	return "&H00" + strings.ToUpper(bb+gg+rr)
}

func escapeASSText(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "{", "\\{"), "}", "\\}")
}

func escapeFilterPath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.ReplaceAll(path, ":", "\\:")
	path = strings.ReplaceAll(path, "'", "\\'")
	return path
}

func renderWithHighlight(text, keyword string, fontSize int, scale float64, highlightColor string, effectType string) string {
	keyword = strings.TrimSpace(keyword)
	escaped := escapeASSText(text)
	if keyword == "" {
		return escaped
	}
	pos := strings.Index(text, keyword)
	if pos < 0 {
		return escaped
	}
	before := escapeASSText(text[:pos])
	key := escapeASSText(keyword)
	after := escapeASSText(text[pos+len(keyword):])
	highlightSize := int(float64(fontSize) * scale)
	if highlightSize < fontSize+2 {
		highlightSize = fontSize + 2
	}
	result := before + "{\\b1\\fs" + fmt.Sprintf("%d", highlightSize) + "\\1c" + highlightColor + "}" + key + "{\\r}" + after

	// Append effect animation
	result = appendEffect(result, effectType)
	return result
}

func appendEffect(text, effectType string) string {
	if effectType == "" || effectType == "none" {
		return text
	}
	duration := 200 // ms
	var tag string
	switch effectType {
	case "blur_in":
		tag = "\\t(0," + fmt.Sprintf("%d", duration) + ",1.0\\blur10\\blur0)"
	case "bounce_in":
		tag = "\\t(0," + fmt.Sprintf("%d", duration) + ",0\\fscx120\\fscy120\\fscx100\\fscy100)"
	case "scale_pop":
		tag = "\\t(0," + fmt.Sprintf("%d", duration) + ",0\\fscx50\\fscy50\\fscx110\\fscy110\\fscx100\\fscy100)"
	case "rotate_pop":
		tag = "\\t(0," + fmt.Sprintf("%d", duration) + ",0\\fr-20\\fr5\\fr0)"
	case "wave_bounce":
		tag = "\\t(0," + fmt.Sprintf("%d", duration) + ",0\\fscx90\\fscy110\\fscx105\\fscy100)"
	default:
		return text
	}
	return text + "{" + tag + "}"
}

type assFontInfo struct {
	Family string
	Bold   int
	Italic int
	Font   *sfnt.Font
}

func loadASSFontInfo(fontPath string, fontSize int) assFontInfo {
	fontPath = strings.TrimSpace(fontPath)
	if fontPath == "" {
		return assFontInfo{}
	}
	data, err := os.ReadFile(fontPath)
	if err != nil {
		return assFontInfo{}
	}
	family, style := readFontNames(data)
	parsed, err := sfnt.Parse(data)
	if err != nil {
		collection, collErr := sfnt.ParseCollection(data)
		if collErr != nil || collection.NumFonts() == 0 {
			return assFontInfo{Family: family, Bold: fontBold(style), Italic: fontItalic(style)}
		}
		parsed, err = collection.Font(0)
		if err != nil {
			return assFontInfo{Family: family, Bold: fontBold(style), Italic: fontItalic(style)}
		}
	}
	var buf sfnt.Buffer
	if family == "" {
		family, _ = parsed.Name(&buf, sfnt.NameIDFamily)
		if family == "" {
			family, _ = parsed.Name(&buf, sfnt.NameIDTypographicFamily)
		}
	}
	if style == "" {
		style, _ = parsed.Name(&buf, sfnt.NameIDSubfamily)
	}
	return assFontInfo{Family: strings.TrimSpace(family), Bold: fontBold(style), Italic: fontItalic(style), Font: parsed}
}

func fontBold(style string) int {
	style = strings.ToLower(style)
	if strings.Contains(style, "bold") || strings.Contains(style, "black") || strings.Contains(style, "heavy") || strings.Contains(style, "semibold") {
		return -1
	}
	return 0
}

func fontItalic(style string) int {
	style = strings.ToLower(style)
	if strings.Contains(style, "italic") || strings.Contains(style, "oblique") {
		return -1
	}
	return 0
}

func readFontNames(data []byte) (family, style string) {
	offset := uint32(0)
	if len(data) >= 12 && string(data[0:4]) == "ttcf" {
		if len(data) < 16 || binary.BigEndian.Uint32(data[8:12]) == 0 {
			return "", ""
		}
		offset = binary.BigEndian.Uint32(data[12:16])
	}
	names := readNameTable(data, offset)
	family = firstNonEmpty(names[16], names[1])
	style = firstNonEmpty(names[17], names[2])
	return family, style
}

func readNameTable(data []byte, fontOffset uint32) map[uint16]string {
	out := map[uint16]string{}
	if uint64(fontOffset)+12 > uint64(len(data)) {
		return out
	}
	base := int(fontOffset)
	numTables := int(binary.BigEndian.Uint16(data[base+4 : base+6]))
	tableDir := base + 12
	var nameOffset, nameLength uint32
	for i := 0; i < numTables; i++ {
		entry := tableDir + i*16
		if entry+16 > len(data) {
			return out
		}
		if string(data[entry:entry+4]) == "name" {
			nameOffset = binary.BigEndian.Uint32(data[entry+8 : entry+12])
			nameLength = binary.BigEndian.Uint32(data[entry+12 : entry+16])
			break
		}
	}
	if nameOffset == 0 || uint64(nameOffset)+uint64(nameLength) > uint64(len(data)) || nameLength < 6 {
		return out
	}
	table := data[nameOffset : nameOffset+nameLength]
	count := int(binary.BigEndian.Uint16(table[2:4]))
	stringOffset := int(binary.BigEndian.Uint16(table[4:6]))
	for i := 0; i < count; i++ {
		rec := 6 + i*12
		if rec+12 > len(table) {
			break
		}
		platformID := binary.BigEndian.Uint16(table[rec : rec+2])
		nameID := binary.BigEndian.Uint16(table[rec+6 : rec+8])
		if nameID != 1 && nameID != 2 && nameID != 16 && nameID != 17 {
			continue
		}
		length := int(binary.BigEndian.Uint16(table[rec+8 : rec+10]))
		off := stringOffset + int(binary.BigEndian.Uint16(table[rec+10:rec+12]))
		if off < 0 || length <= 0 || off+length > len(table) {
			continue
		}
		value := decodeFontName(platformID, table[off:off+length])
		if value != "" && out[nameID] == "" {
			out[nameID] = value
		}
	}
	return out
}

func decodeFontName(platformID uint16, raw []byte) string {
	if platformID == 0 || platformID == 3 {
		u16 := make([]uint16, 0, len(raw)/2)
		for i := 0; i+1 < len(raw); i += 2 {
			u16 = append(u16, binary.BigEndian.Uint16(raw[i:i+2]))
		}
		return strings.TrimSpace(string(utf16.Decode(u16)))
	}
	return strings.TrimSpace(string(raw))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func assFontSection(fontName, fontPath string) []string {
	fontPath = strings.TrimSpace(fontPath)
	if fontPath == "" {
		return nil
	}
	data, err := os.ReadFile(fontPath)
	if err != nil || len(data) == 0 {
		return nil
	}
	return []string{"", "[Fonts]", "fontname: " + fontName + ".ttf", encodeFontForASS(data)}
}

func encodeFontForASS(data []byte) string {
	var chars []rune
	for i := 0; i < len(data); i += 3 {
		chunk := data[i:min(i+3, len(data))]
		n := len(chunk)
		b := []byte{0, 0, 0}
		copy(b, chunk)
		values := []byte{
			b[0] >> 2,
			((b[0] & 0x03) << 4) | (b[1] >> 4),
			((b[1] & 0x0F) << 2) | (b[2] >> 6),
			b[2] & 0x3F,
		}
		outCount := map[int]int{1: 2, 2: 3, 3: 4}[n]
		for _, value := range values[:outCount] {
			chars = append(chars, rune(value+33))
		}
	}
	var lines []string
	for start := 0; start < len(chars); start += 80 {
		end := min(start+80, len(chars))
		lines = append(lines, string(chars[start:end]))
	}
	return strings.Join(lines, "\n")
}

func wrapTextByPixelWidth(text string, maxWidth int, fontSize int, fontInfo assFontInfo) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= 18 {
		return string(runes)
	}
	if maxWidth <= 0 {
		maxWidth = 800
	}
	if fontSize <= 0 {
		fontSize = 48
	}
	widthOf := func(r rune) float64 {
		if fontInfo.Font != nil {
			var buf sfnt.Buffer
			glyph, err := fontInfo.Font.GlyphIndex(&buf, r)
			if err == nil && glyph != 0 {
				advance, err := fontInfo.Font.GlyphAdvance(&buf, glyph, fixed.I(fontSize), font.HintingNone)
				if err == nil && advance > 0 {
					return float64(advance) / 64.0
				}
			}
		}
		if r >= 0x4E00 && r <= 0x9FFF {
			return float64(fontSize) * 0.98
		}
		return float64(fontSize) * 0.55
	}

	var lines []string
	var current strings.Builder
	currentWidth := 0.0
	for _, r := range runes {
		w := widthOf(r)
		if current.Len() > 0 && currentWidth+w > float64(maxWidth) {
			lines = append(lines, current.String())
			current.Reset()
			currentWidth = 0
		}
		current.WriteRune(r)
		currentWidth += w
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return strings.Join(lines, "\\N")
}

// ClearDir removes all files in a directory (used for cleanup).
func ClearDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		os.Remove(filepath.Join(dir, entry.Name()))
	}
}
