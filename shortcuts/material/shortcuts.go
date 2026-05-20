package material

import "github.com/luma-cli/lumer-cli/shortcuts/common"

// Shortcuts returns local material metadata capabilities.
func Shortcuts() []common.Shortcut {
	return []common.Shortcut{
		{
			ID:          "material.describe",
			Service:     "material",
			Command:     "describe",
			Description: "Describe local image/video materials into an agent-readable JSON candidate list.",
			Risk:        "read",
			Flags: []common.Flag{
				{Name: "file_or_dir", Description: "Local material file or directory.", Required: true},
				{Name: "output", Description: "Output JSON path.", Default: "materials.json"},
			},
			Outputs:  []string{"materials", "output_path"},
			Examples: []string{"luma-cli material describe ./materials --output materials.json"},
			Skills:   []string{"luma-assets", "luma-video-workflow"},
		},
		{
			ID:          "pip.scan",
			Service:     "pip",
			Command:     "scan",
			Description: "Alias for local material description before PIP planning.",
			Risk:        "read",
			Flags: []common.Flag{
				{Name: "material_dir", Description: "Local material directory.", Required: true},
				{Name: "output", Description: "Output JSON path.", Default: "materials.json"},
			},
			Outputs:  []string{"materials", "output_path"},
			Examples: []string{"luma-cli pip scan ./materials --output materials.json"},
			Skills:   []string{"luma-video-workflow"},
		},
		{
			ID:          "pip.plan",
			Service:     "pip",
			Command:     "plan",
			Description: "Generate a PIP plan from subtitle segments and local material candidates.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "segments", Description: "Subtitle segments JSON path.", Required: true},
				{Name: "materials", Description: "Material candidates JSON path.", Required: true},
				{Name: "output", Description: "Output plan JSON path.", Default: "pip_plan.json"},
			},
			Outputs:  []string{"scene_units", "inserts", "output_path"},
			Examples: []string{"luma-cli pip plan --segments segments.json --materials materials.json --output pip_plan.json"},
			Skills:   []string{"luma-video-workflow"},
		},
	}
}
