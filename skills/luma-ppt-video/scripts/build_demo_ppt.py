#!/usr/bin/env python3
"""Build a simple offline PPT-style HTML deck from align.json."""

from __future__ import annotations

import argparse
import html
import json
from pathlib import Path


def load_units(path: Path) -> list[dict]:
    data = json.loads(path.read_text(encoding="utf-8-sig"))
    if isinstance(data, dict):
        for key in ("sentence_units", "sentences", "units"):
            value = data.get(key)
            if isinstance(value, list) and value:
                return normalize_units(value)
        segments = data.get("segments")
        if isinstance(segments, list) and segments:
            return units_from_segments(segments)
    if isinstance(data, list):
        return normalize_units(data)
    raise ValueError("align file must contain sentence_units, sentences, units, or segments")


def normalize_units(items: list[dict]) -> list[dict]:
    out = []
    for index, item in enumerate(items):
        text = str(item.get("text") or item.get("sentence") or "").strip()
        if not text:
            continue
        start = float(item.get("start", item.get("start_time", 0)) or 0)
        end = float(item.get("end", item.get("end_time", start + 1)) or start + 1)
        sent_id = str(item.get("sent_id") or item.get("id") or f"sent_{index:04d}")
        out.append({"sent_id": sent_id, "start": start, "end": max(end, start + 0.1), "text": text})
    if not out:
        raise ValueError("no timed text units found")
    return out


def units_from_segments(segments: list[dict]) -> list[dict]:
    grouped: dict[str, dict] = {}
    order: list[str] = []
    for index, seg in enumerate(segments):
        text = str(seg.get("text") or "").strip()
        if not text:
            continue
        sent_id = str(seg.get("sent_id") or seg.get("sentence_id") or f"sent_{index:04d}")
        if sent_id not in grouped:
            grouped[sent_id] = {"sent_id": sent_id, "start": float(seg.get("start", 0) or 0), "end": 0.0, "parts": []}
            order.append(sent_id)
        grouped[sent_id]["parts"].append(text)
        grouped[sent_id]["end"] = max(grouped[sent_id]["end"], float(seg.get("end", grouped[sent_id]["start"] + 1) or 0))
    return normalize_units([
        {"sent_id": sent_id, "start": grouped[sent_id]["start"], "end": grouped[sent_id]["end"], "text": "".join(grouped[sent_id]["parts"])}
        for sent_id in order
    ])


def group_slides(units: list[dict], max_units_per_slide: int) -> tuple[list[list[dict]], dict[str, int]]:
    slides: list[list[dict]] = []
    mapping: dict[str, int] = {}
    current: list[dict] = []
    for unit in units:
        current.append(unit)
        if len(current) >= max_units_per_slide or unit["text"].endswith(("。", "！", "？", ".", "!", "?")):
            slides.append(current)
            current = []
    if current:
        slides.append(current)
    for slide_index, slide in enumerate(slides):
        for unit in slide:
            mapping[unit["sent_id"]] = slide_index
    return slides, mapping


def slide_title(text: str, fallback: str) -> str:
    clean = " ".join(text.replace("\n", " ").split())
    for sep in ("，", "。", "；", ",", ".", ";"):
        if sep in clean:
            clean = clean.split(sep, 1)[0]
            break
    return clean[:28] or fallback


