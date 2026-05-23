---
name: luma-workflow-viral-remix
version: 0.1.0
description: "Use when the user wants a complete viral-remix short-video workflow: research, rewrite, TTS, digital human, PIP materials, subtitles, BGM, and cover."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli tools list"
  category: "workflow"
  entrypoint: true
  relatedSkills: ["luma-shared", "luma-content-research", "luma-digital-human", "luma-material", "luma-subtitle"]
---

# 爆款仿写 Workflow

Use this skill when the user wants an agent to turn a topic, account role, or viral reference into a spoken short video.

Read these first when needed:

- `../luma-shared/SKILL.md` for common project, artifact, auth, and failure rules.
- `../luma-content-research/SKILL.md` for topic search and keyword tables.
- `../luma-material/SKILL.md` for local material groups and PIP matching.
- `../luma-digital-human/SKILL.md` for voice, TTS, avatar, and lip-sync.
- `../luma-subtitle/SKILL.md` for subtitle rendering.

## When To Use

- The user asks for 爆款仿写, 对标视频, 口播短视频, 种草视频, or a complete video production run.
- The user wants all intermediate files so the agent can inspect and iterate.
- The expected output is a produced video plus cover, not just a script.

Do not use this workflow when the user only asks for one atomic operation such as TTS, subtitle, or material search.

## Standard Files

- `step0_content_research.json`
- `step0_content_research.csv`
- `step0_keywords.json`
- `step0_keywords.csv`
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
- `step6_bgm.mp4`
- `step7_covers/cover_manifest.json`
- `step7_covers/cover_01.jpg`

## Flow

1. Create or select a project:
   ```bash
   luma-cli project create viral-remix --dir ./viral-remix
   luma-cli project use viral-remix
   ```

2. Research references:
   ```bash
   luma-cli research run --role "<role_or_topic>" --mode precise --date-range 7d --output step0_content_research.json
   luma-cli research export --input step0_content_research.json --output step0_content_research.csv
   luma-cli research keywords --input step0_content_research.json --output step0_keywords.json --csv step0_keywords.csv
   ```

3. Rewrite the chosen reference or source script:
   ```bash
   luma-cli script rewrite --input source_script.txt --length short --output step1_rewrite.json
   ```

4. Generate speech from the rewritten text:
   ```bash
   luma-cli tts "<rewritten_text>" --voice 男声3 --speech-rate 1.1 --output step2_tts.wav
   ```

5. Generate digital-human video:
   ```bash
   luma-cli lipsync --avatar 数字人男 --audio step2_tts.wav --random-start --output step3_lipsync.mp4
   ```

6. Segment text and build scene units:
   ```bash
   luma-cli subtitle "<rewritten_text>" --text --segments-output step4_segments.json --no-effects --no-highlight
   luma-cli pip scene --segments step4_segments.json --output step4_scene_units.json
   ```

7. Prepare and match local PIP materials:
   ```bash
   luma-cli material group describe vlm_ai --output step4_materials_enriched.json
   luma-cli pip match --scenes step4_scene_units.json --materials step4_materials_enriched.json --mode auto --output step4_material_matches.json
   ```

8. Plan and render PIP:
   ```bash
   luma-cli pip plan --segments step4_segments.json --materials step4_materials_enriched.json --match-mode auto --output step4_picture_in_picture_plan.json
   luma-cli pip render step3_lipsync.mp4 --plan step4_picture_in_picture_plan.json --output step4_picture_in_picture.mp4
   ```

   If no insert is matched, continue with `step3_lipsync.mp4` as the subtitle input.

9. Add subtitles:
   ```bash
   luma-cli subtitle step4_picture_in_picture.mp4 --output step5_subtitle.mp4 --segments-output step5_subtitle_segments.json
   ```

10. Add BGM:
    ```bash
    luma-cli bgm mix step5_subtitle.mp4 --output step6_bgm.mp4
    ```

11. Create a cover:
    ```bash
    luma-cli cover generate step4_picture_in_picture.mp4 --title "<cover_title>" --subtitle "<cover_subtitle>" --count 12 --output-dir step7_covers
    ```
    Use the original visual video for cover frames: prefer `step4_picture_in_picture.mp4`; if PIP was skipped, use `step3_lipsync.mp4`. Do not extract the cover frame from `step5_subtitle.mp4`, because burned subtitles will become part of the cover background.

## Agent Rules

- Keep every intermediate file; do not collapse the flow into one hidden step.
- Use the rewritten script as the single source for TTS, segmentation, subtitles, and cover text extraction.
- If the material library does not fit the script, skip PIP instead of forcing weak matches.
- Use `project artifact list` before resuming or rerunning a partial workflow.
- Report exact output paths for all generated files.
