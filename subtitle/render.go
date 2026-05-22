package subtitle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

		wrapped := wrapTextByPixelWidth(text, resX-opts.MarginL-opts.MarginR, opts.FontSize)
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
	escaped := escapeASSText(text)
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

func wrapTextByPixelWidth(text string, maxWidth int, fontSize int) string {
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
	charWidth := float64(fontSize) * 0.72
	maxChars := int(float64(maxWidth) / charWidth)
	if maxChars < 12 {
		maxChars = 12
	}

	var lines []string
	for start := 0; start < len(runes); start += maxChars {
		end := start + maxChars
		if end > len(runes) {
			end = len(runes)
		}
		lines = append(lines, string(runes[start:end]))
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
