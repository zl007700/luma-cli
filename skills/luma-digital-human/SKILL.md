---
name: luma-digital-human
version: 0.1.0
description: "Generate digital-human short videos by composing voice clone, TTS, avatar, lip-sync, subtitle, and enhancement tools."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli tools list"
---

# Luma Digital Human

Use this skill when an agent needs to create a digital-human spoken video from script text, a voice, and an avatar.

## Asset First

Inspect available voices and avatars:

```bash
luma-cli voice list
luma-cli asset list roles
```

If the user provides a reference voice sample, clone it first:

```bash
luma-cli voice clone ./voice.wav --name my_voice
```

If the user provides a local avatar video, upload it:

```bash
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
   luma-cli tts "script text" --voice my_voice --speech-rate 1.1 --output step2_tts.wav
   ```
3. Generate lip-sync video:
   ```bash
   luma-cli lipsync --avatar 数字人男 --audio step2_tts.wav --output step3_lipsync.mp4
   ```
4. Add subtitles:
   ```bash
   luma-cli subtitle step3_lipsync.mp4 --output step5_subtitle.mp4
   ```
5. Optionally enhance:
   ```bash
   luma-cli enhance step5_subtitle.mp4 --scale 2
   ```

## Agent Notes

- Use `voice.clone` when a user provides a voice sample.
- Use `voice.list` and `asset.list roles` when the user asks what is available.
- Use the latest project TTS output for lip-sync unless the user explicitly provides `--audio`.
- Do not enhance every draft; enhance only the selected final render.
- Keep script revisions outside the media commands. The CLI should receive the final text for each generation attempt.
- Use advanced backend parameters only when the user asks for them:
  - TTS: `--trim-long-silence`
  - Lip-sync: `--random-start`, `--guidance-scale`, `--num-inference-steps`, `--no-superres`, `--superres-scale`, `--multi-shot-json`
