package commands

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
)

func loadSegmentsForPIP(path string) ([]any, error) {
	payload, err := readJSONObject(path)
	if err != nil {
		return nil, err
	}
	segments := anyListFromPayload(payload, "segments")
	if len(segments) == 0 {
		return nil, fmt.Errorf("segments cannot be empty")
	}
	return segments, nil
}

func loadMaterialsForPIP(path string) ([]any, error) {
	payload, err := readJSONObject(path)
	if err != nil {
		return nil, err
	}
	materials := anyListFromPayload(payload, "materials")
	if len(materials) == 0 {
		return nil, fmt.Errorf("materials cannot be empty")
	}
	return materials, nil
}

func loadMaterialMapsForPIP(path string) ([]map[string]any, error) {
	materials, err := loadMaterialsForPIP(path)
	if err != nil {
		return nil, err
	}
	return listMapFromAny(materials), nil
}

func loadSceneUnitsForPIP(path string) ([]map[string]any, error) {
	payload, err := readJSONObject(path)
	if err != nil {
		return nil, err
	}
	sceneUnits := anyListFromPayload(payload, "scene_units")
	if len(sceneUnits) == 0 {
		sceneUnits = anyListFromPayload(payload, "scenes")
	}
	if len(sceneUnits) == 0 {
		return nil, fmt.Errorf("scene_units cannot be empty")
	}
	return listMapFromAny(sceneUnits), nil
}

func cloudSceneUnits(segments []any, cardKey string) ([]any, error) {
	sceneResp, err := cloud.RunAgentAbility("/v1/agent/storyboard/scene", map[string]any{"segments": segments}, nil, cardKey)
	if err != nil {
		return nil, err
	}
	sceneUnits := anyListFromPayload(sceneResp.Result, "scene_units")
	if len(sceneUnits) == 0 {
		return nil, fmt.Errorf("backend returned no scene_units")
	}
	return sceneUnits, nil
}

func matchPIPMaterials(sceneUnits, materials []map[string]any, mode string, maxInserts int) ([]map[string]any, string, error) {
	if maxInserts <= 0 {
		maxInserts = 8
	}
	if mode == "" {
		mode = "auto"
	}
	if mode != "local" {
		cfg := loadConfig()
		if cfg != nil {
			matchResp, err := cloud.RunAgentAbility("/v1/agent/material/match", map[string]any{"scene_units": sceneUnits, "materials": materials, "max_inserts": maxInserts}, nil, cfg.CardKey)
			if err == nil {
				rawInserts := anyListFromPayload(matchResp.Result, "inserts")
				inserts := normalizePIPInserts(rawInserts, sceneUnits, materials)
				if len(inserts) > maxInserts {
					inserts = inserts[:maxInserts]
				}
				return inserts, "cloud", nil
			}
			if mode == "cloud" {
				return nil, "cloud", err
			}
		} else if mode == "cloud" {
			return nil, "cloud", fmt.Errorf("not logged in. Run: luma-cli auth login <phone_or_account>")
		}
	}
	return localPIPMaterialMatches(sceneUnits, materials, maxInserts), "local", nil
}

func localPIPMaterialMatches(sceneUnits, materials []map[string]any, maxInserts int) []map[string]any {
	descriptors := materialDescriptorsFromMaps(materials)
	inserts := []map[string]any{}
	used := map[string]bool{}
	for _, scene := range sceneUnits {
		query := sceneSearchText(scene)
		if query == "" {
			continue
		}
		matches := searchMaterials(descriptors, query, len(descriptors))
		if len(matches) == 0 {
			continue
		}
		var chosen materialSearchMatch
		found := false
		for _, match := range matches {
			id := match.Material.MaterialID
			if id == "" {
				id = match.Material.Path
			}
			if !used[id] {
				chosen = match
				used[id] = true
				found = true
				break
			}
		}
		if !found {
			continue
		}
		sceneID := firstString(scene, "scene_id", "id", "segment_id", "sent_id")
		insert := map[string]any{
			"scene_id":        sceneID,
			"start_scene_id":  sceneID,
			"end_scene_id":    sceneID,
			"material_id":     chosen.Material.MaterialID,
			"material_path":   chosen.Material.Path,
			"media_type":      chosen.Material.MediaType,
			"start":           scene["start"],
			"end":             scene["end"],
			"score":           chosen.Score,
			"matched_fields":  chosen.MatchedFields,
			"reason":          "local keyword match",
			"planning_source": "local",
		}
		inserts = append(inserts, insert)
		if len(inserts) >= maxInserts {
			break
		}
	}
	return normalizePIPInserts(anySliceFromMaps(inserts), sceneUnits, materials)
}

