---
name: luma-viral-remix-workflow
version: 0.1.0
description: "Create a short-video remix from content research through rewrite, voice, digital human, PIP materials, subtitles, and cover using luma-cli atomic tools."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli tools list"
---

# 爆款仿写 Workflow

Use this skill when the user wants an agent to turn a role, topic, or viral reference into a finished spoken short video.

This is an orchestration skill. Keep each step as an explicit `luma-cli` atom call and save every intermediate file.

## Standard Files

Use these names unless the user asks for different paths:

- `step0_content_research.json`
- `step0_content_research.csv`
- `step1_rewrite.json`
- `step2_tts.wav`
- `step3_lipsync.mp4`
- `step4_segments.json`
- `step4_materials.json`
- `step4_materials_enriched.json`
- `step4_picture_in_picture_plan.json`
- `step4_picture_in_picture.mp4`
- `step5_subtitle.mp4`
- `step6_cover_frame.png`
- `step6_cover.jpg`

## Flow

1. Create or select a project:
   ```bash
   luma-cli project create viral-remix --dir ./viral-remix
   luma-cli project use viral-remix
   ```

2. Search for references from a role description:
   ```bash
   luma-cli research run --role "AI工具创业者，想找适合口播拆解的爆款选题" --mode precise --date-range 7d --output step0_content_research.json
   luma-cli research export --input step0_content_research.json --output step0_content_research.csv
   ```

3. Pick one reference title or script and rewrite it:
   ```bash
   luma-cli script rewrite --input source_script.txt --length short --output step1_rewrite.json
   ```

4. Extract `rewritten_text` from `step1_rewrite.json`, then generate speech:
   ```bash
   luma-cli tts "<rewritten_text>" --voice 男声3 --speech-rate 1.1 --output step2_tts.wav
   ```

5. Generate the digital human video:
   ```bash
   luma-cli lipsync --avatar 数字人男 --audio step2_tts.wav --random-start --output step3_lipsync.mp4
   ```

6. Create timed text segments for PIP planning:
   ```bash
   luma-cli subtitle "<rewritten_text>" --text --segments-output step4_segments.json --no-effects --no-highlight
   ```

7. Prepare local PIP materials:
   ```bash
   luma-cli material describe ./materials --output step4_materials.json
   luma-cli material merge --materials step4_materials.json --meta ./materials_meta --output step4_materials_enriched.json
   ```

   If the materials have not been understood yet, run `material understand` for each useful material first:
   ```bash
   luma-cli material understand ./materials/a.jpg --output a.meta.json --descriptor-output a.material.json
   ```

8. Plan and render PIP:
   ```bash
   luma-cli pip plan --segments step4_segments.json --materials step4_materials_enriched.json --output step4_picture_in_picture_plan.json
   luma-cli pip render step3_lipsync.mp4 --plan step4_picture_in_picture_plan.json --output step4_picture_in_picture.mp4
   ```

   If no insert is matched, continue with `step3_lipsync.mp4` as the subtitle input. Do not force unrelated materials into the video.

9. Add subtitles:
   ```bash
   luma-cli subtitle step4_picture_in_picture.mp4 --output step5_subtitle.mp4 --segments-output step5_subtitle_segments.json
   ```

10. Create a cover:
    ```bash
    luma-cli cover frame step5_subtitle.mp4 --time 1.0 --output step6_cover_frame.png
    luma-cli cover render step6_cover_frame.png --title "封面标题" --subtitle "一句补充说明" --output step6_cover.jpg
    ```

## Agent Rules

- Always inspect tool details with `luma-cli --json tools describe <tool_id>` if a command fails or the backend changes.
- Keep `step0_content_research.csv` for human review, even if the agent also reads JSON.
- Use the rewritten script as the single source for TTS, segmentation, subtitles, and cover title extraction.
- PIP materials are optional. A non-match is acceptable when the material library does not fit the script.
- Do not ask the user to manually copy object keys. Prefer friendly names, local paths, and output paths.
- Do not invent missing video files. If a cloud task does not download an output, report the task id and stop at that boundary.
