---
name: luma-workflow-original-ip-talk
description: Create an original IP spoken-video workflow from profile memory: original topic/script, digital-human narration, PPT-style knowledge visuals, subtitle, and final MP4.
---

# Luma Original IP Talk

Read `../luma-shared/SKILL.md` first. This workflow creates one original knowledge/IP spoken video
from a profile. It is not a viral-remix workflow.

```text
profile -> original content script -> digital human -> align.json -> PPT video -> avatar PiP -> subtitles -> final video
```

## Boundary

Use this workflow when the user wants an original account/IP video based on the creator profile,
stance, audience, and history.

Do not use this workflow when:

- the video should imitate a viral reference or competitor structure. Use `luma-workflow-viral-remix`.
- the user only has a finished script and wants a digital-human video. Use `luma-digital-human`.
- the user only wants `align.json` rendered into PPT MP4. Use `luma-ppt-video`.

This workflow may call:

- `luma-profile-onboarding` when no usable profile exists.
- `luma-content-script` for original topic, plan, material, script, and script review.
- `luma-digital-human` for TTS, voice, avatar, and lip-sync.
- `luma-ppt-video` for PPT-style knowledge video and optional digital-human PiP.
- `luma-subtitle` for subtitles.
- `luma-assets` to select voice/avatar assets.

## Default Output Mode

Default mode is:

```text
PPT main canvas + circular digital human at bottom-left
```

This keeps information density high and makes the video feel like a knowledge presentation with an
IP narrator. Do not default to full-screen avatar plus occasional PPT inserts until the user asks for
that more complex scene treatment.

## Standard Files

- `01_profile.json`
- `02_content_history.json`
- `03_topic_review.json`
- `04_longform_plan.json`
- `05_plan_review.json`
- `06_material_assets.json`
- `07_script.json`
- `08_script_review.json`
- `09_tts.wav`
- `10_digital_human.mp4`
- `11_align.json`
- `12_ppt/index.html`
- `12_ppt/config.json`
- `13_ppt.mp4`
- `14_ppt_with_avatar.mp4`
- `15_subtitle.mp4`
- `16_cover.jpg`

## Procedure

1. Load or create the profile.

   ```bash
   luma-cli --json profile current
   luma-cli --json profile get <profile_id>
   ```

   If the profile does not exist, use `luma-profile-onboarding` first. The profile is the creative
   source of this workflow.

2. Run the original content script workflow.

   Use `luma-content-script` to avoid used topics, discover/select a fresh original topic, submit
   `plan.review`, find materials, write the script locally, and submit `script.review`.

   Do not accept a `research.run` response as `02_raw_signals.json`. Original-topic discovery must
   combine Douyin/social signals and websearch through `content topic mine`, and must pass the
   source-diversity and minimum-signal gates in `luma-content-script`.

   Output must include a reviewed `07_script.json` with `full_script`.

3. Select voice and avatar.

   ```bash
   luma-cli asset list voice
   luma-cli asset list roles
   ```

   Use existing profile/default assets when possible. Do not invent an avatar. If no avatar exists,
   stop and ask the user to select/upload one, or continue with PPT-only mode.

4. Generate TTS and digital-human narration.

   ```bash
   luma-cli --json tts --file transcript.txt --voice <voice_name> --output 09_tts.wav
   luma-cli lipsync --avatar <role_name> --audio-key <audio_object_key> --output 10_digital_human.mp4
   ```

   Keep `transcript.txt` exactly aligned to the final spoken script.

5. Create `11_align.json`.

   Prefer the available subtitle/ASR alignment atom used elsewhere in the project. The output must
   include timed `sentence_units[]` or compatible `segments[]`.

6. Design and build the production PPT from align, script sections, and material assets.

   Use `luma-ppt-video` production flow. First create a storyboard that maps logical claims,
   `material_asset_ids`, slide layout types, and timed sentence IDs. Then author a custom
   `12_ppt/index.html` and valid `12_ppt/config.json`.

   Do not use `build_demo_ppt.py` for final output. It is a smoke-test utility with a deliberately
   uniform layout and is not a production presentation generator.

7. Render PPT video locally.

   ```bash
   python <luma-ppt-video>/scripts/render_ppt_video.py \
     --config 12_ppt/config.json \
     --ffmpeg "$(luma-cli runtime path ffmpeg)" \
     --output 13_ppt.mp4
   ```

   If local Chromium/Playwright is missing, install local dependencies or pause with clear setup
   instructions. Cloud PPT render is not the default for this free/local-first workflow.

8. Overlay the digital human.

   ```bash
   python <luma-ppt-video>/scripts/overlay_avatar.py \
     --slide 13_ppt.mp4 \
     --avatar 10_digital_human.mp4 \
     --output 14_ppt_with_avatar.mp4 \
     --ffmpeg "$(luma-cli runtime path ffmpeg)"
   ```

   If face framing is poor, extract a thumbnail and adjust `--bbox`.

9. Add subtitles and cover.

   ```bash
   luma-cli subtitle 14_ppt_with_avatar.mp4 --transcript transcript.txt --output 15_subtitle.mp4
   luma-cli cover frame 15_subtitle.mp4 --output 16_cover.jpg
   ```

   If the avatar was skipped, use `13_ppt.mp4` as the subtitle input.

## Quality Rules

- The script must be original to the profile; do not quietly imitate a viral source.
- PPT slides should explain claims visually, not repeat the entire script as dense paragraphs.
- Use collected source screenshots and generated explanatory components from `06_material_assets.json`.
  Do not silently replace missing materials with text-only slides.
- Reserve the bottom-left avatar zone before layout. For a 380px circular avatar at 60px margins,
  keep core text, evidence, labels, and progress UI out of `x=0..500, y=560..1080`.
- Use at least four layout types in a normal multi-slide video, and never repeat the same layout more
  than twice consecutively.
- Use large typography for phone readability. For 1920x1080, body text should usually be at least
  38px.
- The digital human is the narrator, not the main canvas. Keep it bottom-left by default.
- Preserve topic/script review outputs. Do not let PPT design rewrite the approved viewpoint.

## Cost Boundary

- Content/script review and digital-human generation use existing backend costs.
- PPT rendering is local-first and free from backend worker cost in this version.
- Do not promise cloud PPT render unless a future paid/limited worker is explicitly available.
