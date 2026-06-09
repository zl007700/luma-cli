"""
============================================================================
数字人叠加到幻灯片视频 —— **v4 推荐版 (圆脸 PiP)**
============================================================================

【推荐参数 (v4 固定版, 2026-06-04 定稿)】
----------------------------------------
圆形状     纯圆形, 直径 380px
位置       左下角, 距左 60px / 距底 60px
外环       5px 厚, 主题渐变 #ff3366 → #7c3aed (135°)
内高光     环内边缘 1px 白线, alpha 180 (金属质感)
投影       双层: 近层 (offset 0,4 blur 10 α90) + 远层 (offset 8,18 blur 28 α140)
bbox       (0, 350, 1080, 1500)  ← 整源宽, 0 headroom, 头+胸
音频       数字人视频自带音轨 (已 lip-sync 对齐)
图层层数   6 层 (slide + shadow2 + shadow1 + ring + avatar_masked)

【设计原则 (踩过的坑)】
----------------------------------------
✗ 不要 纯白环       → 跟 PPT 整体配色不搭, "low"
✗ 不要 加外发光 halo → 喧宾夺主, 比主标题还抢
✗ 不要 用 square 圆 → 太呆板, 偏 "webcam 截图"
✗ 不要 圆太小       → < 300 在 1920 屏里没存在感
✗ 不要 圆太大       → > 420 抢主标题的戏
✗ 不要 bbox 留大 headroom → 头顶空太多, 像 "漂浮的头"
✗ 不要 bbox 太高   → 脸会变形 (瘦高拉伸)
✓ 应该 主题色渐变环 → 跟 PPT 融为一体
✓ 应该 双层投影     → 才有"浮起"的层次感
✓ 应该 bbox 整源宽  → 脸占圆 50% 比例最平衡
✓ 应该 0 headroom   → 头顶贴圆顶, 紧凑不浪费
✓ 应该 1px 内高光   → 金属环的 "polish" 细节

【face 占比控制技巧】
----------------------------------------
方形 circle + bbox 比 = 1.0 → face 占圆 ~80% (局促)
矩形 bbox (整源宽)        → face 占圆 ~50% (推荐)
矩形 bbox (高瘦)         → face 被纵向压缩 → 用 scale+crop 保护
face 绝对 = 550 × (SIZE / bbox_width), 越小越"小脸"

【用法】
----------------------------------------
  # 1. 抽一帧给 VLM 看
  py overlay_avatar.py --thumb-only

  # 2. Agent 视觉读图, 给出 bbox (x y w h)
  #    标准推荐 (人脸在画面上半部, 数字人视频通用): 0 350 1080 1500
  #    调 headroom: 起点 y 从 350 调小 (e.g. 300 = 50px headroom)
  #    调宽度: 起点 x 和 w 决定脸在源里的水平位置
  py overlay_avatar.py 0 350 1080 1500

  # 3. 调试用: 只看 mask 长啥样
  py overlay_avatar.py --mask-only 0 350 1080 1500

【未来给新数字人视频用】
----------------------------------------
源视频是 1080×1920 竖屏的 lip-sync 数字人:
  - 抽帧 → VLM 看缩略图 → 给 bbox
  - bbox y 通常 200-400 (脸的纵向位置)
  - bbox w 通常 800-1080 (整源宽或稍窄)
  - bbox h 通常 800-1500 (头+胸, 不宜过高)
  - 调 RING_THICKNESS 改环厚度, 默认 5

【已知局限】
----------------------------------------
  - 输出是方形 circle, face 占比下限 ~50% (受方形 + bbox 物理限制)
  - 如果想 face < 50%, 改用 圆角矩形 + scale+crop 保持比例
  - 字幕 JSON 的 text 字段**不参与视频**, 只用来做时间轴
"""
import argparse
import os
import subprocess
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFilter

# === 路径配置 (环境变量 > 默认值) ===
# 可用环境变量:
#   OVERLAY_BASE    - 项目根目录 (默认: 脚本运行目录)
#   OVERLAY_FFMPEG  - ffmpeg 路径 (默认: 脚本同目录的 tools/ffmpeg/bin/ffmpeg.exe)
#   OVERLAY_SLIDE   - 幻灯片视频 (默认: $OVERLAY_BASE/output.mp4)
#   OVERLAY_AVATAR  - 数字人视频 (默认: $OVERLAY_BASE/digital_human_synth.mp4)
#   OVERLAY_OUTPUT  - 输出视频 (默认: $OVERLAY_BASE/output_final.mp4)
_DEFAULT_BASE = Path.cwd()
_DEFAULT_FFMPEG = str(Path(__file__).parent / "ffmpeg" / "bin" / "ffmpeg.exe")

