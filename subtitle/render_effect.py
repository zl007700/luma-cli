#!/usr/bin/env python3
"""
Subtitle effect renderer - mirrors Z agent's FontEffect system.
Go calls this to render text frames with effects, then encode to MOV.
"""

import sys
import json
import math
import tempfile
import os
import subprocess
from PIL import Image, ImageDraw, ImageFont

ENTRANCE_EFFECTS = {
    "blur_in", "fade_in", "slide_left", "slide_right", "slide_down", "slide_up",
    "scale_pop", "rotate_pop", "bounce_in", "letter_drop", "typewriter",
    "wipe", "wave_bounce", "glitch_in", "neon_flicker", "metal_shine",
    "fire_flicker", "trail_echo", "mask_reveal", "ink_spread", "pixel_build",
    "wave_scan", "stroke_draw",
}
MAX_ANIMATION_SECONDS = 1.0


def ease_back_out(t):
    if t >= 1.0:
        return 1.0
    c1 = 1.70158
    c3 = c1 + 1
    return 1 + c3 * pow(t - 1, 3) + c1 * pow(t - 1, 2)


def ease_cubic_out(t):
    if t >= 1.0:
        return 1.0
    return 1 - pow(1 - t, 3)


def ease_bounce(t):
    if t < 0.25:
        return t * 4
    if t < 0.5:
        return 1 - pow(t - 0.25, 2) * 4
    if t < 0.75:
        return 0.75 + (t - 0.5) * 2
    return 1 - pow(t - 0.75, 2) * 4


def measure_text(text, font, stroke_width=0):
    """Measure text bbox - pass stroke_width for accurate measurement."""
    if hasattr(font, 'getbbox'):
        left, top, right, bottom = font.getbbox(text, stroke_width=stroke_width)
    else:
        left, top, right, bottom = font.getbbox(text)
    return {
        'left': left, 'top': top,
        'width': max(1, right - left),
        'height': max(1, bottom - top)
    }


def to_tuple(v):
    if isinstance(v, list):
        return tuple(v)
    return v


def _render_line_with_highlight(line, font, font_color, stroke_color, stroke_width,
                                 highlight_word, highlight_color, highlight_scale):
    """Render one line with optional highlight word - mirrors Z agent's _render_line_with_highlight."""
    keyword = str(highlight_word or "").strip()
    if not keyword or keyword not in line:
        left, top, right, bottom = font.getbbox(line, stroke_width=stroke_width)
        image = Image.new("RGBA", (max(1, right - left), max(1, bottom - top)), (0, 0, 0, 0))
        draw = ImageDraw.Draw(image)
        draw.text((-left, -top), line, font=font, fill=font_color,
                  stroke_width=stroke_width, stroke_fill=stroke_color)
        return image

    # Has highlight - split into prefix/keyword/suffix
    prefix, _, suffix = line.partition(keyword)
    h_font_size = max(font.size + 2, int(round(font.size * highlight_scale)))
    try:
        highlight_font = font.font_variant(size=h_font_size)
    except:
        highlight_font = font

    parts = []
    for text_part, text_font, fill in (
        (prefix, font, font_color),
        (keyword, highlight_font, highlight_color),
        (suffix, font, font_color),
    ):
        if text_part == "":
            continue
        left, top, right, bottom = text_font.getbbox(text_part, stroke_width=stroke_width)
        part = Image.new("RGBA", (max(1, right - left), max(1, bottom - top)), (0, 0, 0, 0))
        draw = ImageDraw.Draw(part)
        draw.text((-left, -top), text_part, font=text_font, fill=fill,
                  stroke_width=stroke_width, stroke_fill=stroke_color)
        parts.append(part)

    if not parts:
        return Image.new("RGBA", (1, 1), (0, 0, 0, 0))

    width = sum(part.width for part in parts)
    height = max(part.height for part in parts) if parts else 1
    image = Image.new("RGBA", (max(1, width), max(1, height)), (0, 0, 0, 0))
    cursor_x = 0
    for part in parts:
        cursor_y = (height - part.height) // 2
        image.alpha_composite(part, (cursor_x, cursor_y))
        cursor_x += part.width
    return image


