---
name: luma-assets
version: 0.1.0
description: "Manage Luma cloud assets used by generation tools, including voices, avatars, media inputs, and named groups."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli tools describe asset.upload"
  category: "capability"
  entrypoint: false
  relatedSkills: ["luma-shared", "luma-material", "luma-digital-human"]
---

# Luma Assets

Use this skill when an agent needs to inspect or upload reusable media assets for Luma workflows.

For local material libraries and PIP material matching, prefer `../luma-material/SKILL.md`.

## Common Groups

- `voice`: voice samples or generated voice assets
- `roles`: digital human avatar videos
- `asr_input`: uploaded media for transcription
- `enhance_input`: uploaded videos for enhancement
- `lipsync_input`: uploaded audio for lip-sync
- `default`: generic assets

## Commands

```bash
luma-cli asset list voice
luma-cli asset list roles
luma-cli asset upload avatar.mp4 --group roles
luma-cli asset upload voice.wav --group voice
luma-cli voice clone ./sample.wav --name my_voice
luma-cli material group list --output material_groups.json
luma-cli material group describe vlm_ai --output materials.json
luma-cli material search --materials materials.json --query "AI assistant" --limit 5 --output material_matches.json
```

## Agent Rules

- Prefer friendly names from `asset list` when available.
- Prefer stable asset names or IDs returned by CLI commands.
- Upload local files before referencing them in cloud-only workflows.
- Keep asset upload separate from creative workflow planning.
- For local material libraries, prefer `material group describe` over hand-building a materials file.
- Use `material search` before PIP planning when the script is long or the material group has many candidates.