BASE = Path(os.environ.get("OVERLAY_BASE", _DEFAULT_BASE))
FFMPEG = os.environ.get("OVERLAY_FFMPEG", _DEFAULT_FFMPEG)
SLIDE = os.environ.get("OVERLAY_SLIDE", str(BASE / "output.mp4"))
AVATAR = str(BASE / "digital_human_synth.mp4")
OUTPUT = str(BASE / "output_final.mp4")
THUMB = str(BASE / "avatar_thumb.jpg")
MASK = str(BASE / "avatar_mask.png")
SHADOW1 = str(BASE / "avatar_shadow1.png")
SHADOW2 = str(BASE / "avatar_shadow2.png")
RING = str(BASE / "avatar_ring.png")

# === 环境变量覆盖剩余路径 (兼容各种项目布局) ===
AVATAR = os.environ.get("OVERLAY_AVATAR", str(BASE / "digital_human_synth.mp4"))
OUTPUT = os.environ.get("OVERLAY_OUTPUT", str(BASE / "output_final.mp4"))

SRC_W, SRC_H = 1080, 1920

# === 视觉参数 ===
SIZE = 340            # 圆直径
MARGIN_X = 0          # 与左边缘相切
MARGIN_BOTTOM = 60    # 距底部
RING_THICKNESS = 5    # 主题渐变环厚度
HIGHLIGHT_ALPHA = 180 # 内边缘高光不透明度 (1px 白线)

# === 双层投影参数 (电影感) ===
# 近层: 紧贴主体, 小模糊, 低透明度 (ambient occlusion)
SHADOW1_OFFSET = (0, 4)
SHADOW1_BLUR = 10
SHADOW1_ALPHA = 90

# 远层: 离主体远, 大模糊, 中等透明度 (key light shadow)
SHADOW2_OFFSET = (8, 18)
SHADOW2_BLUR = 28
SHADOW2_ALPHA = 140


def extract_thumb():
    cmd = [FFMPEG, "-y", "-ss", "1.0", "-i", AVATAR,
           "-vframes", "1", "-update", "1", "-q:v", "2", THUMB]
    r = subprocess.run(cmd, capture_output=True, text=True)
    if r.returncode != 0:
        print("抽帧失败:", r.stderr[-500:])
        sys.exit(1)
    print(f"抽帧已保存: {THUMB}")


def make_assets():
    """生成: 圆形 mask + 双层投影 + 白色外环"""
    # --- Mask: 白色圆形 ---
    mask = Image.new("L", (SIZE, SIZE), 0)
    draw = ImageDraw.Draw(mask)
    draw.ellipse((0, 0, SIZE - 1, SIZE - 1), fill=255)
    mask.save(MASK)
    print(f"Mask: {MASK}  ({SIZE}x{SIZE} 圆形)")

    # --- Shadow 1 (近层 ambient) ---
    sh1 = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    d = ImageDraw.Draw(sh1)
    ox, oy = SHADOW1_OFFSET
    d.ellipse((ox, oy, SIZE - 1 + ox, SIZE - 1 + oy), fill=(0, 0, 0, SHADOW1_ALPHA))
    sh1 = sh1.filter(ImageFilter.GaussianBlur(SHADOW1_BLUR))
    sh1.save(SHADOW1)
    print(f"Shadow1: {SHADOW1}  (offset={SHADOW1_OFFSET}, blur={SHADOW1_BLUR}, α={SHADOW1_ALPHA})")

    # --- Shadow 2 (远层 direct) ---
    sh2 = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    d = ImageDraw.Draw(sh2)
    ox, oy = SHADOW2_OFFSET
    d.ellipse((ox, oy, SIZE - 1 + ox, SIZE - 1 + oy), fill=(0, 0, 0, SHADOW2_ALPHA))
    sh2 = sh2.filter(ImageFilter.GaussianBlur(SHADOW2_BLUR))
    sh2.save(SHADOW2)
    print(f"Shadow2: {SHADOW2}  (offset={SHADOW2_OFFSET}, blur={SHADOW2_BLUR}, α={SHADOW2_ALPHA})")

    # --- Ring: 主题色渐变环 (#ff3366 → #7c3aed, 135°) + 1px 内边缘高光 ---
    # 用 32x32 渐变再 resize 到 ring_size, 既快又平滑
    ring_size = SIZE + 2 * RING_THICKNESS
    small = 32
    gradient = Image.new("RGBA", (small, small))
    for y in range(small):
        for x in range(small):
            t = (x + y) / (2 * (small - 1))
            r = int(255 * (1 - t) + 124 * t)
            g = int(51 * (1 - t) + 58 * t)
            b = int(102 * (1 - t) + 237 * t)
            gradient.putpixel((x, y), (r, g, b, 255))
    gradient = gradient.resize((ring_size, ring_size), Image.BICUBIC)

    # 挖空成圆环
    mask = Image.new("L", (ring_size, ring_size), 0)
    d = ImageDraw.Draw(mask)
    d.ellipse((0, 0, ring_size - 1, ring_size - 1), fill=255)
    d.ellipse((RING_THICKNESS, RING_THICKNESS,
               ring_size - 1 - RING_THICKNESS, ring_size - 1 - RING_THICKNESS),
              fill=0)
    gradient.putalpha(mask)

    # 在环内边缘加 1px 白色高光 (金属质感, "polish" 效果)
    inner_radius = (ring_size - 2 * RING_THICKNESS) / 2
    hl = Image.new("RGBA", (ring_size, ring_size), (0, 0, 0, 0))
    hd = ImageDraw.Draw(hl)
    hd.ellipse((RING_THICKNESS - 1, RING_THICKNESS - 1,
                ring_size - RING_THICKNESS, ring_size - RING_THICKNESS),
               fill=(255, 255, 255, HIGHLIGHT_ALPHA))
    hd.ellipse((RING_THICKNESS, RING_THICKNESS,
                ring_size - 1 - RING_THICKNESS, ring_size - 1 - RING_THICKNESS),
               fill=(0, 0, 0, 0))
    gradient = Image.alpha_composite(gradient, hl)
    gradient.save(RING)
    print(f"Ring: {RING}  ({ring_size}x{ring_size}, 厚 {RING_THICKNESS}px, 主题渐变 + 内高光)")


