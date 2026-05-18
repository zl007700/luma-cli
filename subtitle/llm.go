package subtitle

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
)

// LLMClient wraps calls to the router LLM backend.
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

// Chat sends a prompt and returns the LLM's text response.
func (c *LLMClient) Chat(prompt string) (string, error) {
	messages := []map[string]string{
		{"role": "user", "content": prompt},
	}
	reqBody := map[string]any{
		"model": c.Model,
		"messages": messages,
		"temperature": 0.3,
		"max_tokens": 2048,
		"extra_body": map[string]any{"thinking": map[string]any{"type": "disabled"}},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := newHTTPRequest("POST", c.BaseURL+"/v1/chat/completions", strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", c.APIKey)

	resp, err := doRequest(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse response failed: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("LLM error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}
	return result.Choices[0].Message.Content, nil
}

// HighlightByLLM asks LLM to identify highlight keywords for each segment.
// Returns segments with hight_light_word populated (max 30% of segments highlighted).
func (c *LLMClient) HighlightByLLM(segments []Segment) ([]Segment, error) {
	segJSON, _ := json.Marshal(segments)
	prompt := fmt.Sprintf(`你是字幕高亮助手。

【任务】
从每条字幕中选择最值得强调的词（情绪词、关键词等）。

【规则】
1. 高亮词必须来自原文，不能自己编
2. 每条最多选 1 个词，词长 1-5 个字
3. 输出 JSON 格式
4. 最多高亮 30%% 的 segments

【字幕列表】
%s

【输出格式】
{"highlights": [{"seg_id": 0, "word": "关键词"}, ...]}`, string(segJSON))

	response, err := c.Chat(prompt)
	if err != nil {
		return segments, err
	}

	// Extract JSON from response
	match := findJSON(response)
	if match == nil {
		return segments, nil
	}

	var result struct {
		Highlights []struct {
			SegID int    `json:"seg_id"`
			Word  string `json:"word"`
		} `json:"highlights"`
	}
	if err := json.Unmarshal(match, &result); err != nil {
		return segments, err
	}

	highlightMap := make(map[int]string)
	for _, h := range result.Highlights {
		highlightMap[h.SegID] = h.Word
	}

	for i := range segments {
		word := highlightMap[segments[i].SegID]
		if word != "" && strings.Contains(segments[i].Text, word) {
			segments[i].HighlightWord = word
		}
	}
	return segments, nil
}

// AssignEffectsByLLM asks LLM to assign subtitle effects to segments.
// Supported effects: blur_in, bounce_in, scale_pop, rotate_pop, wave_bounce, none.
func (c *LLMClient) AssignEffectsByLLM(segments []Segment, maxEffects int) ([]Segment, error) {
	// Build segment info string
	var segLines []string
	for _, seg := range segments {
		duration := seg.End - seg.Start
		highlight := seg.HighlightWord
		if highlight == "" {
			highlight = "无"
		}
		segLines = append(segLines, fmt.Sprintf("seg_id=%d, start=%.2f, end=%.2f, duration=%.2f, highlight=%s, text=%s",
			seg.SegID, seg.Start, seg.End, duration, highlight, seg.Text))
	}

	prompt := fmt.Sprintf(`你是短视频字幕特效编排助手。
任务：结合整段字幕的上下文和叙事节奏，只给少量真正值得强调的句子分配字幕特效；不适合的句子输出 none。
叙事提示：优先考虑转折、情绪峰值、结论落点、数字卖点、规则步骤、强提醒；普通承接句、铺垫句、大多数过渡句都应该是 none。
特效与叙事适配：
- blur_in: 适合铺垫、回忆感、柔和引入、娓娓道来。
- bounce_in: 适合强强调、情绪抬升、结论落点、金句提醒。
- scale_pop: 适合数字结论、卖点、收益点、突然出现的信息重点。
- rotate_pop: 适合轻松俏皮的小转折、小包袱、娱乐化语气。
- wave_bounce: 适合节奏感较强的口播、连续强调、轻微热闹感。
硬规则：
1. 只能使用这些 effect_type：blur_in, bounce_in, scale_pop, rotate_pop, wave_bounce，或者 none。
2. 整体最多选择 %d 个有特效的 seg，绝大多数句子应为 none。
3. 尽量不要让相邻 seg 连续都带特效，除非确实是连续高潮。
4. duration 小于 0.60 秒的句子必须输出 none。
5. rotate_pop 只用于轻松俏皮内容，避免滥用夸张风格。
6. 输出只要 JSON，不要解释，不要 markdown。

字幕 segments:
- %s

输出 JSON:
{"effects": [{"seg_id": 1, "effect_type": "bounce_in"}, ...]}`, maxEffects, strings.Join(segLines, "\n- "))

	response, err := c.Chat(prompt)
	if err != nil {
		return segments, err
	}

	match := findJSON(response)
	if match == nil {
		return segments, nil
	}

	var result struct {
		Effects []struct {
			SegID      int    `json:"seg_id"`
			EffectType string `json:"effect_type"`
		} `json:"effects"`
	}
	if err := json.Unmarshal(match, &result); err != nil {
		return segments, err
	}

	effectMap := make(map[int]string)
	for _, e := range result.Effects {
		effectMap[e.SegID] = e.EffectType
	}

	for i := range segments {
		if eff, ok := effectMap[segments[i].SegID]; ok {
			segments[i].EffectType = eff
		}
	}
	return segments, nil
}

// findJSON extracts first JSON object from text.
func findJSON(s string) []byte {
	start := strings.Index(s, "{")
	if start < 0 {
		return nil
	}
	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == '{' {
			depth++
		} else if s[i] == '}' {
			depth--
			if depth == 0 {
				return []byte(s[start : i+1])
			}
		}
	}
	return nil
}