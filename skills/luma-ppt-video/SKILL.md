---
name: luma-ppt-video
description: Render a PPT-style knowledge video from an aligned spoken script. Use when the input is align.json or timed sentence/segment JSON and the desired output is a presentation MP4, optionally with a digital-human picture-in-picture overlay.
metadata:
  relatedSkills: ["luma-shared", "luma-digital-human", "luma-workflow-original-ip-talk"]
---

# Luma PPT Video

Read `../luma-shared/SKILL.md` first. This is a video-format execution skill:

```text
align.json + script + materials -> storyboard -> custom PPT HTML/config -> ppt.mp4 -> optional digital-human PiP -> final.mp4
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
- Reviewed script JSON with section, claim, and `material_asset_ids`.
- Material asset manifest with local paths, source URLs, and generated-component specs.

## Standard Files

- `align.json`
- `index.html`
- `config.json`
- `ppt.mp4`
- `digital_human.mp4`
- `ppt_with_avatar.mp4`

## Production Flow

1. Analyze the complete spoken script. Group timed sentences by logical point, not by a fixed number
   of sentences. One slide carries one main claim.

2. Write a storyboard before HTML. Each slide must record:

   - sentence IDs and timing
   - main claim and visual intent
   - layout type
   - ready material asset IDs or generated-component spec
   - source attribution when evidence is shown
   - whether the avatar safe zone is active

3. Use varied layouts chosen for the content:

   - hero thesis / large-number emphasis
   - scenario or dialogue
   - contrast / before-after
   - three-state or decision board
   - process / timeline
   - evidence screenshot with Chinese annotation
   - matrix / table
   - checklist / conclusion

   A normal multi-slide video must use at least four layout types. Do not repeat one layout more than
   twice consecutively. Variation must express different information structures, not merely change
   colors.

4. Author custom `index.html` and `config.json`, then render and inspect representative frames.
   `build_demo_ppt.py` is prohibited for final delivery.

5. Overlay the avatar and verify that no important content is hidden.

## Avatar Safe Zone

The default 1920x1080 composition uses a 380px circular avatar with 60px left/bottom margins.
Reserve `x=0..500, y=560..1080` for the avatar, ring, shadow, and breathing room. Core text, diagrams,
evidence screenshots, captions, and progress UI must not enter this rectangle.

Prefer a main content grid beginning at `x >= 520` on avatar slides. A full-screen background image
may extend underneath only when no meaningful content exists in the reserved zone.

## Smoke-Test Demo

Use this only to validate align parsing and the local renderer:

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
- Use real ready materials when the storyboard names them. A text card is not a substitute for a
  missing screenshot or visual.
- Inspect at least the opening, one evidence slide, one dense slide, and the final slide after avatar
  overlay.
- Do not commit ffmpeg binaries. Use `luma-cli runtime install ffmpeg`.

## Dependencies

- Python 3.10+
- Playwright Python package and browser:

  ```bash
  pip install playwright pillow
  python -m playwright install chromium
  ```

- ffmpeg from `luma-cli runtime install ffmpeg` or system PATH.