def run_ffmpeg_overlay(face_bbox):
    x, y, w, h = face_bbox
    print(f"源裁剪: x={x}, y={y}, {w}x{h}  →  输出 {SIZE}x{SIZE} 圆形 + 高级感双层投影 + 白环")

    bbox_aspect = h / w
    if abs(bbox_aspect - 1.0) < 0.01:
        scale_crop = f"scale={SIZE}:{SIZE}"
    elif bbox_aspect > 1.0:
        scale_crop = f"scale={SIZE}:-1,crop={SIZE}:{SIZE}:0:0"
    else:
        scale_crop = f"scale=-1:{SIZE},crop={SIZE}:{SIZE}:0:0"
    print(f"bbox 比例 {bbox_aspect:.2f} → {scale_crop}")

    make_assets()

    # 位置计算
    avatar_x = MARGIN_X
    avatar_y = 1080 - SIZE - MARGIN_BOTTOM
    sh1_x = avatar_x + SHADOW1_OFFSET[0]
    sh1_y = avatar_y + SHADOW1_OFFSET[1]
    sh2_x = avatar_x + SHADOW2_OFFSET[0]
    sh2_y = avatar_y + SHADOW2_OFFSET[1]
    ring_x = avatar_x - RING_THICKNESS
    ring_y = avatar_y - RING_THICKNESS
    print(f"avatar @ ({avatar_x}, {avatar_y})")
    print(f"shadow1 @ ({sh1_x}, {sh1_y})  shadow2 @ ({sh2_x}, {sh2_y})")
    print(f"ring @ ({ring_x}, {ring_y})")

    # 5 个输入: slide(0) + avatar(1) + mask(2) + shadow1(3) (no ring, single subtle shadow)
    # 合成顺序 (从底到顶): slide → shadow1 → avatar
    filter_complex = (
        f"[1:v]crop={w}:{h}:{x}:{y},"
        f"{scale_crop},"
        f"format=rgba"
        f"[avatar_raw];"
        f"[avatar_raw][2:v]alphamerge"
        f"[avatar_masked];"
        f"[0:v][3:v]overlay={sh1_x}:{sh1_y}"  # 单层微妙投影
        f"[bg_s1];"
        f"[bg_s1][avatar_masked]overlay={avatar_x}:{avatar_y}"  # 头像
        f"[v]"
    )

    cmd = [
        FFMPEG, "-y",
        "-i", SLIDE,
        "-i", AVATAR,
        "-loop", "1", "-i", MASK,
        "-loop", "1", "-i", SHADOW1,
        "-filter_complex", filter_complex,
        "-map", "[v]",
        "-map", "1:a",
        "-c:v", "libx264", "-preset", "medium", "-crf", "23",
        "-pix_fmt", "yuv420p",
        "-c:a", "aac", "-b:a", "192k",
        "-shortest",
        OUTPUT,
    ]

    result = subprocess.run(cmd, capture_output=True, text=True)
    print(f"Return code: {result.returncode}")
    if result.returncode != 0:
        print("STDERR (last 2000 chars):")
        print(result.stderr[-2000:])
        return False
    out = Path(OUTPUT)
    if out.exists():
        size_mb = out.stat().st_size / (1024 * 1024)
        print(f"OK -> {OUTPUT}  ({size_mb:.2f} MB)")
    return True


def main():
    p = argparse.ArgumentParser()
    p.add_argument("bbox", nargs="*", type=int, help="bbox: x y w h (源坐标)")
    p.add_argument("--thumb-only", action="store_true", help="只抽一帧")
    args = p.parse_args()

    if args.thumb_only:
        extract_thumb()
        return

    if not args.bbox or len(args.bbox) != 4:
        print("需要 4 个 bbox 参数: x y w h")
        sys.exit(1)

    run_ffmpeg_overlay(tuple(args.bbox))


if __name__ == "__main__":
    main()
