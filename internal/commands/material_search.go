package commands

import (
	"path/filepath"
	"sort"
	"strings"
)

func searchMaterials(materials []materialDescriptor, query string, limit int) []materialSearchMatch {
	terms := splitSearchTerms(query)
	matches := make([]materialSearchMatch, 0, len(materials))
	for _, material := range materials {
		score, fields := scoreMaterial(material, terms)
		if score <= 0 {
			continue
		}
		matches = append(matches, materialSearchMatch{
			Material:      material,
			Score:         score,
			MatchedFields: fields,
		})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].Material.Title < matches[j].Material.Title
		}
		return matches[i].Score > matches[j].Score
	})
	if len(matches) > limit {
		return matches[:limit]
	}
	return matches
}

func splitSearchTerms(query string) []string {
	parts := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == ',' || r == ';' || r == '，' || r == '。' || r == '、'
	})
	terms := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			terms = append(terms, part)
		}
	}
	if len(terms) == 0 && strings.TrimSpace(query) != "" {
		terms = []string{strings.ToLower(strings.TrimSpace(query))}
	}
	return terms
}

func scoreMaterial(material materialDescriptor, terms []string) (int, []string) {
	fields := map[string]string{
		"title":          material.Title,
		"summary":        material.Summary,
		"description":    material.Description,
		"visual_focus":   material.VisualFocus,
		"tags":           strings.Join(material.Tags, " "),
		"selling_points": strings.Join(material.SellingPoints, " "),
		"path":           filepath.Base(material.Path),
	}
	weights := map[string]int{
		"title":          8,
		"summary":        5,
		"description":    5,
		"visual_focus":   5,
		"tags":           4,
		"selling_points": 4,
		"path":           2,
	}
	score := 0
	matched := []string{}
	for name, value := range fields {
		text := strings.ToLower(value)
		fieldScore := 0
		for _, term := range terms {
			if term != "" && strings.Contains(text, term) {
				fieldScore += weights[name]
			}
		}
		if fieldScore > 0 {
			score += fieldScore
			matched = append(matched, name)
		}
	}
	sort.Strings(matched)
	return score, matched
}
