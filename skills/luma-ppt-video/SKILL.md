---
name: luma-ppt-video
description: Render a PPT-style knowledge video from an aligned spoken script. Use when the input is align.json or timed sentence/segment JSON and the desired output is a presentation MP4, optionally with a digital-human picture-in-picture overlay.
metadata:
  relatedSkills: ["luma-shared", "luma-digital-human", "luma-workflow-original-ip-talk"]
---

# Luma PPT Video

Read `../luma-shared/SKILL.md` first. This is a video-format execution skill:

```text
align.json + script -> PPT HTML/config -> ppt.mp4 -> optional digital-human PiP -> final.mp4
```

It does not choose topics, write scripts, or imitate viral references. Use `luma-original-script` for
original script creation, `luma-digital-human` for making the avatar/lip-sync video, and
`luma-workflow-viral-remix` for reference-based end-to-end production.

## Inputs

Required:

- `align.json` with `sentence_units[]` or `segments[]` containing `sent_id`, `start`, `end`, `text`.

Optional:

- `digital_human.mp4`: a lip-sync/avatar video to overlay at bottom-left.
- `audio.wav` or `audio.mp3`: narration audio to mux into the pure PPT video.
- Reviewed script JSON with section, claim, and `material_asset_ids`.
- Material asset manifest with local paths, source URLs, and generated-component specs.

## Standard Files

- `subtitle_segments.json`  (align input, sentence_units with sent_id/start/end/text)
- `index.html`              (horizontal-sliding slideshow)
- `config.json`             (subtitle_file, html_file, output_file, ffmpeg_path, sentence_to_slide)
- `13_ppt.mp4`              (pure PPT output)
- `14_ppt_with_avatar.mp4`  (PPT + digital-human PiP)
- `15_subtitle.mp4`         (final with subtitles)
- `16_cover.jpg`            (cover frame)

---

## Step 1: Understand the Input Format

The skill works with JSON subtitle files containing timed segments. Expected format:

```json
{
  "sentence_units": [
    {
      "sent_id": "sent_0000",
      "start": 0.38,
      "end": 4.3,
      "text": "完整的句子文本"
    }
  ]
}
```

Or with segment-level timing:

```json
{
  "segments": [
    {
      "seg_id": 0,
      "start": 0.38,
      "end": 1.4,
      "text": "词段文本",
      "sent_id": "sent_0000"
    }
  ]
}
```

## Step 2: Analyze Content and Plan Slides

1. Read the subtitle JSON and understand the content flow
2. Group sentences into logical slides (1 slide per topic/point)
3. Map each sentence to a slide number
4. Create a `config.json` with the mapping:

```json
{
  "subtitle_file": "path/to/subtitle_segments.json",
  "html_file": "path/to/index.html",
  "output_file": "path/to/13_ppt.mp4",
  "ffmpeg_path": "path/to/ffmpeg.exe",
  "video_size": [1920, 1080],
  "sentence_to_slide": {
    "sent_0000": 0,
    "sent_0001": 0,
    "sent_0002": 1
  }
}
```

## Step 3: Create HTML Presentation

Create a horizontal-sliding HTML presentation (`index.html`). **Use the template below exactly as given** — do not author custom CSS from scratch, do not invent new layout types, and do not vary the visual style per slide.

### Core Design Rules (READ BEFORE WRITING HTML)

1. **No watermarks, no brand footers, no "powered by" lines.** The video is a finished product. A watermark makes it look like a demo, not a deliverable.
2. **Slides are visual companions, not teleprompters.** Never paste the full spoken script onto a slide. Each slide carries one short headline (≤8 characters per line — use `white-space: nowrap` on `.slide-title` to prevent ugly line breaks) plus at most 2-3 short bullet points. No single sentence longer than ~10 characters on screen — the audience hears the narration, the slide is the visual anchor.
3. **Avatar circle: no colored ring, tangent to left edge.** Circle diameter = **340px**. Position = bottom-left, 0px left margin (tangent to video left edge), 60px bottom margin. No gradient ring, no colored border — just the circular mask with a single subtle shadow (offset 0,4 blur 10 α90).