def render_sprite_with_highlight(text, font, font_color, stroke_color, stroke_width,
                                  highlight_word, highlight_color, highlight_scale):
    """Build text sprite with highlight support - mirrors Z agent's sprite_builder."""
    lines = text.splitlines() if text else [str(text or "").strip() or " "]
    if not lines:
        lines = [text]

    padding = max(6, int(font.size * 0.18))
    line_gap = max(6, int(font.size * 0.18))

    line_images = []
    max_width = 1
    total_height = 0
    for line in lines:
        line_img = _render_line_with_highlight(
            line=line, font=font, font_color=font_color,
            stroke_color=stroke_color, stroke_width=stroke_width,
            highlight_word=highlight_word, highlight_color=highlight_color,
            highlight_scale=highlight_scale,
        )
        line_images.append(line_img)
        max_width = max(max_width, line_img.width)
        total_height += line_img.height
    total_height += line_gap * max(0, len(line_images) - 1)

    image = Image.new("RGBA", (max_width + padding * 2, total_height + padding * 2), (0, 0, 0, 0))
    cursor_y = padding
    for line_img in line_images:
        cursor_x = padding + (max_width - line_img.width) // 2
        image.alpha_composite(line_img, (cursor_x, cursor_y))
        cursor_y += line_img.height + line_gap

    return image


