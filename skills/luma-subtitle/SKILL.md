---
name: luma-subtitle
version: 0.1.0
description: "Create short-video subtitles with luma-cli. Use ASR, segmentation, styling, and burn-in as composable steps; keep editorial decisions in the agent instructions."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli tools describe subtitle.render"
---

# Luma Subtitle

Use this skill when an agent needs to add subtitles to a video or generate styled subtitle assets from text.

## Decision Rules

- If the input is a video, use `subtitle.render` directly for the current integrated pipeline.
- If the agent needs to inspect or rewrite transcript text first, call `asr.transcribe` before subtitle generation.
- If the input is already text, use `luma-cli subtitle --text "<text>"` to generate ASS timing estimates.
- Keep wording, tone, and subtitle rhythm decisions in the skill/agent layer, not hidden in CLI defaults.

## Commands

```bash
luma-cli subtitle input.mp4 --project demo
luma-cli subtitle input.mp4 --max-chars 12 --font-size 56
luma-cli subtitle --text "raw script text" --no-effects
```

## Practical Guidance

- Use shorter segments for sales, live-commerce, and high-energy narration.
- Disable effects for formal or compliance-sensitive content.
- Prefer `--project` when the video is part of a multi-step production, so ASS, effects, and final outputs are organized.
- If ASR accuracy matters, run `luma-cli asr <file>` first and inspect the transcript before rendering.

## Expected Outputs

- Burned subtitle video when input is a video.
- `.ass` subtitle file in text mode.
- Project history entry when a project is active.
