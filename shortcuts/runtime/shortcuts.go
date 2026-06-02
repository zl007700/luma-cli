package runtime

import "github.com/luma-cli/lumer-cli/shortcuts/common"

// Shortcuts returns local runtime and product resource capabilities.
func Shortcuts() []common.Shortcut {
	return []common.Shortcut{
		{
			ID:          "runtime.install",
			Service:     "runtime",
			Command:     "install",
			Description: "Install a signed local runtime, such as ffmpeg, from Luma cloud.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "name", Description: "Runtime name.", Required: true, Default: "ffmpeg"},
				{Name: "version", Description: "Runtime version, or auto.", Default: "auto"},
			},
			Outputs:  []string{"name", "version", "platform", "executable_path"},
			Examples: []string{"luma-cli runtime install ffmpeg"},
		},
		{
			ID:          "client-resource.list",
			Service:     "resource",
			Command:     "list",
			Description: "List cloud-managed fonts, BGM tracks, and templates available for on-demand caching.",
			Risk:        "read",
			Flags: []common.Flag{
				{Name: "type", Description: "Resource type, such as font, bgm, or template."},
				{Name: "tag", Description: "Optional tag filter."},
			},
			Outputs:  []string{"items"},
			Examples: []string{"luma-cli resource list font"},
		},
		{
			ID:          "client-resource.cache",
			Service:     "resource",
			Command:     "cache",
			Description: "Download and cache one signed product resource locally.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "resource_id", Description: "Resource id returned by resource list.", Required: true},
			},
			Outputs:  []string{"id", "type", "version", "path"},
			Examples: []string{"luma-cli resource cache font_default_zh"},
		},
	}
}