def render_html(slides: list[list[dict]], title: str) -> str:
    slide_html = []
    for index, slide in enumerate(slides):
        text = " ".join(unit["text"] for unit in slide)
        title_text = slide_title(text, f"Point {index + 1}")
        body_items = "\n".join(f"<li>{html.escape(unit['text'])}</li>" for unit in slide)
        slide_html.append(f"""
        <section class="slide" data-index="{index}">
          <div class="badge">#{index + 1:02d}</div>
          <div class="content">
            <h1>{html.escape(title_text)}</h1>
            <ul>{body_items}</ul>
          </div>
        </section>""")
    return f"""<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{html.escape(title)}</title>
  <style>
    :root {{
      --bg: #071013;
      --panel: #102026;
      --text: #f5f7f2;
      --muted: #b7c5bd;
      --accent: #20c997;
      --accent-2: #ffcc66;
    }}
    * {{ box-sizing: border-box; }}
    html, body {{ margin: 0; width: 100%; height: 100%; overflow: hidden; background: var(--bg); color: var(--text); }}
    body {{ font-family: "Microsoft YaHei", "Noto Sans CJK SC", Arial, sans-serif; }}
    .deck {{ width: 100vw; height: 100vh; position: relative; }}
    .slide {{ position: absolute; inset: 0; display: none; padding: 72px 92px; }}
    .slide.active {{ display: grid; align-items: center; }}
    .slide::before {{
      content: ""; position: absolute; inset: 0;
      background:
        linear-gradient(90deg, rgba(32,201,151,.13), transparent 32%),
        radial-gradient(circle at 82% 18%, rgba(255,204,102,.16), transparent 30%);
      pointer-events: none;
    }}
    .badge {{
      position: absolute; top: 42px; left: 52px; color: var(--accent);
      font-size: 28px; font-weight: 800; letter-spacing: .08em;
    }}
    .content {{ position: relative; max-width: 1260px; }}
    h1 {{ margin: 0 0 42px; font-size: 82px; line-height: 1.08; letter-spacing: 0; }}
    ul {{ list-style: none; margin: 0; padding: 0; display: grid; gap: 24px; }}
    li {{
      font-size: 42px; line-height: 1.45; color: var(--muted);
      background: rgba(16,32,38,.82); border-left: 8px solid var(--accent);
      border-radius: 8px; padding: 24px 30px;
    }}
    .progress {{ position: absolute; left: 52px; right: 52px; bottom: 38px; height: 6px; background: rgba(255,255,255,.16); }}
    .progress span {{ display: block; height: 100%; width: 0; background: var(--accent-2); }}
  </style>
</head>
<body>
  <main class="deck" id="deck">
    {''.join(slide_html)}
    <div class="progress"><span id="progress"></span></div>
  </main>
  <script>
    const slides = [...document.querySelectorAll('.slide')];
    const progress = document.getElementById('progress');
    window.showSlide = function(index) {{
      const safe = Math.max(0, Math.min(slides.length - 1, Number(index) || 0));
      slides.forEach((slide, i) => slide.classList.toggle('active', i === safe));
      progress.style.width = ((safe + 1) / Math.max(slides.length, 1) * 100) + '%';
    }};
    window.showSlide(0);
  </script>
</body>
</html>
"""


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--align", required=True, help="Path to align.json")
    parser.add_argument("--output-dir", default="ppt_video_demo")
    parser.add_argument("--title", default="PPT Video Demo")
    parser.add_argument("--max-units-per-slide", type=int, default=2)
    parser.add_argument("--audio", default="")
    parser.add_argument("--output-video", default="ppt.mp4")
    args = parser.parse_args()

    align_path = Path(args.align).resolve()
    out_dir = Path(args.output_dir).resolve()
    out_dir.mkdir(parents=True, exist_ok=True)

    units = load_units(align_path)
    slides, mapping = group_slides(units, max(1, args.max_units_per_slide))
    html_path = out_dir / "index.html"
    config_path = out_dir / "config.json"
    output_video = out_dir / args.output_video

    html_path.write_text(render_html(slides, args.title), encoding="utf-8")
    config = {
        "align_file": str(align_path),
        "html_file": str(html_path),
        "output_file": str(output_video),
        "video_size": [1920, 1080],
        "extra_time": 1.5,
        "sentence_to_slide": mapping,
    }
    if args.audio:
        config["audio_file"] = str(Path(args.audio).resolve())
    config_path.write_text(json.dumps(config, ensure_ascii=False, indent=2), encoding="utf-8")
    print(json.dumps({"html": str(html_path), "config": str(config_path), "slides": len(slides), "output": str(output_video)}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
