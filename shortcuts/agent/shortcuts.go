package agent

import "github.com/luma-cli/lumer-cli/shortcuts/common"

// Shortcuts returns backend-owned prompt abilities.
func Shortcuts() []common.Shortcut {
	return []common.Shortcut{
		{
			ID:          "subtitle.split",
			Service:     "agent",
			Command:     "run subtitle.split",
			Description: "Split text into subtitle segments through backend-owned prompt logic.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "input", Description: "Payload JSON path.", Required: true},
				{Name: "output", Description: "Output JSON path."},
			},
			Outputs:  []string{"segments", "sentence_groups", "output_path"},
			Examples: []string{"luma-cli agent run subtitle.split --input payload.json --output split.json"},
			Skills:   []string{"luma-subtitle", "luma-video-workflow"},
		},
		{
			ID:          "subtitle.highlight",
			Service:     "agent",
			Command:     "run subtitle.highlight",
			Description: "Assign subtitle highlight keywords through backend-owned prompt logic.",
			Risk:        "write",
			Flags:       []common.Flag{{Name: "input", Description: "Payload JSON path.", Required: true}, {Name: "output", Description: "Output JSON path."}},
			Outputs:     []string{"highlights", "output_path"},
			Examples:    []string{"luma-cli agent run subtitle.highlight --input segments.json --output highlights.json"},
			Skills:      []string{"luma-subtitle", "luma-video-workflow"},
		},
		{
			ID:          "subtitle.effects",
			Service:     "agent",
			Command:     "run subtitle.effects",
			Description: "Assign subtitle effects through backend-owned prompt logic.",
			Risk:        "write",
			Flags:       []common.Flag{{Name: "input", Description: "Payload JSON path.", Required: true}, {Name: "output", Description: "Output JSON path."}},
			Outputs:     []string{"effects", "output_path"},
			Examples:    []string{"luma-cli agent run subtitle.effects --input segments.json --output effects.json"},
			Skills:      []string{"luma-subtitle", "luma-video-workflow"},
		},
		{
			ID:          "title.generate",
			Service:     "agent",
			Command:     "run title.generate",
			Description: "Generate video title, cover copy, and tags through backend-owned prompt logic.",
			Risk:        "write",
			Flags:       []common.Flag{{Name: "input", Description: "Payload JSON path.", Required: true}, {Name: "output", Description: "Output JSON path."}},
			Outputs:     []string{"cover_title", "cover_subtitle", "video_title", "tags", "output_path"},
			Examples:    []string{"luma-cli agent run title.generate --input title_payload.json --output title.json"},
			Skills:      []string{"luma-video-workflow"},
		},
		{
			ID:          "storyboard.scene",
			Service:     "agent",
			Command:     "run storyboard.scene",
			Description: "Generate picture-in-picture scene plan through backend-owned prompt logic.",
			Risk:        "write",
			Flags:       []common.Flag{{Name: "input", Description: "Payload JSON path.", Required: true}, {Name: "output", Description: "Output JSON path."}},
			Outputs:     []string{"scene_units", "output_path"},
			Examples:    []string{"luma-cli agent run storyboard.scene --input segments.json --output scene.json"},
			Skills:      []string{"luma-video-workflow"},
		},
		{
			ID:          "storyboard.shot",
			Service:     "agent",
			Command:     "run storyboard.shot",
			Description: "Generate multi-shot table from scene units.",
			Risk:        "write",
			Flags:       []common.Flag{{Name: "input", Description: "Payload JSON path.", Required: true}, {Name: "output", Description: "Output JSON path."}},
			Outputs:     []string{"shot_units", "output_path"},
			Examples:    []string{"luma-cli agent run storyboard.shot --input scene_payload.json --output shots.json"},
			Skills:      []string{"luma-digital-human", "luma-video-workflow"},
		},
		{
			ID:          "material.match",
			Service:     "agent",
			Command:     "run material.match",
			Description: "Match picture-in-picture materials to scene units through backend-owned prompt logic.",
			Risk:        "write",
			Flags:       []common.Flag{{Name: "input", Description: "Payload JSON path.", Required: true}, {Name: "output", Description: "Output JSON path."}},
			Outputs:     []string{"inserts", "output_path"},
			Examples:    []string{"luma-cli agent run material.match --input match_payload.json --output inserts.json"},
			Skills:      []string{"luma-video-workflow"},
		},
	}
}