func materialDescriptorsFromMaps(items []map[string]any) []materialDescriptor {
	out := make([]materialDescriptor, 0, len(items))
	for _, item := range items {
		descriptor := materialDescriptor{
			MaterialID:    firstString(item, "material_id", "id", "resource_id"),
			MediaType:     firstNonEmpty(firstString(item, "media_type", "type"), "video"),
			Path:          firstString(item, "path", "file_path", "local_path", "material_path"),
			Title:         firstString(item, "title", "name", "file_name"),
			Summary:       firstString(item, "summary", "description", "content_summary", "visual_summary"),
			Description:   firstString(item, "description", "summary", "content_summary", "visual_summary"),
			Tags:          stringListFromKeys(item, "tags", "keywords", "labels"),
			VisualFocus:   firstString(item, "visual_focus", "main_object", "subject", "scene"),
			SellingPoints: stringListFromKeys(item, "selling_points", "highlights", "features"),
		}
		if descriptor.MaterialID == "" && descriptor.Path != "" {
			descriptor.MaterialID = materialID(descriptor.Path)
		}
		if descriptor.Title == "" {
			descriptor.Title = strings.TrimSuffix(filepath.Base(descriptor.Path), filepath.Ext(descriptor.Path))
		}
		out = append(out, descriptor)
	}
	return out
}

func sceneSearchText(scene map[string]any) string {
	parts := []string{
		firstString(scene, "title", "topic", "scene_title"),
		firstString(scene, "summary", "description", "content", "text", "script"),
		firstString(scene, "visual_focus", "main_object", "subject"),
		strings.Join(stringListFromKeys(scene, "keywords", "tags", "labels"), " "),
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func anySliceFromMaps(items []map[string]any) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}

func normalizePIPInserts(raw []any, scenes []map[string]any, materials []map[string]any) []map[string]any {
	sceneMap := map[string]map[string]any{}
	for _, scene := range scenes {
		id := firstString(scene, "scene_id", "sent_id", "segment_id", "id")
		if id != "" {
			sceneMap[id] = scene
		}
	}
	materialIDs := map[string]bool{}
	for _, material := range materials {
		id := firstString(material, "material_id", "id", "resource_id")
		if id != "" {
			materialIDs[id] = true
		}
	}
	inserts := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		insert, ok := item.(map[string]any)
		if !ok {
			continue
		}
		normalized := cloneMap(insert)
		materialID := firstString(normalized, "material_id", "id", "resource_id")
		if materialID == "" {
			continue
		}
		normalized["material_id"] = materialID
		if !materialIDs[materialID] {
			normalized["material_known"] = false
		}
		sceneID := firstString(normalized, "scene_id", "start_scene_id", "start_sent_id")
		if sceneID != "" {
			if normalized["start_scene_id"] == nil {
				normalized["start_scene_id"] = sceneID
			}
			if normalized["end_scene_id"] == nil {
				normalized["end_scene_id"] = firstNonEmpty(firstString(normalized, "end_scene_id", "end_sent_id"), sceneID)
			}
			if scene, ok := sceneMap[sceneID]; ok {
				if _, ok := normalized["start"]; !ok {
					normalized["start"] = scene["start"]
				}
				if _, ok := normalized["end"]; !ok {
					normalized["end"] = scene["end"]
				}
				if _, ok := normalized["start_segment_id"]; !ok {
					normalized["start_segment_id"] = firstString(scene, "start_segment_id", "segment_id", "sent_id")
				}
				if _, ok := normalized["end_segment_id"]; !ok {
					normalized["end_segment_id"] = firstString(scene, "end_segment_id", "segment_id", "sent_id")
				}
			}
		}
		inserts = append(inserts, normalized)
	}
	sort.SliceStable(inserts, func(i, j int) bool {
		return insertSortValue(inserts[i]) < insertSortValue(inserts[j])
	})
	filtered := make([]map[string]any, 0, len(inserts))
	lastEnd := -1.0
	for _, insert := range inserts {
		start, _ := numberAny(insert["start"])
		end, _ := numberAny(insert["end"])
		if end > start && start < lastEnd {
			continue
		}
		if end > lastEnd {
			lastEnd = end
		}
		filtered = append(filtered, insert)
	}
	return filtered
}

func buildPIPAssignments(inserts []map[string]any) []map[string]any {
	assignments := make([]map[string]any, 0, len(inserts))
	for _, insert := range inserts {
		assignments = append(assignments, map[string]any{
			"scene_id":    firstString(insert, "scene_id", "start_scene_id"),
			"material_id": firstString(insert, "material_id"),
			"start":       insert["start"],
			"end":         insert["end"],
			"reason":      firstString(insert, "reason", "match_reason"),
		})
	}
	return assignments
}

func insertSortValue(insert map[string]any) float64 {
	if start, ok := numberAny(insert["start"]); ok {
		return start
	}
	if id := firstString(insert, "start_segment_id", "segment_id", "sent_id"); id != "" {
		if n, err := strconv.ParseFloat(strings.TrimLeft(id, "s_"), 64); err == nil {
			return n
		}
	}
	return 0
}
