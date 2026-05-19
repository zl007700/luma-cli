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
			friendly := AssetFriendlyName(objKey)
			normalized := ResourceKeyFromMap(m, cardKey)
			if friendly == name || filename == name || objKey == name || normalized == name {
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
			friendly := AssetFriendlyName(objKey)
			normalized := ResourceKeyFromMap(m, cardKey)
			if friendly == name || filename == name || objKey == name || normalized == name {
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
	return base
}
