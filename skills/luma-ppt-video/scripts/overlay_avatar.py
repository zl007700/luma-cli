#!/usr/bin/env python3
"""Overlay a digital-human video as a bottom-left circular PiP."""

from __future__ import annotations

import argparse
import json
import shutil
import subprocess
import tempfile
from pathlib import Path

try:
    from PIL import Image, ImageDraw, ImageFilter
except ImportError as exc:
    raise SystemExit("Missing Pillow. Run: pip install pillow") from exc


def resolve_ffmpeg(value: str) -> str:
    if value and Path(value).exists():
        return value
    found = shutil.which("ffmpeg")
    if found:
        return found
    raise SystemExit("ffmpeg not found. Run: luma-cli runtime install ffmpeg, then pass --ffmpeg <path>.")


def make_circle_assets(work_dir: Path, size: int, ring: int) -> dict[str, Path]:
    mask = Image.new("L", (size, size), 0)
    draw = ImageDraw.Draw(mask)
    draw.ellipse((0, 0, size - 1, size - 1), fill=255)
    mask_path = work_dir / "avatar_mask.png"
    mask.save(mask_path)

    def shadow(path: Path, offset: tuple[int, int], blur: int, alpha: int) -> None:
        img = Image.new("RGBA", (size + offset[0] + blur * 2, size + offset[1] + blur * 2), (0, 0, 0, 0))
        d = ImageDraw.Draw(img)
        d.ellipse((blur + offset[0], blur + offset[1], blur + offset[0] + size - 1, blur + offset[1] + size - 1), fill=(0, 0, 0, alpha))
        img.filter(ImageFilter.GaussianBlur(blur)).save(path)

    shadow1 = work_dir / "avatar_shadow1.png"
    shadow2 = work_dir / "avatar_shadow2.png"
    shadow(shadow1, (0, 4), 10, 90)
    shadow(shadow2, (8, 18), 28, 140)

    ring_size = size + ring * 2
    gradient = Image.new("RGBA", (ring_size, ring_size), (0, 0, 0, 0))
    for y in range(ring_size):
        for x in range(ring_size):
            t = (x + y) / max(1, 2 * (ring_size - 1))
            r = int(32 * (1 - t) + 255 * t)
            g = int(201 * (1 - t) + 204 * t)
            b = int(151 * (1 - t) + 102 * t)
            gradient.putpixel((x, y), (r, g, b, 255))
    ring_mask = Image.new("L", (ring_size, ring_size), 0)
    d = ImageDraw.Draw(ring_mask)
    d.ellipse((0, 0, ring_size - 1, ring_size - 1), fill=255)
    d.ellipse((ring, ring, ring_size - 1 - ring, ring_size - 1 - ring), fill=0)
    gradient.putalpha(ring_mask)
    ring_path = work_dir / "avatar_ring.png"
    gradient.save(ring_path)

    return {"mask": mask_path, "shadow1": shadow1, "shadow2": shadow2, "ring": ring_path}


def extract_thumb(ffmpeg: str, avatar: Path, output: Path) -> None:
    cmd = [ffmpeg, "-y", "-ss", "1.0", "-i", str(avatar), "-vframes", "1", "-q:v", "2", str(output)]
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        raise SystemExit("thumb extraction failed:\n" + result.stderr[-1200:])


def overlay(args: argparse.Namespace) -> None:
    ffmpeg = resolve_ffmpeg(args.ffmpeg)
    slide = Path(args.slide).resolve()
    avatar = Path(args.avatar).resolve()
    output = Path(args.output).resolve()
    output.parent.mkdir(parents=True, exist_ok=True)
    if not slide.exists():
        raise SystemExit(f"slide video not found: {slide}")
    if not avatar.exists():
        raise SystemExit(f"avatar video not found: {avatar}")

    x, y, w, h = args.bbox
    size = int(args.size)
    margin_x = int(args.margin_x)
    margin_bottom = int(args.margin_bottom)
    ring = int(args.ring)
    temp_dir = Path(tempfile.mkdtemp(prefix="luma-avatar-overlay-"))
    try:
        assets = make_circle_assets(temp_dir, size, ring)
        shadow1_w = size + 0 + 10 * 2
        shadow1_h = size + 4 + 10 * 2
        shadow2_w = size + 8 + 28 * 2
        shadow2_h = size + 18 + 28 * 2
        base_y = f"H-{margin_bottom}-{size}"
        shadow1_x = margin_x - 10
        shadow1_y = f"H-{margin_bottom}-{size}-10+4"
        shadow2_x = margin_x - 28 + 8
        shadow2_y = f"H-{margin_bottom}-{size}-28+18"
        ring_x = margin_x - ring
        ring_y = f"H-{margin_bottom}-{size}-{ring}"

        filter_complex = (
            f"[1:v]crop={w}:{h}:{x}:{y},scale={size}:{size}:force_original_aspect_ratio=increase,"
            f"crop={size}:{size},format=rgba[avatar];"
            f"[avatar][5:v]alphamerge[avatar_masked];"
            f"[0:v][2:v]overlay={shadow2_x}:{shadow2_y}[v1];"
            f"[v1][3:v]overlay={shadow1_x}:{shadow1_y}[v2];"
            f"[v2][4:v]overlay={ring_x}:{ring_y}[v3];"
            f"[v3][avatar_masked]overlay={margin_x}:{base_y}[vout]"
        )
        cmd = [
            ffmpeg,
            "-y",
            "-i",
            str(slide),
            "-i",
            str(avatar),
            "-i",
            str(assets["shadow2"]),
            "-i",
            str(assets["shadow1"]),
            "-i",
            str(assets["ring"]),
            "-i",
            str(assets["mask"]),
            "-filter_complex",
            filter_complex,
            "-map",
            "[vout]",
            "-map",
            "1:a?",
            "-c:v",
            "libx264",
            "-preset",
            "veryfast",
            "-crf",
            "20",
            "-c:a",
            "aac",
            "-shortest",
            "-movflags",
            "+faststart",
            str(output),
        ]
        result = subprocess.run(cmd, capture_output=True, text=True)
        if result.returncode != 0:
            raise SystemExit("ffmpeg overlay failed:\n" + result.stderr[-3000:])
        print(json.dumps({"output": str(output), "size": size, "bbox": [x, y, w, h]}, indent=2))
    finally:
        if not args.keep_assets:
            shutil.rmtree(temp_dir, ignore_errors=True)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--slide", default="ppt.mp4")
    parser.add_argument("--avatar", default="digital_human.mp4")
    parser.add_argument("--output", default="ppt_with_avatar.mp4")
    parser.add_argument("--ffmpeg", default="")
    parser.add_argument("--bbox", nargs=4, type=int, default=[0, 350, 1080, 1500], metavar=("X", "Y", "W", "H"))
    parser.add_argument("--size", type=int, default=380)
    parser.add_argument("--margin-x", type=int, default=60)
    parser.add_argument("--margin-bottom", type=int, default=60)
    parser.add_argument("--ring", type=int, default=5)
    parser.add_argument("--thumb-only", action="store_true")
    parser.add_argument("--thumb-output", default="avatar_thumb.jpg")
    parser.add_argument("--keep-assets", action="store_true")
    args = parser.parse_args()

    if args.thumb_only:
        ffmpeg = resolve_ffmpeg(args.ffmpeg)
        extract_thumb(ffmpeg, Path(args.avatar).resolve(), Path(args.thumb_output).resolve())
        print(json.dumps({"thumb": str(Path(args.thumb_output).resolve())}, indent=2))
        return
    overlay(args)


if __name__ == "__main__":
    main()
