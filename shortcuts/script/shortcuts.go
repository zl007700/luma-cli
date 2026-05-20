package script

import "github.com/luma-cli/lumer-cli/shortcuts/common"

// Shortcuts returns cloud script generation capabilities.
func Shortcuts() []common.Shortcut {
	return []common.Shortcut{
		{
			ID:          "script.rewrite",
			Service:     "script",
			Command:     "rewrite",
			Description: "Rewrite source copy into short-video spoken script through backend-owned prompt logic.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "text", Description: "Source copy text. Can also be positional.", Required: true},
				{Name: "input", Description: "Read source copy from a local text file."},
				{Name: "length", Description: "Length preset: short, medium, long, or 短/中/长.", Default: "short"},
				{Name: "output", Description: "Output JSON path.", Default: "step1_rewrite.json"},
			},
			Outputs:  []string{"request_id", "original_text", "rewritten_text", "usage", "output_path"},
			Examples: []string{"luma-cli script rewrite --input source_script.txt --length short --output step1_rewrite.json"},
			Skills:   []string{"luma-video-workflow", "luma-viral-remix-workflow"},
		},
	}
}
