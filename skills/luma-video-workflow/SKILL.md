---
name: luma-video-workflow
version: 0.1.0
description: "Use luma-cli atomic tools to create short-video assets. Prefer composing ASR, TTS, LipSync, subtitle, enhancement, asset, and task commands instead of relying on one large opaque workflow."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli tools list"
---

# Luma Video Workflow

Use this skill when an agent needs to produce or inspect short-video generation work through `luma-cli`.

## Core Rule

Treat `luma-cli` commands as atomic tools. Keep business orchestration in the agent plan:

1. Discover tools with `luma-cli --json tools list`.
2. Inspect a specific tool with `luma-cli --json tools describe <tool_id>`.
3. Call one atomic command.
4. Read the returned task id, object key, or output path.
5. Decide the next step from the result.

Do not assume one CLI command should hide the whole product workflow.

## Common Atoms

- `asr.transcribe`: `luma-cli asr <file> --language zh`
- `tts.synthesize`: `luma-cli tts <text> --voice <name> --speech-rate 1.1`
- `lipsync.create`: `luma-cli lipsync --avatar <name> --audio <file>`
- `video.enhance`: `luma-cli enhance <video> --scale 2`
- `subtitle.render`: `luma-cli subtitle <video> [options]`
- `asset.upload`: `luma-cli asset upload <file> --group <name>`
- `asset.list`: `luma-cli asset list <group>`
- `task.status`: `luma-cli task status <task_id>`

## Recommended Digital Human Flow

1. List available voices and avatars if names are unknown:
   ```bash
   luma-cli asset list voice
   luma-cli asset list roles
   ```
2. Generate speech:
   ```bash
   luma-cli tts "script text" --voice 男声3
   ```
3. Generate lip-sync video:
   ```bash
   luma-cli lipsync --avatar 数字人男
   ```
4. Add subtitles when needed:
   ```bash
   luma-cli subtitle output.mp4 --project demo
   ```
5. Enhance final output only after the creative edit is stable:
   ```bash
   luma-cli enhance output_subtitled.mp4 --scale 2
   ```

## Agent Behavior

- Prefer explicit intermediate files and project history when the user may iterate.
- Use `project create` and `project use` for multi-step workflows.
- Use `task status` when a long-running task times out or the agent resumes later.
- Keep final outputs in the project `output/` directory when a project is active.
- Advanced backend parameters are available, but should be explicit user choices:
  - `luma-cli tts ... --trim-long-silence`
  - `luma-cli lipsync ... --random-start --guidance-scale 1.0 --num-inference-steps 15 --no-superres --superres-scale 2 --multi-shot-json payload.json`
