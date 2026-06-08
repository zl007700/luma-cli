---
name: luma-ppt-video
description: Render a PPT-style knowledge video from an aligned spoken script. Use when the input is align.json or timed sentence/segment JSON and the desired output is a presentation MP4, optionally with a digital-human picture-in-picture overlay.
metadata:
  relatedSkills: ["luma-shared", "luma-digital-human", "luma-workflow-original-ip-talk"]
---

# Luma PPT Video

Read `../luma-shared/SKILL.md` first. This is a video-format execution skill:

```text
align.json -> demo PPT HTML/config -> ppt.mp4 -> optional digital-human PiP -> final.mp4
```

It does not choose topics, write scripts, or imitate viral references. Use `luma-content-script` for
original script creation, `luma-digital-human` for making the avatar/lip-sync video, and
`luma-workflow-viral-remix` for reference-based end-to-end production.

## Inputs

Required:

- `align.json` with `sentence_units[]`, `sentences[]`, or `segments[]` containing `start`, `end`, and `text`.

Optional:

- `digital_human.mp4`: a lip-sync/avatar video to overlay at bottom-left.
- `audio.wav` or `audio.mp3`: narration audio to mux into the pure PPT video.
- Existing `index.html` and `config.json` if the agent has already designed custom slides.

## Standard Files

- `align.json`
- `index.html`
- `config.json`
- `ppt.mp4`
- `digital_human.mp4`
- `ppt_with_avatar.mp4`

## Quick Demo Flow

1. Create a project directory and generate demo slides from align:

   ```bash
   python <skill_dir>/scripts/build_demo_ppt.py \
     --align align.json \
     --output-dir ppt_video_demo
   ```

2. Render the PPT video:

   ```bash
   python <skill_dir>/scripts/render_ppt_video.py \
     --config ppt_video_demo/config.json \
     --ffmpeg "$(luma-cli runtime path ffmpeg)"
   ```

3. Optionally overlay a digital human at bottom-left:

   ```bash
   python <skill_dir>/scripts/overlay_avatar.py \
     --slide ppt_video_demo/ppt.mp4 \
     --avatar digital_human.mp4 \
     --output ppt_video_demo/ppt_with_avatar.mp4 \
     --ffmpeg "$(luma-cli runtime path ffmpeg)"
   ```

   If the face framing is poor, first run:

   ```bash
   python <skill_dir>/scripts/overlay_avatar.py --avatar digital_human.mp4 --thumb-only
   ```

   Inspect `avatar_thumb.jpg`, then pass a bounding box:

   ```bash
   python <skill_dir>/scripts/overlay_avatar.py \
     --slide ppt_video_demo/ppt.mp4 \
     --avatar digital_human.mp4 \
     --bbox 0 350 1080 1500 \
     --output ppt_video_demo/ppt_with_avatar.mp4 \
     --ffmpeg "$(luma-cli runtime path ffmpeg)"
   ```

## Design Rules

- For 1920x1080 source video, body text should usually be at least 38px. Smaller text often becomes unreadable on phones.
- One slide should carry one point. Use concise titles and a few strong phrases, not paragraph walls.
- Avoid network fonts in generated demo HTML; the renderer should work offline.
- Digital human overlay is optional and should not cover the main claim. Default circle size is 380px at bottom-left.
- Do not commit ffmpeg binaries. Use `luma-cli runtime install ffmpeg`.

## Dependencies

- Python 3.10+
- Playwright Python package and browser:

  ```bash
  pip install playwright pillow
  python -m playwright install chromium
  ```

- ffmpeg from `luma-cli runtime install ffmpeg` or system PATH.
