package subtitle

import (
	"fmt"
	"regexp"
	"strings"
)

// Segment represents a subtitle segment with timing and text.
type Segment struct {
	SegID         int     // segment index
	Start         float64 // start time in seconds
	End           float64 // end time in seconds
	Text          string  // display text
	HighlightWord string  // keyword to highlight (LLM-assigned)
	EffectType    string  // "blur_in", "bounce_in", "scale_pop", "rotate_pop", "wave_bounce", or ""
}

// SentenceGroup groups segments that belong to the same spoken sentence.
type SentenceGroup struct {
	StartSegID int
	EndSegID   int
	Text       string
}

// SplitByLLM uses LLM to split raw text into subtitle segments respecting sentence boundaries.
// Returns segments and sentence groupings. Falls back to punctuation split on error.
func SplitByLLM(text string, maxCharsPerSeg int, llmClient *LLMClient, persona string) ([]Segment, []SentenceGroup, error) {
	segments, sentenceGroups, err := llmClient.SplitSubtitle(text, maxCharsPerSeg, persona)
	if err != nil {
		segs := splitByPunctuation(text, maxCharsPerSeg)
		return segs, buildSentenceGroups(segs), nil
	}

	// Client-side safety net: the backend LLM may return segments longer than
	// maxCharsPerSeg. Split any overlong segment so subtitles never overflow the screen.
	segments = enforceMaxChars(segments, maxCharsPerSeg)
	segments = normalizeSegments(segments)

	return segments, sentenceGroups, nil
}

// parseSplitOutput parses the LLM's plain-text split output into segments and sentence groups.
func parseSplitOutput(content, originalText string, maxChars int) ([]Segment, []SentenceGroup, error) {
	normalized := strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n")
	normalized = strings.TrimSpace(normalized)

	// Split by double newlines (sentence boundaries)
	blocks := regexp.MustCompile(`\n\s*\n`).Split(normalized, -1)

	var segments []Segment
	var sentenceGroups []SentenceGroup
	segID := 0

	for _, block := range blocks {
		lines := strings.Split(block, "\n")
		var lineSegs []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
						lineSegs = append(lineSegs, line)
		}
		if len(lineSegs) == 0 {
			continue
		}

		startSegID := segID
		for _, segText := range lineSegs {
			segments = append(segments, Segment{SegID: segID, Text: segText})
			segID++
		}
		sentenceGroups = append(sentenceGroups, SentenceGroup{
			StartSegID: startSegID,
			EndSegID:   segID - 1,
			Text:       strings.Join(lineSegs, ""),
		})
	}

	// Verify text integrity
	merged := hanziOnly(mergedSegmentsText(segments))
	original := hanziOnly(originalText)
	if len(segments) == 0 || merged != original {
		return nil, nil, ErrTextMismatch
	}

	return segments, sentenceGroups, nil
}

// mergedSegmentsText concatenates all segment texts.
func mergedSegmentsText(segs []Segment) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(s.Text)
	}
	return b.String()
}

// ErrTextMismatch is returned when LLM output doesn't match original text.
var ErrTextMismatch = fmt.Errorf("subtitle split output does not match original text")

// hanziOnly removes non-Chinese characters for comparison.
func hanziOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// enforceMaxChars splits any segment longer than maxChars into shorter pieces.
// Segment IDs are re-indexed. Sentence groups are NOT updated here because they
// carry their own text and are used independently for cloud alignment.
func enforceMaxChars(segments []Segment, maxChars int) []Segment {
	out := make([]Segment, 0, len(segments))
	segID := 0
	for _, seg := range segments {
		runes := []rune(seg.Text)
		if len(runes) <= maxChars {
			seg.SegID = segID
			out = append(out, seg)
			segID++
			continue
		}
		for _, piece := range splitLongLine(seg.Text, maxChars) {
			out = append(out, Segment{
				SegID: segID,
				Start: seg.Start,
				End:   seg.End,
				Text:  piece,
			})
			segID++
		}
	}
	return out
}

func normalizeSegments(segments []Segment) []Segment {
	out := make([]Segment, 0, len(segments))
	segID := 0
	for _, seg := range segments {
		seg.Text = trimSegmentPunctuation(seg.Text)
		if strings.TrimSpace(seg.Text) == "" {
			continue
		}
		seg.SegID = segID
		out = append(out, seg)
		segID++
	}
	return out
}

func trimSegmentPunctuation(text string) string {
	text = strings.TrimSpace(text)
	isPunctuation := func(r rune) bool {
		switch r {
		case '。', '，', '、', '？', '！', '；', '：', '.', ',', '?', '!', ';', ':':
			return true
		default:
			return false
		}
	}
	text = strings.TrimLeftFunc(text, isPunctuation)
	return strings.TrimRightFunc(text, isPunctuation)
}

// splitLongLine splits a line into chunks of maxChars.
func splitLongLine(line string, maxChars int) []string {
	if len([]rune(line)) <= maxChars {
		return []string{line}
	}
	var result []string
	var current strings.Builder
	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		current.WriteRune(runes[i])
		if current.Len() >= maxChars && i < len(runes)-1 {
			result = append(result, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}

// splitByPunctuation is the fallback when LLM fails.
func splitByPunctuation(text string, maxChars int) []Segment {
	re := regexp.MustCompile(`[。！？；：，、]+`)
	parts := re.Split(text, -1)
	var segments []Segment
	segID := 0
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pieces := splitLongLine(part, maxChars)
		for _, p := range pieces {
			p = trimSegmentPunctuation(p)
			if p == "" {
				continue
			}
			segments = append(segments, Segment{SegID: segID, Text: p})
			segID++
		}
	}
	return segments
}

// buildSentenceGroups groups consecutive segments into sentence units.
func buildSentenceGroups(segs []Segment) []SentenceGroup {
	if len(segs) == 0 {
		return nil
	}
	// Simple: treat each segment as its own group for fallback
	var groups []SentenceGroup
	for _, seg := range segs {
		groups = append(groups, SentenceGroup{
			StartSegID: seg.SegID,
			EndSegID:   seg.SegID,
			Text:       seg.Text,
		})
	}
	return groups
}
