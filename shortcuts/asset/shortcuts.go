package asset

import "github.com/luma-cli/lumer-cli/shortcuts/common"

// Shortcuts returns asset-related atomic capabilities.
func Shortcuts() []common.Shortcut {
	return []common.Shortcut{
		{
			ID:          "asset.upload",
			Service:     "asset",
			Command:     "upload",
			Description: "Upload a local media asset to a named cloud group.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "file", Description: "Local file path.", Required: true},
				{Name: "group", Description: "Asset group name.", Default: "default"},
			},
			Outputs:  []string{"object_key"},
			Examples: []string{"luma-cli asset upload avatar.mp4 --group roles"},
			Skills:   []string{"luma-assets", "luma-digital-human"},
		},
		{
			ID:          "asset.list",
			Service:     "asset",
			Command:     "list",
			Description: "List assets in a cloud group.",
			Risk:        "read",
			Flags: []common.Flag{
				{Name: "group", Description: "Asset group name.", Default: "default"},
			},
			Outputs:  []string{"items"},
			Examples: []string{"luma-cli asset list voice"},
			Skills:   []string{"luma-assets"},
		},
		{
			ID:          "task.status",
			Service:     "task",
			Command:     "status",
			Description: "Fetch cloud task status and result metadata.",
			Risk:        "read",
			Flags: []common.Flag{
				{Name: "task_id", Description: "Cloud task id.", Required: true},
			},
			Outputs:  []string{"status", "output"},
			Examples: []string{"luma-cli task status task_xxx"},
			Skills:   []string{"luma-video-workflow"},
		},
	}
}
