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
				{Name: "output", Description: "Output JSON path.", Default: "research.json"},
			},
			Outputs:  []string{"response", "output_path"},
			Examples: []string{"luma-cli research run --role \"做家装瓷砖的一线门店老板\" --mode precise --output research.json"},
			Skills:   []string{"luma-content-research", "luma-video-workflow"},
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
			Examples: []string{"luma-cli research persona save tile_shop --role \"三线城市瓷砖门店老板\""},
			Skills:   []string{"luma-content-research"},
		},
	}
}
