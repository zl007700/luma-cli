package subtitle

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Segment represents a subtitle segment with timing and text.
type Segment struct {
	SegID          int     // segment index
	Start          float64 // start time in seconds
	End            float64 // end time in seconds
	Text           string  // display text
	HighlightWord  string  // keyword to highlight (LLM-assigned)
	EffectType     string  // "blur_in", "bounce_in", "scale_pop", "rotate_pop", "wave_bounce", or ""
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
	prompt := buildSplitPrompt(text, maxCharsPerSeg, persona)

	response, err := llmClient.Chat(prompt)
	if err != nil {
		// fallback to punctuation
		segs := splitByPunctuation(text, maxCharsPerSeg)
		return segs, buildSentenceGroups(segs), nil
	}

	segments, sentenceGroups, err := parseSplitOutput(response, text, maxCharsPerSeg)
	if err != nil {
		segs := splitByPunctuation(text, maxCharsPerSeg)
		return segs, buildSentenceGroups(segs), nil
	}

	return segments, sentenceGroups, nil
}

// buildSplitPrompt creates the LLM prompt for sentence-aware subtitle splitting.
func buildSplitPrompt(text string, maxChars int, persona string) string {
	personaHint := ""
	if persona != "" {
		personaHint = "\n\n人设要求：" + persona
	}
	return `你是字幕切分助手。

任务：在不改动原文任何字的前提下，把文本切成适合竖屏短视频展示的字幕段，并同时标出句子边界。` + personaHint + `

硬性规则：
1. 只能复制原文并换行，禁止增删改任何字。
2. 每个 seg 尽量不超过 ` + strconv.Itoa(maxChars) + ` 个字。
3. 同一句话如果过长，可以拆成多个连续 seg。
4. 严禁把前一句尾巴和后一句开头放进同一个 seg。
5. 不同句子之间用空行分隔，也就是两个换行 \n\n。
6. 同一句中的多个 seg 之间只用单个换行 \n 分隔。
7. 一个句子不要包含超过3个seg, 句子不要太长。
8. 输出只能是纯文本，不要 JSON，不要编号，不要解释。

输出示例：
第一句的第一段
第一句的第二段

第二句的第一段
第二句的第二段

原文：
` + text
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
			// Further split long lines
			pieces := splitLongLine(line, maxChars)
			lineSegs = append(lineSegs, pieces...)
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