### Required HTML Template

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
<title>Presentation</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Noto+Sans+SC:wght@300;400;500;700;900&family=JetBrains+Mono:wght@400;700&family=Space+Grotesk:wght@400;500;700&display=swap" rel="stylesheet">
<style>
  :root {
    --bg-primary: #050510;
    --bg-card: #0d0d1a;
    --text-primary: #f0f0f5;
    --text-secondary: #8888a0;
    --accent: #00d4ff;
    --accent-secondary: #7c3aed;
    --gradient-accent: linear-gradient(135deg, #00d4ff 0%, #7c3aed 100%);
  }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: 'Noto Sans SC', -apple-system, sans-serif;
    background: var(--bg-primary);
    color: var(--text-primary);
    overflow-x: hidden;
    font-size: 24px;
  }
  .bg-grid {
    position: fixed; top: 0; left: 0; width: 100%; height: 100%;
    background-image:
      linear-gradient(rgba(0, 212, 255, 0.03) 1px, transparent 1px),
      linear-gradient(90deg, rgba(0, 212, 255, 0.03) 1px, transparent 1px);
    background-size: 60px 60px;
    pointer-events: none; z-index: 0;
  }
  .container {
    position: relative; z-index: 1;
    display: flex; overflow-x: auto;
    scroll-snap-type: x mandatory;
    height: 100vh; scrollbar-width: none;
  }
  .container::-webkit-scrollbar { display: none; }
  .slide {
    flex: 0 0 100vw; height: 100vh;
    scroll-snap-align: start;
    display: flex; flex-direction: column;
    justify-content: center; align-items: center;
    padding: 60px 40px;
    text-align: center;
  }
  .slide-badge {
    position: absolute; top: 30px; left: 30px;
    display: flex; align-items: center; gap: 10px;
  }
  .slide-badge-num {
    font-family: 'JetBrains Mono', monospace;
    font-size: 18px; font-weight: 700;
    color: var(--accent);
    background: rgba(0, 212, 255, 0.1);
    padding: 10px 20px; border-radius: 20px;
    border: 1px solid rgba(0, 212, 255, 0.3);
  }
  .slide-content { max-width: 900px; width: 100%; text-align: center; }
  .slide-title {
    font-family: 'Space Grotesk', 'Noto Sans SC', sans-serif;
    font-size: clamp(52px, 11vw, 96px);
    font-weight: 900; margin-bottom: 28px;
    line-height: 1.15; letter-spacing: -1px;
    max-width: 800px;
  }
  .slide-title .gradient-text {
    background: var(--gradient-accent);
    -webkit-background-clip: text; -webkit-text-fill-color: transparent;
    background-clip: text;
  }
  .slide-title .highlight { color: var(--accent); -webkit-text-fill-color: var(--accent); }
  .slide-subtitle {
    font-size: clamp(28px, 4.5vw, 38px);
    color: var(--text-secondary); font-weight: 400;
    margin-bottom: 40px; line-height: 1.5;
  }
  .info-cards {
    display: grid; grid-template-columns: repeat(3, 1fr);
    gap: 24px; margin: 40px 0;
  }
  .info-card {
    background: var(--bg-card);
    border: 1px solid rgba(255,255,255,0.06);
    border-radius: 20px; padding: 36px 24px; text-align: center;
  }
  .info-card .emoji { font-size: 48px; margin-bottom: 12px; }
  .info-card .label { font-size: 30px; color: var(--text-secondary); line-height: 1.4; }
  .info-card .value {
    font-family: 'Space Grotesk', sans-serif;
    font-size: 78px; font-weight: 900;
    background: var(--gradient-accent);
    -webkit-background-clip: text; -webkit-text-fill-color: transparent;
    background-clip: text;
    margin-bottom: 4px;
  }
  .tags { display: flex; flex-wrap: wrap; gap: 14px; justify-content: center; margin-top: 35px; }
  .tag {
    padding: 16px 30px; background: var(--bg-card);
    border: 1px solid rgba(255,255,255,0.08); border-radius: 30px;
    font-size: 28px; color: var(--text-secondary);
  }
  .tag.highlight {
    background: rgba(0, 212, 255, 0.15);
    border-color: var(--accent); color: var(--accent);
  }
  .progress {
    position: fixed; bottom: 30px; left: 50%; transform: translateX(-50%);
    display: flex; gap: 10px; z-index: 100;
    background: rgba(13, 13, 26, 0.95);
    padding: 12px 22px; border-radius: 30px;
    border: 1px solid rgba(255,255,255,0.1);
  }
  .progress-dot {
    width: 10px; height: 10px; border-radius: 50%;
    background: rgba(255,255,255,0.25); cursor: pointer;
  }
  .progress-dot.active { background: var(--accent); transform: scale(1.4); }
  @media (max-width: 900px) {
    body { font-size: 22px; }
    .slide { padding: 50px 25px; }
    .slide-title { font-size: clamp(46px, 13vw, 72px); }
    .slide-subtitle { font-size: clamp(26px, 5.5vw, 32px); }
    .info-cards { grid-template-columns: repeat(2, 1fr); gap: 14px; }
    .info-card { padding: 28px 16px; }
    .info-card .value { font-size: 58px; }
    .info-card .label { font-size: 24px; }
    .tag { padding: 14px 24px; font-size: 22px; }
    .progress { bottom: 20px; }
  }
