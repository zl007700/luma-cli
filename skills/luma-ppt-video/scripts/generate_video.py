"""
PPT Video Generator - 自动根据时间戳生成视频
依赖: pip install playwright && playwright install chromium
"""

import asyncio
import json
import sys
import subprocess
import shutil
from pathlib import Path

try:
    from playwright.async_api import async_playwright
except ImportError:
    print("错误: 请先安装 playwright")
    print("运行: pip install playwright && playwright install chromium")
    sys.exit(1)


def load_config():
    """加载配置"""
    config = {}

    # 从当前目录加载配置
    config_file = Path(__file__).parent.parent / "config.json"
    if config_file.exists():
        with open(config_file, 'r', encoding='utf-8') as f:
            user_config = json.load(f)
            config.update(user_config)

    # 环境变量覆盖
    import os
    if os.environ.get('SUBTITLE_FILE'):
        config['subtitle_file'] = os.environ['SUBTITLE_FILE']
    if os.environ.get('HTML_FILE'):
        config['html_file'] = os.environ['HTML_FILE']
    if os.environ.get('OUTPUT_FILE'):
        config['output_file'] = os.environ['OUTPUT_FILE']
    if os.environ.get('FFMPEG_PATH'):
        config['ffmpeg_path'] = os.environ['FFMPEG_PATH']

    # 默认值
    config.setdefault('subtitle_file', str(Path(__file__).parent.parent / "subtitle_segments.json"))
    config.setdefault('html_file', str(Path(__file__).parent.parent / "index.html"))
    config.setdefault('output_file', str(Path(__file__).parent.parent / "output.mp4"))
    config.setdefault('ffmpeg_path', str(Path(__file__).parent.parent / "tools" / "ffmpeg.exe"))
    config.setdefault('video_size', [1920, 1080])
    config.setdefault('extra_time', 3.0)

    return config


def build_slide_schedule(sentence_units, config):
    """构建幻灯片播放时间表"""
    sentence_to_slide = config.get('sentence_to_slide', {})

    schedule = []
    prev_slide = None
    slide_start = 0.0

    for unit in sentence_units:
        sent_id = unit['sent_id']
        start = unit['start']
        end = unit['end']
        slide = sentence_to_slide.get(sent_id, 0)

        if slide != prev_slide:
            if prev_slide is not None:
                schedule.append({
                    'slide': prev_slide,
                    'start': slide_start,
                    'duration': start - slide_start
                })
            prev_slide = slide
            slide_start = start

    if prev_slide is not None:
        last_end = sentence_units[-1]['end']
        extra = config.get('extra_time', 3.0)
        schedule.append({
            'slide': prev_slide,
            'start': slide_start,
            'duration': last_end - slide_start + extra
        })

    return schedule


async def scroll_to_slide(page, slide_index):
    """滚动到指定幻灯片"""
    await page.evaluate(f"""
        const container = document.getElementById('container');
        if (container) {{
            const slideWidth = window.innerWidth;
            container.scrollTo({{ left: slideWidth * {slide_index}, behavior: 'smooth' }});
        }}
    """)
    await page.wait_for_timeout(800)


async def capture_screenshots(schedule, temp_dir, config):
    """截图并保存到临时文件夹"""
    html_file = config['html_file']
    video_size = config['video_size']

    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=True)
        context = await browser.new_context(
            viewport={"width": video_size[0], "height": video_size[1]},
            device_scale_factor=1,
        )
        page = await context.new_page()

        html_path = Path(html_file).resolve().as_uri()
        print(f"打开网页: {html_path}")
        await page.goto(html_path)
        await page.wait_for_load_state('networkidle')
        await page.wait_for_timeout(1000)

        print("\n开始录制截图...")

        frame_count = 0
        for i, item in enumerate(schedule):
            slide = item['slide']
            duration = item['duration']

            if i == 0 or slide != schedule[i-1]['slide']:
                print(f"  切换到幻灯片 {slide}...")
                await scroll_to_slide(page, slide)

            num_frames = max(1, int(duration * 30))
            print(f"    截图 {num_frames} 帧 ({duration:.2f}秒)...")

            for frame_num in range(num_frames):
                frame_path = temp_dir / f"{frame_count:06d}.jpg"
                await page.screenshot(path=str(frame_path), timeout=30000)
                frame_count += 1
                await asyncio.sleep(duration / num_frames)

        await browser.close()
        return frame_count


