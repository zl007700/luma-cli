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
luma-cli auth login <phone_or_account>
luma-cli auth complete
luma-cli auth status
```

如果你已经有 API key，或老用户还在使用 card key，也可以直接保存：

```bash
luma-cli auth login --key <API_KEY_OR_LEGACY_CARD_KEY>
```

安装本地视频运行时：

```bash
luma-cli runtime install ffmpeg
```

同步 Agent skills：

```bash
luma-cli skills sync
```

等价于：

```bash
npx -y skills add zl007700/luma-cli -g -y
```

后续更新 CLI 和 skills：

```bash
luma-cli update
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
| 声音克隆 | `voice clone` |
| 云端资产 | `asset list voice`, `asset list roles`, `asset upload` |
| 语音识别 | `asr` |
| 语音对齐 | `align` |
| 语音合成 | `tts` |
| 数字人口播 | `lipsync` |
| 本地素材描述 | `material describe`, `material group list`, `material group describe`, `material search`, `material merge`, `material understand` |
| 画中画 | `pip scene`, `pip match`, `pip plan`, `pip render` |
| 字幕 | `subtitle` |
| BGM | `bgm mix` |
| 封面 | `cover frame`, `cover render` |
| 图片/视频生成 | `image generate`, `video generate` |
| 项目管理 | `project create/use/info`, `project artifact list/schema` |

## 爆款仿写流程

内置 skill：

```text
skills/luma-workflow-viral-remix/SKILL.md
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
luma-cli asset list roles
luma-cli lipsync --avatar <selected_role_name> --audio step2_tts.wav --output step3_lipsync.mp4

luma-cli material group describe vlm_ai --output step4_materials_enriched.json
luma-cli material search --materials step4_materials_enriched.json --query "<rewritten_text>" --limit 8 --output step4_material_matches.json
luma-cli pip scene --segments step4_segments.json --output step4_scene_units.json
luma-cli pip match --scenes step4_scene_units.json --materials step4_materials_enriched.json --mode auto --output step4_material_matches.json
luma-cli pip plan --segments step4_segments.json --materials step4_materials_enriched.json --output step4_picture_in_picture_plan.json
luma-cli pip render step3_lipsync.mp4 --plan step4_picture_in_picture_plan.json --output step4_picture_in_picture.mp4

luma-cli subtitle step4_picture_in_picture.mp4 --output step5_subtitle.mp4
luma-cli bgm mix step5_subtitle.mp4 --output step6_bgm.mp4
luma-cli cover generate step4_picture_in_picture.mp4 --title "封面标题" --count 12 --output-dir step7_covers
```

## 声音克隆

上传一段参考音频，保存为后续 TTS 可使用的音色：

```bash
luma-cli voice clone ./sample.wav --name my_voice
luma-cli asset list voice
luma-cli tts "这是一段测试口播" --voice my_voice --output step2_tts.wav
```

## 查看可用音色和数字人

```bash
luma-cli asset list voice
luma-cli asset list roles
```

`voice list` 是旧版兼容入口，新流程统一用 `asset list voice` 查看音色资产。

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
| 背景音乐 | `step6_bgm.mp4` |
| 封面 | `step7_covers/cover_*.jpg` |

## 内置 Skills

| Skill | 说明 |
| --- | --- |
| `luma-shared` | 通用认证、项目、产物和失败处理规则 |
| `luma-core-topic-discovery` | 生成检索矩阵并调用 topic mine |
| `luma-core-topic-selection` | 基于 profile、历史和 signals 本地选题 |
| `luma-core-longform-plan` | 起草或按审核意见修改 longform plan |
| `luma-core-script-writing` | 起草或修改口播稿与审核 payload |
| `luma-content-script` | 纯胶水：串联选题、计划、素材、写稿和审核 |
| `luma-benchmark-discovery` | 发现和评估对标账号 |
| `luma-find-material` | 为已通过的内容计划查找证据和视觉素材 |
| `luma-workflow-viral-remix` | 爆款仿写完整流程 |
| `luma-digital-human` | 把已定稿脚本制作成数字人口播视频 |
| `luma-subtitle` | 字幕生成、切分、样式和烧录 |
| `luma-material` | 本地可复用素材库、素材检索和 PIP 匹配 |
| `luma-assets` | 素材上传、选择和复用 |

本地素材库默认放在 `~/.luma/material-library`。可以把常用素材组导入到默认库，之后 Agent 只需要按素材组名称引用：

```bash
luma-cli material library path
luma-cli material library import ./material_library/groups/vlm_ai --replace
luma-cli material group list
luma-cli material group describe vlm_ai --output materials.json
```

Luma 将命令行工具和 Agent skills 分开分发：npm 负责安装 `luma-cli`，skills 通过 skills installer 单独同步。用户通常只需要执行一次全量同步：

```bash
npm install -g @lumageo/luma-cli
luma-cli skills sync
```

`luma-cli skills sync` 会安装 Luma Agent skills，并写入本地同步记录；`luma-cli update` 会先更新 npm 包，再同步 skills。更多说明见 [docs/SKILLS.md](./docs/SKILLS.md)。

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

## 项目结构

`luma-cli` 按“命令壳 + 原子能力模块 + skills 说明书”的方式组织：

```text
cmd/luma-cli/             CLI 入口
internal/commands/        命令参数解析、输出和轻量调度
internal/commands/pip_*   画中画 scene/match/render 相关实现
internal/commands/material_* 本地素材库、素材组、素材检索
internal/clientruntime/   ffmpeg、字体、BGM、模板等本地运行时缓存
cloud/                    PikGeo / Luma 后端 API 客户端
project/                  项目目录、history、artifacts
shortcuts/                Agent 可发现的工具说明
skills/                   给 Agent 使用的流程说明书
scripts/                  npm 安装和运行入口
```

设计原则：

- CLI 命令保持原子化，不把完整业务流程写死在 Go 代码里。
- 多步骤视频制作流程放在 `skills/`，由 Agent 按中间产物逐步执行。
- 素材理解、语义匹配等高级能力由 Luma 云端服务提供。
- 本地只保留跨平台稳定的能力，例如文件组织、资源缓存、ffmpeg 渲染、结果整理。
- 新增能力时优先补 `tools describe` / `shortcuts`，让 Agent 能自动发现和理解参数。
