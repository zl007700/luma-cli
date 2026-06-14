package asset

import "github.com/luma-cli/lumer-cli/shortcuts/common"

// Shortcuts returns asset-related atomic capabilities.
func Shortcuts() []common.Shortcut {
	return []common.Shortcut{
		{
			ID:          "assets.search",
			Service:     "assets",
			Command:     "search",
			Description: "Search Asset Registry V2 assets by kind, group, and scope. Use --scope system when choosing platform defaults for an Agent so user uploads do not affect selection. Read NAME, PROBE, and CAPTION before choosing an asset_id.",
			Risk:        "read",
			Flags: []common.Flag{
				{Name: "kind", Description: "Asset kind, for example template, font, bgm, sfx, avatar, voice, persona_image, material_image, or material_video."},
				{Name: "group", Description: "Optional group_name, for example hook_portrait or common."},
				{Name: "scope", Description: "Optional scope filter: system, user, or job. Use system for platform defaults."},
				{Name: "limit", Description: "Maximum number of assets to return.", Default: "30"},
			},
			Outputs:  []string{"asset_id", "kind", "scope", "group_name", "display_name", "probe", "semantic.caption", "metadata"},
			Examples: []string{"luma-cli assets search --kind template --group hook_portrait --scope system --limit 8", "luma-cli assets search --kind bgm --scope system --limit 8"},
			Skills:   []string{"luma-assets"},
		},
		{
			ID:          "assets.groups",
			Service:     "assets",
			Command:     "groups",
			Description: "List Asset Registry V2 group names visible to the current user. Use --scope system for platform default groups.",
			Risk:        "read",
			Flags: []common.Flag{
				{Name: "kind", Description: "Optional asset kind filter, for example template or bgm."},
				{Name: "scope", Description: "Optional scope filter: system, user, or job. Use system for platform defaults."},
			},
			Outputs:  []string{"group_name", "display_name"},
			Examples: []string{"luma-cli assets groups --kind template --scope system"},
			Skills:   []string{"luma-assets"},
		},
		{
			ID:          "assets.upload",
			Service:     "assets",
			Command:     "upload",
			Description: "Upload a user asset into Asset Registry V2 and enqueue backend intake/preprocessing. Provide --kind and a friendly --name.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "file", Description: "Local file path.", Required: true},
				{Name: "kind", Description: "Asset kind, for example material_image, material_video, voice, bgm, or sfx.", Required: true},
				{Name: "group", Description: "Reusable group_name for this asset."},
				{Name: "name", Description: "Short friendly Chinese display name."},
			},
			Outputs:  []string{"asset_id", "status", "job_id", "group_name"},
			Examples: []string{"luma-cli assets upload image.png --kind material_image --group references --name 门店外景"},
			Skills:   []string{"luma-assets"},
		},
		{
			ID:          "asset.upload",
			Service:     "asset",
			Command:     "upload",
			Description: "Upload a local media asset to a named cloud group. For uploaded videos, confirm whether the user intends voice clone, avatar/source role, PIP material, ASR/rewrite, or video processing before choosing a group. Provide --name with a short friendly display name; do not leave the user with a hash-like asset name.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "file", Description: "Local file path.", Required: true},
				{Name: "group", Description: "Asset group name. Use voice for audio/voice samples, roles only when the user wants the visual person as an avatar/source role, and material/PIP groups for reusable video materials.", Default: "default"},
				{Name: "name", Description: "Short friendly display name, 5-10 Chinese characters when possible. Required for user-uploaded videos unless the user explicitly supplied a name."},
			},
			Outputs:  []string{"name", "display_name", "group"},
			Examples: []string{"luma-cli asset upload avatar.mp4 --group roles --name 老板在家里"},
			Skills:   []string{"luma-assets", "luma-digital-human"},
		},
		{
			ID:          "asset.list",
			Service:     "asset",
			Command:     "list",
			Description: "List assets in a cloud group. Use roles for digital-human avatars; avatar/avatars are aliases for roles. Default/common assets are included when available.",
			Risk:        "read",
			Flags: []common.Flag{
				{Name: "group", Description: "Asset group name. Use roles for digital-human avatars and voice for voice assets.", Default: "default"},
			},
			Outputs:  []string{"items"},
			Examples: []string{"luma-cli asset list voice", "luma-cli asset list roles"},
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
			ID:          "asset.delete",
			Service:     "asset",
			Command:     "delete",
			Description: "Delete a user asset from a group by friendly name or storage stem. Ask for confirmation before deleting user assets.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "name_or_stem", Description: "Asset friendly name or stem, for example upload8393.", Required: true},
				{Name: "group", Description: "Asset group name, for example voice or roles.", Required: true},
			},
			Outputs:  []string{"group_name", "stem", "status", "deleted_meta_count"},
			Examples: []string{"luma-cli asset delete upload8393 --group voice"},
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
