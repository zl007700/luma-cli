package atom

import (
	"os"
	"strings"
	"unicode/utf8"

	"github.com/luma-cli/lumer-cli/cloud"
)

func firstPlayAddrURL(value any) string {
	items, _ := value.([]any)
	for _, item := range items {
		obj, _ := item.(map[string]any)
		src, _ := obj["src"].(string)
		if normalized := normalizeDouyinURL(src); normalized != "" {
			return normalized
		}
	}
	return ""
}

func firstURLListItem(value any) string {
	items, _ := value.([]any)
	for _, item := range items {
		if raw, _ := item.(string); raw != "" {
			return normalizeDouyinURL(raw)
		}
	}
	return ""
}

func numberValue(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return 0
	}
}

func normalizeDouyinURL(value string) string {
	value = strings.TrimSpace(value)
	switch {
	case strings.HasPrefix(value, "//"):
		return "https:" + value
	case strings.HasPrefix(value, "http://"), strings.HasPrefix(value, "https://"):
		return value
	default:
		return ""
	}
}

func sanitizeDouyinFilename(title, fallbackID string) string {
	name := windowsInvalidNameChars.ReplaceAllString(title, "_")
	name = strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	name = strings.Trim(name, " ._")
	if name == "" {
		name = "douyin_" + fallbackID
	}
	return truncateUTF8(name, 80)
}

func truncateUTF8(value string, maxRunes int) string {
	if maxRunes <= 0 || utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxRunes])
}

func loadDouyinCookie(cardKey string) string {
	if cookie := strings.TrimSpace(os.Getenv(envDouyinCookie)); cookie != "" {
		return cookie
	}
	if strings.TrimSpace(cardKey) == "" {
		return ""
	}
	cookie, err := cloud.DouyinDownloadCookie(cardKey)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie)
}
