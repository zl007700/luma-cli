#!/usr/bin/env python3
"""Render a PPT-style HTML deck into MP4 using align timing."""

from __future__ import annotations

import argparse
import asyncio
import json
import os
import shutil
import subprocess
import tempfile
from pathlib import Path

from build_demo_ppt import load_units

try:
    from playwright.async_api import async_playwright
except ImportError as exc:
    raise SystemExit("Missing playwright. Run: pip install playwright && python -m playwright install chromium") from exc


def load_config(path: Path) -> dict:
    config = json.loads(path.read_text(encoding="utf-8-sig"))
    base = path.parent
    for key in ("align_file", "subtitle_file", "html_file", "output_file", "audio_file"):
        if config.get(key):
            value = Path(str(config[key]))
            if not value.is_absolute():
                config[key] = str((base / value).resolve())
    return config


def build_schedule(units: list[dict], config: dict) -> list[dict]:
    mapping = config.get("sentence_to_slide") or {}
    schedule = []
    prev_slide = None
    slide_start = units[0]["start"] if units else 0.0
    for unit in units:
        slide = int(mapping.get(unit["sent_id"], 0) or 0)
        if prev_slide is None:
            prev_slide = slide
            slide_start = unit["start"]
            continue
        if slide != prev_slide:
            schedule.append({"slide": prev_slide, "start": slide_start, "duration": max(0.1, unit["start"] - slide_start)})
            prev_slide = slide
            slide_start = unit["start"]
    if prev_slide is not None:
        extra = float(config.get("extra_time", 1.5) or 0)
        schedule.append({"slide": prev_slide, "start": slide_start, "duration": max(0.1, units[-1]["end"] - slide_start + extra)})
    return schedule


def resolve_ffmpeg(value: str) -> str:
    if value and Path(value).exists():
        return value
    found = shutil.which("ffmpeg")
    if found:
        return found
    raise SystemExit("ffmpeg not found. Run: luma-cli runtime install ffmpeg, then pass --ffmpeg <path>.")


async def capture_slides(config: dict, schedule: list[dict], temp_dir: Path) -> list[Path]:
    html_file = Path(config["html_file"]).resolve()
    width, height = config.get("video_size", [1920, 1080])
    shots = []
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=True)
        context = await browser.new_context(viewport={"width": int(width), "height": int(height)}, device_scale_factor=1)
        page = await context.new_page()
        await page.goto(html_file.as_uri())
        await page.wait_for_load_state("networkidle")
        await page.wait_for_timeout(300)
        for index, item in enumerate(schedule):
            await page.evaluate("(slide) => window.showSlide && window.showSlide(slide)", int(item["slide"]))
            await page.wait_for_timeout(120)
            shot = temp_dir / f"slide_{index:04d}.jpg"
            await page.screenshot(path=str(shot), type="jpeg", quality=94)
            shots.append(shot)
        await browser.close()
    return shots


def write_concat_file(shots: list[Path], schedule: list[dict], temp_dir: Path) -> Path:
    concat = temp_dir / "slides.ffconcat"
    lines = ["ffconcat version 1.0"]
    for shot, item in zip(shots, schedule):
        lines.append(f"file '{shot.as_posix()}'")
        lines.append(f"duration {float(item['duration']):.3f}")
    if shots:
        lines.append(f"file '{shots[-1].as_posix()}'")
    concat.write_text("\n".join(lines) + "\n", encoding="utf-8")
    return concat


def run_ffmpeg(ffmpeg: str, concat: Path, config: dict) -> None:
    output = Path(config["output_file"]).resolve()
    output.parent.mkdir(parents=True, exist_ok=True)
    width, height = config.get("video_size", [1920, 1080])
    cmd = [
        ffmpeg,
        "-y",
        "-f",
        "concat",
        "-safe",
        "0",
        "-i",
        str(concat),
    ]
    audio = str(config.get("audio_file") or "").strip()
    if audio:
        cmd += ["-i", audio]
    cmd += [
        "-vf",
        f"fps=30,scale={int(width)}:{int(height)}:force_original_aspect_ratio=decrease,pad={int(width)}:{int(height)}:(ow-iw)/2:(oh-ih)/2,format=yuv420p",
        "-c:v",
        "libx264",
        "-preset",
        "veryfast",
        "-crf",
        "20",
        "-movflags",
        "+faststart",
    ]
    if audio:
        cmd += ["-map", "0:v:0", "-map", "1:a:0", "-c:a", "aac", "-shortest"]
    cmd.append(str(output))
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        raise SystemExit("ffmpeg failed:\n" + result.stderr[-3000:])


async def main_async() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", required=True)
    parser.add_argument("--ffmpeg", default=os.environ.get("FFMPEG_PATH", ""))
    parser.add_argument("--output", default="")
    parser.add_argument("--audio", default="")
    parser.add_argument("--keep-frames", action="store_true")
    args = parser.parse_args()

    config = load_config(Path(args.config).resolve())
    if args.output:
        config["output_file"] = str(Path(args.output).resolve())
    if args.audio:
        config["audio_file"] = str(Path(args.audio).resolve())
    ffmpeg = resolve_ffmpeg(args.ffmpeg or str(config.get("ffmpeg_path") or ""))
    align_file = config.get("align_file") or config.get("subtitle_file")
    if not align_file:
        raise SystemExit("config requires align_file or subtitle_file")
    units = load_units(Path(align_file))
    schedule = build_schedule(units, config)
    if not schedule:
        raise SystemExit("empty slide schedule")

    temp_root = Path(tempfile.mkdtemp(prefix="luma-ppt-video-"))
    try:
        shots = await capture_slides(config, schedule, temp_root)
        concat = write_concat_file(shots, schedule, temp_root)
        run_ffmpeg(ffmpeg, concat, config)
        print(json.dumps({"output": str(Path(config["output_file"]).resolve()), "slides": len(shots), "duration_sec": round(sum(float(x["duration"]) for x in schedule), 3)}, indent=2))
    finally:
        if not args.keep_frames:
            shutil.rmtree(temp_root, ignore_errors=True)


def main() -> None:
    asyncio.run(main_async())


if __name__ == "__main__":
    main()
