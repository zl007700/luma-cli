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
		return name, nil
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
			if friendly == name || filename == name || objKey == name {
				return objKey, nil
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
			if friendly == name || filename == name || objKey == name {
				return "common/" + objKey, nil
			}
		}
	}

	return "", fmt.Errorf("asset %q not found in group %q", name, group)
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
