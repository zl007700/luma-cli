# luma-cli

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.23-blue.svg)](https://go.dev/)

[中文](./README.md) | English

`luma-cli` is an agent-facing video production toolkit for PikGeo / Luma. It exposes professional short-video capabilities as composable CLI atoms: content research, viral script rewriting, voice cloning, TTS, digital-human lip sync, picture-in-picture materials, subtitles, BGM, covers, and project artifacts.

The CLI stays atomic. Multi-step business workflows live in `skills/` so agents can compose reliable production flows without hard-coding glue into the command layer.

## Install

```bash
npm install -g @lumageo/luma-cli
luma-cli auth login <CARD_KEY>
luma-cli runtime install ffmpeg
```

## Discover Tools

```bash
luma-cli tools list
luma-cli tools describe tts.synthesize
luma-cli --json tools describe pip.plan
```

## Common Atoms

| Capability | Command |
| --- | --- |
| Content research | `research run`, `research export` |
| Script rewrite | `script rewrite` |
| Voice clone | `voice clone`, `voice list` |
| Text to speech | `tts` |
| Digital human | `lipsync` |
| Materials | `material describe`, `material merge`, `material understand` |
| Picture-in-picture | `pip plan`, `pip render` |
| Subtitles | `subtitle` |
| BGM | `bgm mix` |
| Cover | `cover frame`, `cover render` |
| Project workspace | `project create/use/info` |

## Viral Remix Workflow

See:

```text
skills/luma-viral-remix-workflow/SKILL.md
```

Typical flow:

```bash
luma-cli project create viral-demo
luma-cli project use viral-demo

luma-cli research run --role "AI tool founder looking for short-video topics" --output step0_content_research.json
luma-cli research export --input step0_content_research.json --output step0_content_research.csv

luma-cli script rewrite --input source_script.txt --output step1_rewrite.json
luma-cli tts "<rewritten_text>" --voice male3 --output step2_tts.wav
luma-cli lipsync --avatar avatar_male --audio step2_tts.wav --output step3_lipsync.mp4

luma-cli material describe ./materials --output step4_materials.json
luma-cli material merge --materials step4_materials.json --meta ./materials_meta --output step4_materials_enriched.json
luma-cli pip plan --segments step4_segments.json --materials step4_materials_enriched.json --output step4_picture_in_picture_plan.json
luma-cli pip render step3_lipsync.mp4 --plan step4_picture_in_picture_plan.json --output step4_picture_in_picture.mp4

luma-cli subtitle step4_picture_in_picture.mp4 --output step5_subtitle.mp4
luma-cli cover frame step5_subtitle.mp4 --output step6_cover_frame.png
luma-cli cover render step6_cover_frame.png --title "Cover title" --output step6_cover.jpg
```

## Voice Clone

```bash
luma-cli voice clone ./sample.wav --name my_voice
luma-cli voice list
luma-cli tts "Test narration" --voice my_voice --output step2_tts.wav
```

## Standard Artifacts

Recommended intermediate files:

| Step | Files |
| --- | --- |
| Research | `step0_content_research.json`, `step0_content_research.csv` |
| Rewrite | `step1_rewrite.json` |
| Audio | `step2_tts.wav` |
| Digital human | `step3_lipsync.mp4` |
| PIP | `step4_segments.json`, `step4_materials.json`, `step4_picture_in_picture_plan.json`, `step4_picture_in_picture.mp4` |
| Subtitles | `step5_subtitle.mp4` |
| Cover | `step6_cover_frame.png`, `step6_cover.jpg` |

`project.json` records history and artifacts so agents can resume, inspect, and rerun multi-step jobs.
