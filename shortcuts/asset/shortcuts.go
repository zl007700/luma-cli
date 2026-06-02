package asset

import "github.com/luma-cli/lumer-cli/shortcuts/common"

// Shortcuts returns asset-related atomic capabilities.
func Shortcuts() []common.Shortcut {
	return []common.Shortcut{
		{
			ID:          "asset.upload",
			Service:     "asset",
			Command:     "upload",
			Description: "Upload a local media asset to a named cloud group. For uploaded videos, confirm whether the user intends voice clone, avatar/source role, PIP material, ASR/rewrite, or video processing before choosing a group.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "file", Description: "Local file path.", Required: true},
				{Name: "group", Description: "Asset group name. Use voice for audio/voice samples, roles only when the user wants the visual person as an avatar/source role, and material/PIP groups for reusable video materials.", Default: "default"},
			},
			Outputs:  []string{"name", "group"},
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
			ID:          "asset.understand",
			Service:     "asset",
			Command:     "understand",
			Description: "Run cloud VLM understanding for an uploaded material. Use it to create a short friendly display name for hash-like video assets.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "object_name", Description: "Object name or relative object key.", Required: true},
				{Name: "group", Description: "Asset group name.", Default: "default"},
				{Name: "output", Description: "Output metadata JSON path."},
			},
			Outputs:  []string{"meta", "meta_key", "display_name", "output_path"},
			Examples: []string{"luma-cli asset understand demo.mp4 --group vlm_demo --output material_meta.json"},
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
		},
	}
}