def generate_animation_frames(text, effect_type, anim_duration, fps, cfg, sprite):
    """Generate animation frames for entrance effect - 1 second max."""
    width = cfg.get('width', 1080)
    height = cfg.get('height', 1920)
    anchor_y = cfg.get('anchor_y', height - 167)

    total_anim_frames = max(1, int(anim_duration * fps))
    frames = []

    for i in range(total_anim_frames):
        progress = float(i) / float(max(1, total_anim_frames - 1))
        frame = Image.new('RGBA', (width, height), (0, 0, 0, 0))

        if effect_type == 'bounce_in':
            bounce = ease_bounce(min(progress * 1.3, 1.0))
            offset_y = int(100 * (1 - bounce))
            cx = width // 2 - sprite.width // 2
            cy = anchor_y - sprite.height + offset_y
            frame.paste(sprite, (cx, cy), sprite)

        elif effect_type == 'scale_pop':
            scale = 0.1 + 0.9 * ease_back_out(progress)
            w = max(1, int(sprite.width * scale))
            h = max(1, int(sprite.height * scale))
            resized = sprite.resize((w, h), Image.LANCZOS)
            cx = width // 2 - w // 2
            cy = height // 2 - h // 2
            frame.paste(resized, (cx, cy), resized)

        elif effect_type == 'wave_bounce':
            scale_x = 0.9 + 0.15 * math.sin(progress * math.pi)
            scale_y = 1.0 + 0.1 * progress * ease_cubic_out(progress)
            w = max(1, int(sprite.width * scale_x))
            h = max(1, int(sprite.height * scale_y))
            resized = sprite.resize((w, h), Image.LANCZOS)
            cx = width // 2 - w // 2
            cy = height // 2 - h // 2
            frame.paste(resized, (cx, cy), resized)

        elif effect_type == 'rotate_pop':
            angle = -20.0 + 25.0 * ease_back_out(progress)
            rotated = sprite.rotate(angle, expand=True, resample=Image.BICUBIC)
            cx = width // 2 - rotated.width // 2
            cy = height // 2 - rotated.height // 2
            frame.paste(rotated, (cx, cy), rotated)

        elif effect_type == 'blur_in':
            radius = (1.0 - progress) * 10.0
            alpha = int(255 * (0.3 + 0.7 * progress))
            if radius > 0.5:
                blurred = sprite.filter(Image.GaussianBlur(radius=radius))
            else:
                blurred = sprite
            if blurred.mode == 'RGBA':
                alpha_layer = blurred.split()[3].point(lambda v: v * alpha // 255)
                blurred.putalpha(alpha_layer)
            cx = width // 2 - blurred.width // 2
            cy = height // 2 - blurred.height // 2
            frame.paste(blurred, (cx, cy), blurred)

        else:
            cx = width // 2 - sprite.width // 2
            cy = anchor_y - sprite.height
            frame.paste(sprite, (cx, cy), sprite)

        frames.append(frame)

    return frames


def extend_hold_frames(frames, effect_type, total_duration, fps):
    """Extend animation frames with last-frame hold to fill full duration - mirrors Z agent's _extend_effect_hold_frames."""
    normalized = effect_type.strip().lower()
    if normalized not in ENTRANCE_EFFECTS or not frames:
        return frames

    target_count = max(1, int(total_duration * fps))
    if len(frames) >= target_count:
        return frames

    held_frame = frames[-1]
    frames.extend(held_frame.copy() for _ in range(target_count - len(frames)))
    return frames


def generate_frames(text, effect_type, duration, fps, cfg):
    """Generate frame images for a given effect."""
    font_size = cfg.get('font_size', 48)
    font_path = cfg.get('font_path', '')
    width = cfg.get('width', 1080)
    height = cfg.get('height', 1920)
    anchor_y = cfg.get('anchor_y', height - 167)
    text_color = to_tuple(cfg.get('text_color', (253, 253, 255, 255)))
    stroke_color = to_tuple(cfg.get('stroke_color', (31, 1, 1, 255)))
    stroke_width = int(cfg.get('stroke_width', 2.0))
    highlight_word = cfg.get('highlight_word', '')
    highlight_color = to_tuple(cfg.get('highlight_color', (90, 217, 255, 255)))
    highlight_scale = float(cfg.get('highlight_scale', 1.25))

    # Load font
    try:
        if font_path and os.path.exists(font_path):
            font = ImageFont.truetype(font_path, font_size)
        else:
            font = ImageFont.load_default()
    except:
        font = ImageFont.load_default()

    # Build sprite with highlight support
    sprite = render_sprite_with_highlight(
        text, font,
        font_color=text_color,
        stroke_color=stroke_color,
        stroke_width=stroke_width,
        highlight_word=highlight_word,
        highlight_color=highlight_color,
        highlight_scale=highlight_scale,
    )

    # Determine animation duration (entrance effects capped at 1 second)
    normalized = effect_type.strip().lower()
    if normalized in ENTRANCE_EFFECTS:
        anim_duration = min(duration, MAX_ANIMATION_SECONDS)
    else:
        anim_duration = duration

    # Generate animation frames
    frames = generate_animation_frames(text, effect_type, anim_duration, fps, cfg, sprite)

    # Extend with hold frames to fill full duration
    frames = extend_hold_frames(frames, effect_type, duration, fps)

    return frames


def encode_mov(frames, fps, output_path):
    """Encode frames to MOV using ffmpeg."""
    with tempfile.TemporaryDirectory(prefix='effect_frames_') as tmp_dir:
        for i, frame in enumerate(frames):
            frame.save(os.path.join(tmp_dir, f'frame_{i:04d}.png'), 'PNG')

        cmd = [
            'ffmpeg', '-y',
            '-framerate', str(fps),
            '-i', os.path.join(tmp_dir, 'frame_%04d.png'),
            '-c:v', 'qtrle',
            '-pix_fmt', 'argb',
            output_path
        ]
        subprocess.run(cmd, check=True, capture_output=True)


def main():
    if len(sys.argv) < 2:
        print('Usage: render_effect.py <json_config>', file=sys.stderr)
        sys.exit(1)

    cfg = json.loads(sys.argv[1])

    text = cfg.get('text', '')
    effect_type = cfg.get('effect_type', '')
    start = cfg.get('start', 0)
    end = cfg.get('end', 0)
    duration = end - start
    fps = cfg.get('fps', 30)

    if not text or not effect_type or effect_type == 'none' or duration <= 0:
        print('')
        sys.exit(0)

    frames = generate_frames(text, effect_type, duration, fps, cfg)

    output_path = os.path.join(tempfile.gettempdir(), f'effect_{os.getpid()}_{os.times().elapsed*1000000:.0f}.mov')
    encode_mov(frames, fps, output_path)

    print(output_path)


if __name__ == '__main__':
    main()