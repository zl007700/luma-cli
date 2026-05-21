---
name: luma-shared
version: 0.1.0
description: "Use before any Luma production workflow. Defines common luma-cli rules for auth, tool discovery, projects, artifacts, runtime resources, and safe agent behavior."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli tools list"
  category: "shared"
  entrypoint: false
---

# Luma Shared Rules

Use this skill as the common foundation for all `luma-cli` skills.

## Before Running Tools

1. Check auth when a command needs backend access:
   ```bash
   luma-cli auth status
   ```
2. Discover the exact atom before using unfamiliar parameters:
   ```bash
   luma-cli --json tools describe <tool_id>
   ```
3. For multi-step jobs, create or select a project:
   ```bash
   luma-cli project create <project_name>
   luma-cli project use <project_name>
   ```
4. Install local video runtime before local rendering:
   ```bash
   luma-cli runtime install ffmpeg
   ```

## Output Rules

- Prefer explicit `--output` paths for every step.
- Keep intermediate files; do not overwrite them unless the user asks.
- Use project artifacts to inspect or resume work:
  ```bash
  luma-cli project artifact list
  luma-cli project artifact schema
  ```
- When a command supports JSON, use `--json` for agent parsing.

## Standard Files

Use stable names unless the user gives a project convention:

- `step0_content_research.json`
- `step0_content_research.csv`
- `step0_keywords.json`
- `step1_rewrite.json`
- `step2_tts.wav`
- `step3_lipsync.mp4`
- `step4_segments.json`
- `step4_scene_units.json`
- `step4_materials_enriched.json`
- `step4_material_matches.json`
- `step4_picture_in_picture_plan.json`
- `step4_picture_in_picture.mp4`
- `step5_subtitle.mp4`
- `step6_cover_frame.png`
- `step6_cover.jpg`

## Boundary Rules

- CLI atoms should stay atomic. Do not hide a full production workflow in one command.
- Prompt-heavy logic, material understanding, semantic matching, and protected model behavior belong on the backend.
- Local logic is for stable cross-platform work: file organization, runtime cache, ffmpeg rendering, and result formatting.
- Prefer friendly names for voices, avatars, and resources. Do not ask users to manually copy internal object keys.
- Never print card keys, backend tokens, or private object credentials.

## Failure Handling

- If a backend task returns a task id but no file, report the task id and stop at that boundary.
- If PIP material matching returns zero inserts, do not force unrelated materials into the video.
- If local ffmpeg is missing, run `luma-cli runtime install ffmpeg`.
- If a command fails because parameters changed, inspect `luma-cli --json tools describe <tool_id>` before retrying.
