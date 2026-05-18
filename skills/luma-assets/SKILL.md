---
name: luma-assets
version: 0.1.0
description: "Manage Luma cloud assets used by generation tools, including voices, avatars, media inputs, and named groups."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli tools describe asset.upload"
---

# Luma Assets

Use this skill when an agent needs to inspect or upload reusable media assets for Luma workflows.

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
```

## Agent Rules

- Prefer friendly names from `asset list` when available.
- Use full `object_key` only when a command needs an exact asset.
- Upload local files before referencing them in cloud-only workflows.
- Keep asset upload separate from creative workflow planning.
