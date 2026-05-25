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

## 文案是根基：严禁跳过 Research

爆款仿写的核心是"仿写"，不是"原创"。文案的选题、结构、节奏必须基于真实爆款数据，绝不能凭 AI 自己拍脑袋编。

### 为什么 Step 0 (Research) 不能跳过

- 没有数据支撑的文案是盲猜。你不知道什么选题正在爆、什么结构观众买单、什么钩子点击率高。
- 仿写的前提是有对标。Step 0 输出的是：热门关键词、对标视频链接、爆款标题、点赞量、口播/非口播分类。这些信息决定了 Step 1 写什么。
- 跳过 Step 0 直接自己写 = 把"仿写"变成了"盲写"。后面 TTS、lipsync、字幕、BGM 做得再好，方向错了全白费。

### Agent 执行规则（强制）

1. **Step 0 Research 不可跳过。** 不管用户有没有明确要求，必须先跑 `research run`。如果用户说"随便写一个"，你要拒绝，告诉他需要数据支撑选题。

2. **Step 1 必须基于 Research 输出。** 改写时心里要有对标视频：它的钩子是什么、结构是几段式、节奏是快是慢。source_script.txt 要能追溯到 `step0_content_research.json` 里的某个具体爆款。

3. **禁止 AI 自己编文案。** 不允许在没有 research 数据的情况下，凭"我知道这类视频怎么写"直接产出脚本。你看过的训练数据不是当前的抖音热榜。

4. **Research 结果要展示给用户。** 跑完 Step 0 后，列出找到的关键词、Top 3 对标视频（标题+点赞量），让用户知道文案的选题依据是什么。

## Standard Files

- `step0_content_research.json`
- `step0_content_research.csv`
- `step0_keywords.json`
- `step0_keywords.csv`
- `step1_rewrite.json`
- `transcript.txt`
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
   luma-cli project create viral-remix
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
   Save the rewritten text as `transcript.txt` for later subtitle steps (avoids redundant ASR).

4. Generate speech from the rewritten text:
   ```bash
   luma-cli --json tts --file transcript.txt --voice 男声3 --speech-rate 1.1 --output step2_tts.wav
   ```
   The `--json` flag outputs `audio_object_key` which can be passed directly to lipsync, avoiding a redundant upload.

5. Generate digital-human video (use `--audio-key` to reference the cloud audio directly):
   ```bash
   luma-cli lipsync --avatar 数字人男 --audio-key <audio_object_key> --random-start --output step3_lipsync.mp4
   ```
   If `--audio-key` is omitted, lipsync falls back to the project's `latest_tts_key`, then to `--audio` file upload.

6. Segment text and build scene units:
   ```bash
   luma-cli subtitle transcript.txt --text --segments-output step4_segments.json --no-effects --no-highlight
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

9. Add subtitles (uses `--transcript` to skip ASR since we already have the exact script):
   ```bash
   luma-cli subtitle step4_picture_in_picture.mp4 --transcript transcript.txt --output step5_subtitle.mp4
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
