package subtitle

import (
	"encoding/json"
	"fmt"

	"github.com/luma-cli/lumer-cli/cloud"
)

// LLMClient wraps backend-owned agent prompt abilities.
type LLMClient struct {
	BaseURL string
	APIKey  string
	Model   string
}

// NewLLMClient creates a subtitle LLM client with defaults.
func NewLLMClient(apiKey, baseURL, model string) *LLMClient {
	if baseURL == "" {
		baseURL = cloud.BaseURL()
	}
	if model == "" {
		model = "deepseek-ai/DeepSeek-V3.2"
	}
	return &LLMClient{BaseURL: baseURL, APIKey: apiKey, Model: model}
}

// SplitSubtitle asks the backend-owned prompt ability to split raw text.
func (c *LLMClient) SplitSubtitle(text string, maxCharsPerSeg int, persona string) ([]Segment, []SentenceGroup, error) {
	resp, err := cloud.RunAgentAbility("/v1/agent/subtitle/split", map[string]any{
		"text": text,
	}, map[string]any{
		"max_chars": maxCharsPerSeg,
		"persona":   persona,
		"model":     c.Model,
	}, c.APIKey)
	if err != nil {
		return nil, nil, err
	}

	rawSegments, _ := resp.Result["segments"].([]any)
	rawGroups, _ := resp.Result["sentence_groups"].([]any)
	segments := make([]Segment, 0, len(rawSegments))
	for _, raw := range rawSegments {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		segments = append(segments, Segment{
			SegID: intFromAny(item["seg_id"]),
			Start: floatFromAny(item["start"]),
			End:   floatFromAny(item["end"]),
			Text:  stringFromAny(item["text"]),
		})
	}
	groups := make([]SentenceGroup, 0, len(rawGroups))
	for _, raw := range rawGroups {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		groups = append(groups, SentenceGroup{
			StartSegID: intFromAny(item["start_seg_id"]),
			EndSegID:   intFromAny(item["end_seg_id"]),
			Text:       stringFromAny(item["text"]),
		})
	}
	if len(segments) == 0 {
		return nil, nil, fmt.Errorf("backend returned no subtitle segments")
	}
	if len(groups) == 0 {
		groups = buildSentenceGroups(segments)
	}
	return segments, groups, nil
}

// HighlightByLLM asks the backend-owned prompt ability to identify highlight keywords.
func (c *LLMClient) HighlightByLLM(segments []Segment) ([]Segment, error) {
	resp, err := cloud.RunAgentAbility("/v1/agent/subtitle/highlight", map[string]any{
		"segments": segmentsToPayload(segments),
	}, map[string]any{
		"model": c.Model,
	}, c.APIKey)
	if err != nil {
		return segments, err
	}

	highlightMap := make(map[int]string)
	for _, raw := range anySlice(resp.Result["highlights"]) {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		highlightMap[intFromAny(item["seg_id"])] = stringFromAny(item["word"])
	}
	for i := range segments {
		word := highlightMap[segments[i].SegID]
		if word != "" && containsText(segments[i].Text, word) {
			segments[i].HighlightWord = word
		}
	}
	return segments, nil
}

// AssignEffectsByLLM asks the backend-owned prompt ability to assign subtitle effects.
func (c *LLMClient) AssignEffectsByLLM(segments []Segment, maxEffects int) ([]Segment, error) {
	resp, err := cloud.RunAgentAbility("/v1/agent/subtitle/effects", map[string]any{
		"segments": segmentsToPayload(segments),
	}, map[string]any{
		"max_effects": maxEffects,
		"model":       c.Model,
	}, c.APIKey)
	if err != nil {
		return segments, err
	}

	effectMap := make(map[int]string)
	for _, raw := range anySlice(resp.Result["effects"]) {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		effectMap[intFromAny(item["seg_id"])] = stringFromAny(item["effect_type"])
	}
	for i := range segments {
		if eff, ok := effectMap[segments[i].SegID]; ok {
			segments[i].EffectType = eff
		}
	}
	return segments, nil
}

func segmentsToPayload(segments []Segment) []map[string]any {
	items := make([]map[string]any, 0, len(segments))
	for _, seg := range segments {
		items = append(items, map[string]any{
			"seg_id":         seg.SegID,
			"start":          seg.Start,
			"end":            seg.End,
			"text":           seg.Text,
			"highlight_word": seg.HighlightWord,
			"effect_type":    seg.EffectType,
		})
	}
	return items
}

func containsText(text, word string) bool {
	return word == "" || len(text) >= len(word) && (text == word || indexText(text, word) >= 0)
}

func indexText(text, word string) int {
	for i := 0; i+len(word) <= len(text); i++ {
		if text[i:i+len(word)] == word {
			return i
		}
	}
	return -1
}

func anySlice(value any) []any {
	items, _ := value.([]any)
	return items
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}

func floatFromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
