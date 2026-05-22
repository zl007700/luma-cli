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

Sync agent skills:

```bash
luma-cli skills sync
```

Equivalent command:

```bash
npx -y skills add zl007700/luma-cli -g -y
```

Update both the CLI and skills:

```bash
luma-cli update
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
| Content research | `research run`, `research export`, `research keywords` |
| Script rewrite | `script rewrite` |
| Voice clone | `voice clone`, `voice list` |
| Text to speech | `tts` |
| Digital human | `lipsync` |
| Materials | `material describe`, `material group list`, `material group describe`, `material search`, `material merge`, `material understand` |
| Picture-in-picture | `pip scene`, `pip match`, `pip plan`, `pip render` |
| Subtitles | `subtitle` |
| BGM | `bgm mix` |
| Cover | `cover frame`, `cover render` |
| Project workspace | `project create/use/info`, `project artifact list/schema` |

## Viral Remix Workflow

See:

```text
skills/luma-workflow-viral-remix/SKILL.md
```

Typical flow:

```bash
luma-cli project create viral-demo
luma-cli project use viral-demo

luma-cli research run --role "AI tool founder looking for short-video topics" --output step0_content_research.json
luma-cli research export --input step0_content_research.json --output step0_content_research.csv
luma-cli research keywords --input step0_content_research.json --output step0_keywords.json --csv step0_keywords.csv

luma-cli script rewrite --input source_script.txt --output step1_rewrite.json
luma-cli tts "<rewritten_text>" --voice male3 --output step2_tts.wav
luma-cli lipsync --avatar avatar_male --audio step2_tts.wav --output step3_lipsync.mp4

luma-cli material group describe vlm_ai --output step4_materials_enriched.json
luma-cli material search --materials step4_materials_enriched.json --query "<rewritten_text>" --limit 8 --output step4_material_matches.json
luma-cli pip scene --segments step4_segments.json --output step4_scene_units.json
luma-cli pip match --scenes step4_scene_units.json --materials step4_materials_enriched.json --mode auto --output step4_material_matches.json
luma-cli pip plan --segments step4_segments.json --materials step4_materials_enriched.json --output step4_picture_in_picture_plan.json
luma-cli pip render step3_lipsync.mp4 --plan step4_picture_in_picture_plan.json --output step4_picture_in_picture.mp4

luma-cli subtitle step4_picture_in_picture.mp4 --output step5_subtitle.mp4
luma-cli cover generate step4_picture_in_picture.mp4 --title "Cover title" --count 12 --output-dir step6_covers
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

## Local Material Library

The default local material library lives under `~/.luma/material-library`. Import reusable material groups once, then agents can refer to them by group name:

```bash
luma-cli material library path
luma-cli material library import ./material_library/groups/vlm_ai --replace
luma-cli material group list
luma-cli material group describe vlm_ai --output materials.json
```

## Skills Distribution

Luma distributes the command-line tool and agent skills separately: npm installs the CLI, while skills are synced through the skills installer. Users usually install the full Luma skill pack once:

```bash
npm install -g @lumageo/luma-cli
luma-cli skills sync
```

`luma-cli skills sync` installs Luma agent skills and writes a local sync stamp. `luma-cli update` updates the npm package and syncs skills in one command. See [docs/SKILLS.md](./docs/SKILLS.md).

## Project Structure

`luma-cli` is organized around a command shell, atomic capability modules, and agent-facing skills:

```text
cmd/luma-cli/             CLI entrypoint
internal/commands/        Command argument parsing, output, and light orchestration
internal/commands/pip_*   Picture-in-picture scene, match, and render implementation
internal/commands/material_* Local material libraries, groups, and search
internal/clientruntime/   Local runtime cache for ffmpeg, fonts, BGM, and templates
cloud/                    PikGeo / Luma backend API client
project/                  Project workspace, history, and artifacts
shortcuts/                Agent-discoverable tool descriptions
skills/                   Agent workflow instructions
scripts/                  npm install and run entrypoints
```

Maintenance rules:

- Keep CLI commands atomic; do not hard-code full business workflows in Go.
- Put multi-step video workflows in `skills/` and let agents execute them through intermediate artifacts.
- Use Luma cloud services for advanced material understanding and semantic matching.
- Keep local logic focused on stable cross-platform work: files, resource cache, ffmpeg rendering, and result formatting.
- When adding a capability, update `tools describe` / `shortcuts` so agents can discover the parameters automatically.
