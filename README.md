# luma-cli

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.23-blue.svg)](https://go.dev/)

中文 | [English](./README.en.md)

`luma-cli` 是给 AI Agent 使用的视频运营与内容制作能力包。它把 PikGeo / Luma 的视频制作能力包装成稳定的命令行原子能力，让 Agent 可以完成爆款仿写、内容研究、脚本改写、语音合成、声音克隆、数字人口播、画中画素材穿插、字幕、BGM、封面和项目产物管理。

它不是给用户手工点来点去的工具，而是给 Agent 配上的一套专业视频制作技能。复杂 workflow 放在 `skills/` 说明书里，CLI 本身保持为可组合、可测试、可复用的原子能力。

## 适合做什么

- 给 Agent 增加短视频制作能力。
- 从对标内容研究到脚本仿写，再到数字人口播成片。
- 管理音色、数字人、素材、字幕、BGM、封面等中间产物。
- 让运营 SOP 沉淀成 `skills/`，由 Agent 按步骤执行。

## 安装

```bash
npm install -g @lumageo/luma-cli
```

首次使用需要登录：

```bash
luma-cli auth login <CARD_KEY>
luma-cli auth status
```

安装本地视频运行时：

```bash
luma-cli runtime install ffmpeg
```

## Agent 工具发现

```bash
luma-cli tools list
luma-cli tools describe tts.synthesize
luma-cli --json tools describe pip.plan
```

常用能力：

| 能力 | 命令 |
| --- | --- |
| 内容研究 | `research run`, `research export`, `research keywords` |
| 爆款改写 | `script rewrite` |
| 声音克隆 | `voice clone`, `voice list` |
| 语音合成 | `tts` |
| 数字人口播 | `lipsync` |
| 本地素材描述 | `material describe`, `material group list`, `material group describe`, `material search`, `material merge`, `material understand` |
| 画中画 | `pip scene`, `pip match`, `pip plan`, `pip render` |
| 字幕 | `subtitle` |
| BGM | `bgm mix` |
| 封面 | `cover frame`, `cover render` |
| 项目管理 | `project create/use/info`, `project artifact list/schema` |

## 爆款仿写流程

内置 skill：

```text
skills/luma-viral-remix-workflow/SKILL.md
```

标准流程：

```bash
luma-cli project create viral-demo
luma-cli project use viral-demo

luma-cli research run --role "AI工具创业者，想找适合口播拆解的爆款选题" --output step0_content_research.json
luma-cli research export --input step0_content_research.json --output step0_content_research.csv
luma-cli research keywords --input step0_content_research.json --output step0_keywords.json --csv step0_keywords.csv

luma-cli script rewrite --input source_script.txt --output step1_rewrite.json
luma-cli tts "<rewritten_text>" --voice 男声3 --output step2_tts.wav
luma-cli lipsync --avatar 数字人男 --audio step2_tts.wav --output step3_lipsync.mp4

luma-cli material group describe ./material_library/groups/vlm_ai --output step4_materials_enriched.json
luma-cli material search --materials step4_materials_enriched.json --query "<rewritten_text>" --limit 8 --output step4_material_matches.json
luma-cli pip scene --segments step4_segments.json --output step4_scene_units.json
luma-cli pip match --scenes step4_scene_units.json --materials step4_materials_enriched.json --mode auto --output step4_material_matches.json
luma-cli pip plan --segments step4_segments.json --materials step4_materials_enriched.json --output step4_picture_in_picture_plan.json
luma-cli pip render step3_lipsync.mp4 --plan step4_picture_in_picture_plan.json --output step4_picture_in_picture.mp4

luma-cli subtitle step4_picture_in_picture.mp4 --output step5_subtitle.mp4
luma-cli cover frame step5_subtitle.mp4 --output step6_cover_frame.png
luma-cli cover render step6_cover_frame.png --title "封面标题" --output step6_cover.jpg
```

## 声音克隆

上传一段参考音频，保存为后续 TTS 可使用的音色：

```bash
luma-cli voice clone ./sample.wav --name my_voice
luma-cli voice list
luma-cli tts "这是一段测试口播" --voice my_voice --output step2_tts.wav
```

## 标准中间产物

建议 Agent 在多步任务中使用这些文件名：

| 步骤 | 文件 |
| --- | --- |
| 内容研究 | `step0_content_research.json`, `step0_content_research.csv` |
| 改写 | `step1_rewrite.json` |
| 音频 | `step2_tts.wav` |
| 数字人 | `step3_lipsync.mp4` |
| 画中画 | `step4_segments.json`, `step4_materials.json`, `step4_picture_in_picture_plan.json`, `step4_picture_in_picture.mp4` |
| 字幕 | `step5_subtitle.mp4` |
| 封面 | `step6_cover_frame.png`, `step6_cover.jpg` |

## 内置 Skills

| Skill | 说明 |
| --- | --- |
| `luma-viral-remix-workflow` | 爆款仿写完整流程 |
| `luma-video-workflow` | 视频制作通用流程 |
| `luma-digital-human` | 数字人、TTS、声音克隆相关流程 |
| `luma-subtitle` | 字幕生成、切分、样式和烧录 |
| `luma-assets` | 素材上传、选择和复用 |

## 项目工作区

```bash
luma-cli project create demo-video
luma-cli project use demo-video
luma-cli project info
```

项目目录：

```text
source/     原始素材
audio/      配音和音频
subtitles/  字幕文件
effects/    特效文件
output/     输出视频和封面
tmp/        临时文件
```

`project.json` 会记录 history 和 artifacts，方便 Agent 恢复上下文、定位中间产物和复跑步骤。
