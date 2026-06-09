---
name: luma-digital-human
version: 0.1.0
description: "Generate digital-human short videos with Luma / 拾光 / 拾光智能体 / 拾光工具 by composing voice clone, TTS, avatar, lip-sync, subtitle, and enhancement tools."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli tools list"
  category: "capability"
  entrypoint: true
  relatedSkills: ["luma-shared", "luma-subtitle", "luma-ppt-video", "luma-workflow-viral-remix"]
  aliases: ["拾光数字人", "拾光智能体", "拾光工具", "Luma digital human", "数字人", "口播视频"]
---

# Luma Digital Human

Use this skill when an agent needs to create a digital-human spoken video from script text, a voice, and an avatar.

Read `../luma-shared/SKILL.md` first for common auth, project, output, and artifact rules.

## Boundary

This is an execution skill, not a content strategy workflow. Use it when the script is already
approved and the task is to produce a digital-human spoken video:

```text
approved script -> TTS -> avatar/lip-sync -> subtitles -> optional enhancement
```

Do not use this skill to choose a topic, imitate a viral reference, decide the writing angle, or
rewrite the content strategy. Use `luma-content-ip-writing` for original script creation, and
`luma-workflow-viral-remix` for viral-reference remix workflows.

## Asset First

Inspect available voices and avatars:

```bash
luma-cli asset list voice
luma-cli asset list roles
```

Avatar group rules:

- Digital-human avatars are video assets in the `roles` group.
- `avatar` and `avatars` are only natural-language aliases. In commands, use `roles`.
- Default/common avatars should be available to every user through `luma-cli asset list roles`.
- Do not conclude there is no avatar until `luma-cli asset list roles` has been checked.
- Do not run `luma-cli asset list avatar`; translate that intent to `luma-cli asset list roles`.
- Do not use `image generate` to create an avatar. A generated image is not a lip-sync avatar video.
- Do not upload a generated image to `roles`.
- Only upload to `roles` when the user explicitly provides a local avatar/source-role video.
- If no suitable role appears, stop and ask the user to choose or upload a role video. Do not invent one.

If the user provides a reference voice sample, clone it first:

```bash
luma-cli voice clone ./voice.wav --name my_voice
```

If the user provides a video and says they want the sound, voice, tone, or audio from it, treat the video as a voice source and use voice clone. Do not upload that video as a `roles` avatar/source-role asset unless the user explicitly says they want the person/visual in the video to appear.

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
   luma-cli --json tts "script text" --voice my_voice --speech-rate 1.1 --output step2_tts.wav
   ```
   The `--json` flag returns `audio_object_key` in the output envelope. Use this key in step 3 to skip a redundant upload.

3. Generate lip-sync video (prefer `--audio-key` over `--audio`):
   ```bash
   luma-cli asset list roles
   luma-cli lipsync --avatar <selected_role_name> --audio-key <audio_object_key> --output step3_lipsync.mp4
   ```
   If `--audio-key` is omitted, lipsync falls back to the project's `latest_tts_key`, then to `--audio` file upload.
4. Add subtitles:
   ```bash
   luma-cli subtitle step3_lipsync.mp4 --output step5_subtitle.mp4
   ```
5. Optionally enhance:
   ```bash
   luma-cli enhance step5_subtitle.mp4 --scale 2
   ```

## Agent Notes

- **Script must come from research, not imagination.** If the script is for a short-video production, the text source must be backed by `luma-cli research run` data or a known viral reference. Never invent a script topic without data support. See `../luma-workflow-viral-remix/SKILL.md` for the full research → rewrite flow.
- Use `voice.clone` when a user provides a voice sample.
- Uploaded videos are ambiguous. If the user did not clearly say whether the video is for voice clone, avatar/source role, PIP material, ASR/rewrite, or video processing, ask a short confirmation before choosing a workflow.
- Do not assume a user-uploaded video is a digital-human avatar. If the user asks to use "the voice/audio/sound inside this video", extract or use audio for voice clone instead.
- Use `asset.list voice` and `asset.list roles` when the user asks what is available.
- Never use `image.generate` or `video.generate` to fabricate a missing avatar during a production workflow. Use default `roles` assets or ask the user.
- Use the latest project TTS output for lip-sync unless the user explicitly provides `--audio`.
- Keep the script text outside media commands until it is final enough for this generation attempt.
- Do not enhance every draft; enhance only the selected final render.
- Keep script revisions outside the media commands. The CLI should receive the final text for each generation attempt.
- Use advanced backend parameters only when the user asks for them:
  - TTS: `--trim-long-silence`
  - Lip-sync: `--random-start`, `--guidance-scale`, `--num-inference-steps`, `--no-superres`, `--superres-scale`, `--multi-shot-json`
