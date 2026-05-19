---
name: luma-digital-human
version: 0.1.0
description: "Generate digital-human short videos by composing voice, avatar, lip-sync, subtitle, and enhancement tools."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli tools list"
---

# Luma Digital Human

Use this skill when an agent needs to create a digital human video from script text, voice, and avatar assets.

## Asset First

Before generating, make sure the requested assets exist:

```bash
luma-cli asset list voice
luma-cli asset list roles
```

If the user provides a local voice or avatar file, upload it first:

```bash
luma-cli asset upload voice.wav --group voice
luma-cli asset upload avatar.mp4 --group roles
```

## Standard Flow

1. Create or select a project:
   ```bash
   luma-cli project create demo
   luma-cli project use demo
   ```
2. Generate voice:
   ```bash
   luma-cli tts "script text" --voice 男声3 --speech-rate 1.1
   ```
3. Generate lip-sync video:
   ```bash
   luma-cli lipsync --avatar 数字人男
   ```
4. Add subtitles:
   ```bash
   luma-cli subtitle output/lipsync_output.mp4 --project demo
   ```
5. Optionally enhance:
   ```bash
   luma-cli enhance output/lipsync_output_subtitled.mp4 --scale 2
   ```

## Agent Notes

- Use the latest project TTS output for lip-sync unless the user explicitly provides `--audio`.
- Do not enhance every draft; enhance only the selected final render.
- Keep user-facing script revisions outside the CLI. The CLI should receive the final text for each generation attempt.
- Use advanced backend parameters only when the user asks for them:
  - TTS: `--trim-long-silence`
  - Lip-sync: `--random-start`, `--guidance-scale`, `--num-inference-steps`, `--no-superres`, `--superres-scale`, `--multi-shot-json`
