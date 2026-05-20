package research

import "github.com/luma-cli/lumer-cli/shortcuts/common"

// Shortcuts returns content research and persona atomic capabilities.
func Shortcuts() []common.Shortcut {
	return []common.Shortcut{
		{
			ID:          "research.run",
			Service:     "research",
			Command:     "run",
			Description: "Run backend content research for a role description.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "role", Description: "Role/persona description.", Required: true},
				{Name: "mode", Description: "Research mode.", Default: "precise"},
				{Name: "date-range", Description: "Search time window.", Default: "7d"},
				{Name: "output", Description: "Output JSON path.", Default: "step0_content_research.json"},
			},
			Outputs:  []string{"response", "output_path"},
			Examples: []string{"luma-cli research run --role \"AI工具创业者\" --mode precise --output step0_content_research.json"},
			Skills:   []string{"luma-content-research", "luma-viral-remix-workflow", "luma-video-workflow"},
		},
		{
			ID:          "research.export",
			Service:     "research",
			Command:     "export",
			Description: "Export content research JSON into an Excel-friendly CSV table.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "input", Description: "Research JSON path.", Required: true},
				{Name: "output", Description: "Output CSV path.", Default: "step0_content_research.csv"},
			},
			Outputs:  []string{"output_path", "count"},
			Examples: []string{"luma-cli research export --input step0_content_research.json --output step0_content_research.csv"},
			Skills:   []string{"luma-content-research", "luma-viral-remix-workflow"},
		},
		{
			ID:          "persona.save",
			Service:     "research",
			Command:     "persona save",
			Description: "Save a local persona profile for repeated agent calls.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "name", Description: "Persona name.", Required: true},
				{Name: "role", Description: "Role/persona description.", Required: true},
			},
			Outputs:  []string{"id", "name", "summary", "role_description"},
			Examples: []string{"luma-cli research persona save ai_founder --role \"AI工具创业者\""},
			Skills:   []string{"luma-content-research", "luma-viral-remix-workflow"},
		},
	}
}
