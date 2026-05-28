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

2. **Step 1 必须基于真实对标视频转写。** Research 只负责发现候选爆款，不能直接当 source script。必须从 `step0_content_research.json` 里挑选 2-3 个高潜力对标视频，下载视频并 ASR 转写，source script 必须能追溯到这些转写结果。

3. **禁止 AI 自己编 source script。** 不允许只看标题、点赞量、关键词就凭空写 `source_script.txt`。如果对标视频无法下载或 ASR 失败，必须换参考视频；仍无法获得转写时，暂停并向用户说明无法完成"仿写"依据。

4. **Research 结果要展示给用户。** 跑完 Step 0 后，列出找到的关键词、Top 3 对标视频（标题+点赞量），让用户知道文案的选题依据是什么。

## Standard Files

- `step0_content_research.json`
- `step0_content_research.csv`
- `step0_keywords.json`
- `step0_keywords.csv`
- `references/ref_01.mp4`
- `references/ref_01_asr.json`
- `references/ref_02.mp4`
- `references/ref_02_asr.json`
- `source_reference_bundle.md`
- `source_script.txt`
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
   Inspect the JSON/CSV and choose 2-3 real reference videos with strong hooks, clear structure, and usable `link` fields. Prefer 口播 references when the final output is a spoken digital-human video.

3. Download and transcribe the chosen references:
   ```bash
   mkdir -p references
   luma-cli --json social download "<reference_1_link>" --output references/ref_01.mp4
   luma-cli asr references/ref_01.mp4 --language zh --output references/ref_01_asr.json
   luma-cli --json social download "<reference_2_link>" --output references/ref_02.mp4
   luma-cli asr references/ref_02.mp4 --language zh --output references/ref_02_asr.json
   ```
   If using a third reference, save it as `references/ref_03.mp4` and `references/ref_03_asr.json`. `asr` accepts video directly, so a separate local audio-extraction step is not required unless ASR fails on the video file.

4. Build the source material for rewrite:
   - Read each `references/ref_XX_asr.json`.
   - Extract the original transcript, hook, argument structure, emotional turn, punchlines, and CTA.
   - Write `source_reference_bundle.md` with the selected references, original titles/links, transcript excerpts, and the reason each reference is worth copying.
   - Write `source_script.txt` as a grounded source brief: include the user's persona/positioning, the 2-3 reference viewpoints to fuse, reusable structures/hooks from the transcripts, and explicit constraints. Do not invent claims that are absent from the reference transcripts or user brief.

5. Rewrite the grounded source script:
   ```bash
   luma-cli script rewrite --input source_script.txt --length short --output step1_rewrite.json
   ```
   Save the rewritten text as `transcript.txt` for later subtitle steps (avoids redundant ASR).

6. Generate speech from the rewritten text:
   ```bash
   luma-cli --json tts --file transcript.txt --voice 男声3 --speech-rate 1.1 --output step2_tts.wav
   ```
   The `--json` flag outputs `audio_object_key` which can be passed directly to lipsync, avoiding a redundant upload.

7. Generate digital-human video (use `--audio-key` to reference the cloud audio directly):
   ```bash
   luma-cli lipsync --avatar 数字人男 --audio-key <audio_object_key> --random-start --output step3_lipsync.mp4
   ```
   If `--audio-key` is omitted, lipsync falls back to the project's `latest_tts_key`, then to `--audio` file upload.

8. Segment text and build scene units:
   ```bash
   luma-cli subtitle transcript.txt --text --segments-output step4_segments.json --no-effects --no-highlight
   luma-cli pip scene --segments step4_segments.json --output step4_scene_units.json
   ```

9. Prepare and match local PIP materials:
   ```bash
   luma-cli material group describe vlm_ai --output step4_materials_enriched.json
   luma-cli pip match --scenes step4_scene_units.json --materials step4_materials_enriched.json --mode auto --output step4_material_matches.json
   ```

10. Plan and render PIP:
   ```bash
   luma-cli pip plan --segments step4_segments.json --materials step4_materials_enriched.json --match-mode auto --output step4_picture_in_picture_plan.json
   luma-cli pip render step3_lipsync.mp4 --plan step4_picture_in_picture_plan.json --output step4_picture_in_picture.mp4
   ```

   If no insert is matched, continue with `step3_lipsync.mp4` as the subtitle input.

11. Add subtitles (uses `--transcript` to skip ASR since we already have the exact script):
   ```bash
   luma-cli subtitle step4_picture_in_picture.mp4 --transcript transcript.txt --output step5_subtitle.mp4
   ```

12. Add BGM:
    ```bash
    luma-cli bgm mix step5_subtitle.mp4 --output step6_bgm.mp4
    ```

13. Create a cover:
    ```bash
    luma-cli cover generate step4_picture_in_picture.mp4 --title "<cover_title>" --subtitle "<cover_subtitle>" --count 12 --output-dir step7_covers
    ```
    Use the original visual video for cover frames: prefer `step4_picture_in_picture.mp4`; if PIP was skipped, use `step3_lipsync.mp4`. Do not extract the cover frame from `step5_subtitle.mp4`, because burned subtitles will become part of the cover background.

## Agent Rules

- Keep every intermediate file; do not collapse the flow into one hidden step.
- `source_script.txt` is not the final script and not AI-written from memory. It is the grounded rewrite brief produced from downloaded reference transcripts plus the user's persona/angle.
- If `step0_content_research.json` only has titles/links and no transcript, do not proceed to rewrite until reference videos have been downloaded and ASR has produced usable text.
- Use the rewritten script as the single source for TTS, segmentation, subtitles, and cover text extraction.
- If the material library does not fit the script, skip PIP instead of forcing weak matches.
- Use `project artifact list` before resuming or rerunning a partial workflow.
- Report exact output paths for all generated files.
