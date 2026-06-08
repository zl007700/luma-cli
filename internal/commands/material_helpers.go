package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// --- material merge / meta helpers ---

func materialDescriptorsFromPayload(payload map[string]any) ([]materialDescriptor, error) {
	data, err := json.Marshal(payload["materials"])
	if err != nil {
		return nil, err
	}
	var materials []materialDescriptor
	if err := json.Unmarshal(data, &materials); err != nil {
		return nil, err
	}
	return materials, nil
}

func readMaterialMetas(path string) ([]map[string]any, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		var metas []map[string]any
		err := filepath.WalkDir(path, func(itemPath string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || strings.ToLower(filepath.Ext(itemPath)) != ".json" {
				return nil
			}
			payload, err := readJSONObject(itemPath)
			if err != nil {
				return err
			}
			if payload["object_name"] == nil {
				payload["object_name"] = filepath.Base(itemPath)
			}
			if payload["file_name"] == nil {
				payload["file_name"] = strings.TrimSuffix(filepath.Base(itemPath), filepath.Ext(itemPath))
			}
			metas = append(metas, payload)
			return nil
		})
		return metas, err
	}
	payload, err := readJSONObject(path)
	if err != nil {
		return nil, err
	}
	if items := listMap(payload["items"]); len(items) > 0 {
		return items, nil
	}
	if items := listMap(payload["metas"]); len(items) > 0 {
		return items, nil
	}
	if items := listMap(payload["materials"]); len(items) > 0 {
		return items, nil
	}
	return []map[string]any{payload}, nil
}

func mergeMetaIntoMaterials(materials []materialDescriptor, metas []map[string]any) []materialDescriptor {
	if len(materials) == 0 || len(metas) == 0 {
		return materials
	}
	for _, meta := range metas {
		index := matchMaterialMeta(materials, meta)
		if index < 0 {
			continue
		}
		applyMaterialMeta(&materials[index], meta)
	}
	return materials
}

func matchMaterialMeta(materials []materialDescriptor, meta map[string]any) int {
	if len(materials) == 1 {
		return 0
	}
	names := []string{
		firstString(meta, "material_id", "id", "resource_id"),
		firstString(meta, "filename", "file_name", "object_name", "object_key", "title", "name"),
	}
	for i, material := range materials {
		for _, name := range names {
			if name == "" {
				continue
			}
			base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
			materialBase := strings.TrimSuffix(filepath.Base(material.Path), filepath.Ext(material.Path))
			if material.MaterialID == name || materialBase == base || strings.Contains(material.Path, name) || strings.Contains(material.Title, base) || strings.Contains(base, material.Title) {
				return i
			}
		}
	}
	return -1
}

func applyMaterialMeta(material *materialDescriptor, meta map[string]any) {
	flat := flattenMaterialMeta(meta)
	if value := firstString(flat, "summary", "description", "content_summary", "visual_summary", "caption"); value != "" {
		material.Summary = value
		material.Description = value
	}
	if value := firstString(flat, "visual_focus", "main_object", "subject", "scene"); value != "" {
		material.VisualFocus = value
	}
	if tags := stringListFromKeys(flat, "tags", "keywords", "labels", "scenes"); len(tags) > 0 {
		material.Tags = uniqueStrings(append(material.Tags, tags...))
	}
	if points := stringListFromKeys(flat, "selling_points", "highlights", "features"); len(points) > 0 {
		material.SellingPoints = uniqueStrings(append(material.SellingPoints, points...))
	}
}

func flattenMaterialMeta(meta map[string]any) map[string]any {
	out := cloneMap(meta)
	for _, key := range []string{"result", "meta", "metadata", "analysis", "understanding", "data"} {
		if nested, ok := meta[key].(map[string]any); ok {
			for nestedKey, value := range nested {
				if _, exists := out[nestedKey]; !exists {
					out[nestedKey] = value
				}
			}
		}
	}
	return out
}

func normalizeMaterialReview(payload map[string]any, purpose string) map[string]any {
	review := flattenMaterialMeta(payload)
	if nested, ok := payload["review"].(map[string]any); ok {
		review = flattenMaterialMeta(nested)
	}
	score := func(key string) float64 {
		switch value := review[key].(type) {
		case float64:
			return value
		case int:
			return float64(value)
		case json.Number:
			number, _ := value.Float64()
			return number
		default:
			return 0
		}
	}
	issues := stringListFromKeys(review, "issues")
	blocking := map[string]bool{
		"irrelevant": true, "unreadable_text": true, "blurred": true,
		"obstruction": true, "placeholder": true, "bad_crop": true,
		"distorted": true, "low_resolution": true,
	}
	hasBlockingIssue := false
	for _, issue := range issues {
		if blocking[issue] {
			hasBlockingIssue = true
			break
		}
	}
	minRelevance, minReadability, minQuality, minCredibility := 6.0, 5.0, 5.0, 0.0
	if purpose == "evidence" {
		minRelevance, minReadability, minQuality, minCredibility = 7.0, 7.0, 6.0, 6.0
	}
	usable := !hasBlockingIssue &&
		score("relevance_score") >= minRelevance &&
		score("readability_score") >= minReadability &&
		score("visual_quality_score") >= minQuality &&
		score("credibility_score") >= minCredibility
	review["usable"] = usable
	if usable {
		review["decision"] = "accept"
	} else {
		review["decision"] = "reject"
	}
	review["purpose"] = purpose
	review["thresholds"] = map[string]float64{
		"relevance_score": minRelevance, "readability_score": minReadability,
		"visual_quality_score": minQuality, "credibility_score": minCredibility,
	}
	if !usable && firstString(review, "reject_reason") == "" {
		review["reject_reason"] = "Material did not pass deterministic review thresholds."
	}
	return review
}

func stringListFromKeys(m map[string]any, keys ...string) []string {
	for _, key := range keys {
		switch value := m[key].(type) {
		case []any:
			var out []string
			for _, item := range value {
				if text := strAny(item); text != "" {
					out = append(out, text)
				}
			}
			if len(out) > 0 {
				return out
			}
		case []string:
			if len(value) > 0 {
				return value
			}
		case string:
			if strings.TrimSpace(value) != "" {
				return tagsFromName(value)
			}
		}
	}
	return nil
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}