def generate_video_from_frames(temp_dir, config):
    """使用 ffmpeg 从图片序列生成视频"""
    ffmpeg_path = config['ffmpeg_path']
    output_file = config['output_file']
    video_size = config['video_size']

    print("\n正在生成视频文件...")

    first_file = temp_dir / "000000.jpg"
    if not first_file.exists():
        print(f"错误: 找不到截图文件")
        return False

    frame_files = list(temp_dir.glob("*.jpg"))
    print(f"找到 {len(frame_files)} 张截图")

    ffmpeg_cmd = [
        ffmpeg_path,
        '-y',
        '-framerate', '30',
        '-i', str(temp_dir / '%06d.jpg'),
        '-c:v', 'libx264',
        '-preset', 'ultrafast',
        '-crf', '23',
        '-pix_fmt', 'yuv420p',
        '-movflags', '+faststart',
        '-s', f'{video_size[0]}x{video_size[1]}',
        '-threads', '1',
        '-refs', '1',
        '-bf', '0',
        str(output_file)
    ]

    print(f"执行 ffmpeg 合并 {len(frame_files)} 帧...")

    result = subprocess.run(ffmpeg_cmd, capture_output=True, text=True)

    if result.returncode != 0:
        print(f"ffmpeg 错误:")
        print(result.stderr[-2000:])
        return False

    return True


async def generate_video(config):
    """主函数"""
    subtitle_file = config['subtitle_file']
    html_file = config['html_file']
    output_file = config['output_file']
    ffmpeg_path = config['ffmpeg_path']

    print("=" * 50)
    print("PPT 视频生成器")
    print("=" * 50)

    if not Path(subtitle_file).exists():
        print(f"错误: 找不到字幕文件 {subtitle_file}")
        return False

    if not Path(html_file).exists():
        print(f"错误: 找不到网页文件 {html_file}")
        return False

    # 检查 ffmpeg
    if not Path(ffmpeg_path).exists():
        # 尝试在 tools 目录查找
        alt_paths = [
            Path(__file__).parent.parent / "tools" / "ffmpeg.exe",
            Path(__file__).parent.parent / "tools" / "ffmpeg" / "bin" / "ffmpeg.exe",
        ]
        for p in alt_paths:
            if p.exists():
                ffmpeg_path = str(p)
                config['ffmpeg_path'] = ffmpeg_path
                break
        else:
            print(f"警告: 找不到 ffmpeg: {ffmpeg_path}")

    # 删除旧的输出文件
    output_path = Path(output_file)
    if output_path.exists():
        output_path.unlink()

    # 创建临时目录
    temp_dir = Path(__file__).parent.parent / "temp_frames"
    if temp_dir.exists():
        shutil.rmtree(temp_dir)
    temp_dir.mkdir(parents=True)

    # 加载字幕数据
    print(f"\n加载字幕数据: {subtitle_file}")
    with open(subtitle_file, 'r', encoding='utf-8') as f:
        data = json.load(f)

    sentence_units = data.get('sentence_units', data.get('segments', []))
    # 如果是 segments 格式，转换为 sentence_units
    if 'segments' in data and 'sentence_units' not in data:
        sentence_units = data['segments']

    print(f"共 {len(sentence_units)} 个句子单元")

    # 构建播放表
    schedule = build_slide_schedule(sentence_units, config)
    print(f"共 {len(schedule)} 次幻灯片切换")

    total_duration = sum(item['duration'] for item in schedule)
    print(f"预计视频时长: {total_duration:.1f} 秒")

    print("\n启动 Chromium 浏览器...")

    # 截图
    total_frames = await capture_screenshots(schedule, temp_dir, config)
    print(f"截图完成, 共 {total_frames} 帧")

    # 生成视频
    success = generate_video_from_frames(temp_dir, config)

    # 清理临时文件
    print("清理临时文件...")
    shutil.rmtree(temp_dir)

    if success:
        print(f"\n视频已生成: {output_file}")
        if Path(output_file).exists():
            size_mb = Path(output_file).stat().st_size / (1024 * 1024)
            print(f"文件大小: {size_mb:.2f} MB")

    return success


def main():
    config = load_config()

    # 检查 ffmpeg
    ffmpeg_path = config['ffmpeg_path']
    try:
        subprocess.run([ffmpeg_path, '-version'], capture_output=True, check=True)
    except (subprocess.CalledProcessError, FileNotFoundError):
        print(f"错误: 找不到 ffmpeg: {ffmpeg_path}")
        print("请确保 ffmpeg 在 tools 目录或配置正确路径")
        sys.exit(1)

    success = asyncio.run(generate_video(config))
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
