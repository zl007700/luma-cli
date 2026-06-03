---
name: luma-subtitle
version: 0.1.0
description: "Create short-video subtitles with Luma / 拾光 / 拾光工具. Use ASR, segmentation, styling, and burn-in as composable steps; keep editorial decisions in the agent instructions."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli tools describe subtitle.render"
  category: "capability"
  entrypoint: true
  relatedSkills: ["luma-shared", "luma-workflow-viral-remix"]
  aliases: ["拾光字幕", "拾光工具", "Luma subtitles", "字幕生成", "字幕烧录"]
---

# Luma Subtitle

Use this skill when an agent needs to add subtitles to a video or generate styled subtitle assets from text.

Read `../luma-shared/SKILL.md` first for common output and artifact rules.

## Decision Rules

- If the input is a video and the user wants subtitles on that video, run the integrated pipeline directly:
  ```bash
  luma-cli subtitle input.mp4 --output step5_subtitle.mp4
  ```
- If the exact script/transcript is already known, pass it with `--transcript`; do not use `--text` for a video:
  ```bash
  luma-cli subtitle input.mp4 --transcript transcript.txt --output step5_subtitle.mp4
  ```
- Use `--text` only when the input is raw text and the desired output is an ASS/segments draft without burning into a real video. Text mode uses estimated timing and is not suitable for precise subtitles on an existing video.
- If the agent needs to inspect or rewrite transcript text first, call `asr.transcribe` before subtitle generation, then render with `--transcript`.
- Do not hand-write SRT/ASS files, do not guess timestamps, and do not manually extract audio for ASR unless the CLI command failed and the user approves debugging.
- Keep wording, tone, and subtitle rhythm decisions in the skill/agent layer, not hidden in CLI defaults.

## Commands

```bash
luma-cli subtitle input.mp4 --project demo
luma-cli subtitle input.mp4 --max-chars 12 --font-size 56
luma-cli subtitle input.mp4 --transcript transcript.txt --output step5_subtitle.mp4
luma-cli subtitle --text "raw script text" --no-effects
```

## Practical Guidance

- Use shorter segments for sales, live-commerce, and high-energy narration.
- Disable effects for formal or compliance-sensitive content.
- Prefer `--project` when the video is part of a multi-step production, so ASS, effects, and final outputs are organized.
- If ASR accuracy matters, run `luma-cli asr <file>` first and inspect the transcript before rendering, then pass the approved text through `--transcript`.
- If a previous subtitle attempt produced bad timing or truncated text, rerun `luma-cli subtitle` with the correct mode. Do not repair by editing ASS/SRT manually.

## Expected Outputs

- Burned subtitle video when input is a video.
- `.ass` subtitle file in text mode.
- Project history entry when a project is active.