</style>
</head>
<body>
<div class="bg-grid"></div>
<div class="container" id="container">
    <!-- Slides go here: data-index="0", data-index="1", etc. -->
</div>
<div class="progress" id="progress"></div>
<script>
    const container = document.getElementById('container');
    const dots = document.querySelectorAll('.progress-dot');
    function goToSlide(index) {
        const slideWidth = window.innerWidth;
        container.scrollTo({ left: slideWidth * index, behavior: 'smooth' });
        dots.forEach((dot, i) => dot.classList.toggle('active', i === index));
    }
    container.addEventListener('scroll', () => {
        const index = Math.round(container.scrollLeft / window.innerWidth);
        dots.forEach((dot, i) => dot.classList.toggle('active', i === index));
    });
</script>
</body>
</html>
```

### Slide Content Rules

**Each slide has exactly one job.** The audience hears the narration; the slide gives them a visual anchor. If a slide's text reads like a paragraph, it's wrong.

| Slide role | What goes on screen | What does NOT |
|-----------|-------------------|---------------|
| Hook / opening | One bold headline (≤15 chars) | Full hook paragraph |
| Claim / point | One short title + 1-2 numbers or 1 emoji icon | Full script paragraph |
| Contrast | Left word vs Right word (≤6 chars each) | Full comparison prose |
| Process | 3-4 node labels (≤8 chars each) | Full explanation |
| Action / closing | One call-to-action headline | Full closing monologue |

**Test**: screenshot any slide, show it to someone for 2 seconds, then ask them what it was about. If they can't answer, the slide has too much text.

### Font Size Rules (Mobile-Readable Baseline)

Rendered at 1920×1080. On a 6.7-inch phone this is ~720px. **20px source ≈ 7.5px on screen — invisible.**

| Element | Size |
|---------|------|
| `.slide-title` | 80–96px |
| **`.slide-subtitle` (baseline)** | **≥ 38px** |
| Card labels, tags | 28–32px |
| Card values / numbers | 70–90px |
| Emoji / icons | 48px (cards), 130px (hero) |
| Badge / page number | 18px |

**Never** use 18–22px for body or descriptive text.

## Step 4: Render

The `config.json` must be placed in the skill directory (`luma-ppt-video/`). The generator reads it from `Path(__file__).parent.parent`.

To use this skill from a project:
1. Write `config.json` and `index.html` into the skill directory
2. Ensure `subtitle_segments.json` is at the path referenced by config
3. Run:

```bash
cd <skill_dir>
python scripts/generate_video.py
```

The generator opens the HTML in headless Chromium, scrolls through slides per timestamps, captures at 30fps, and merges into MP4 via ffmpeg.

## Step 5: Overlay Digital Human (Optional)

The avatar is a **small presence cue**, not a half-screen co-host. Keep it unobtrusive.

| Parameter | Value | Notes |
|-----------|-------|-------|
| Circle diameter | **340px** | Visible but doesn't compete with slide content |
| Position | bottom-left, 0px left margin (tangent to edge), 60px bottom | |
| Border / ring | **None** | No colored ring — just the circular mask |
| Shadow | **Single subtle** (offset 0,4 blur 10 α90) | Enough depth to separate from background |
| bbox | (0, 350, 1080, 1500) | Full source width, head+chest |

```bash
# Inspect the avatar frame
python <skill_dir>/scripts/overlay_avatar.py --thumb-only

# Overlay
python <skill_dir>/scripts/overlay_avatar.py \
  --slide 13_ppt.mp4 \
  --avatar 10_digital_human.mp4 \
  --output 14_ppt_with_avatar.mp4 \
  --ffmpeg "$(luma-cli runtime path ffmpeg)"
```

The overlay_avatar.py script supports `--size` and `--no-ring` flags; use them to match the design rules above.

## Configuration Reference

| Setting | Description | Default |
|---------|-------------|---------|
| `subtitle_file` | Path to subtitle JSON | `./subtitle_segments.json` |
| `html_file` | Path to HTML presentation | `./index.html` |
| `output_file` | Output video path | `./output.mp4` |
| `ffmpeg_path` | Path to ffmpeg | from `luma-cli runtime path ffmpeg` |
| `video_size` | [width, height] | `[1920, 1080]` |
| `extra_time` | Extra seconds at end | `3.0` |
| `sentence_to_slide` | Sentence ID → slide index | `{}` |

## Dependencies

- Python 3.10+
- Playwright:

  ```bash
  pip install playwright pillow
  python -m playwright install chromium
  ```

- ffmpeg from `luma-cli runtime install ffmpeg` or system PATH.
