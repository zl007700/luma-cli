package atom

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
)

// ResolveAssetKey resolves a friendly asset name to a cloud object key.
func ResolveAssetKey(group, name, cardKey string) (string, error) {
	if strings.Contains(name, "/") {
		return NormalizeResourceKey(name, cardKey), nil
	}

	items, err := cloud.AssetList(group, cardKey)
	if err == nil {
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			objKey, _ := m["object_key"].(string)
			filename, _ := m["filename"].(string)
			displayName := ResourceDisplayName(m)
			friendly := AssetFriendlyName(objKey)
			normalized := ResourceKeyFromMap(m, cardKey)
			if displayName == name || friendly == name || filename == name || objKey == name || normalized == name {
				return normalized, nil
			}
		}
	}

	commonGroup := "common/" + group
	commonItems, err := cloud.AssetList(commonGroup, cardKey)
	if err == nil {
		for _, item := range commonItems {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			objKey, _ := m["object_key"].(string)
			filename, _ := m["filename"].(string)
			displayName := ResourceDisplayName(m)
			friendly := AssetFriendlyName(objKey)
			normalized := ResourceKeyFromMap(m, cardKey)
			if displayName == name || friendly == name || filename == name || objKey == name || normalized == name {
				return normalized, nil
			}
		}
	}

	return "", fmt.Errorf("asset %q not found in group %q", name, group)
}

func ResourceKeyFromMap(item map[string]any, cardKey string) string {
	objKey, _ := item["object_key"].(string)
	normalized := NormalizeResourceKey(objKey, cardKey)
	if userID, _ := item["user_id"].(string); userID == "common" && !strings.HasPrefix(normalized, "common/") {
		normalized = "common/" + normalized
	}
	return normalized
}

func ResourceDisplayName(item map[string]any) string {
	if name, _ := item["display_name"].(string); strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	if meta, ok := item["meta"].(map[string]any); ok {
		if name, _ := meta["display_name"].(string); strings.TrimSpace(name) != "" {
			return strings.TrimSpace(name)
		}
	}
	return ""
}

// NormalizeResourceKey converts storage object keys to the relative keys accepted by task APIs.
func NormalizeResourceKey(objKey, cardKey string) string {
	key := strings.TrimSpace(strings.ReplaceAll(objKey, "\\", "/"))
	if key == "" {
		return ""
	}
	prefix := "prod/resource/" + strings.TrimSpace(cardKey) + "/"
	return strings.TrimPrefix(key, prefix)
}

// AssetFriendlyName extracts a human-readable name from an object key.
func AssetFriendlyName(objKey string) string {
	base := objKey
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.TrimSuffix(base, "_original")
	if idx := strings.LastIndex(base, "_"); idx >= 0 && isShortHex(base[idx+1:]) {
		base = base[:idx]
	}
	return base
}

func isShortHex(value string) bool {
	if len(value) != 12 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') && !(r >= 'A' && r <= 'F') {
			return false
		}
	}
	return true
}
