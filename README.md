# luma-cli

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.23-blue.svg)](https://go.dev/)

[中文](./README.md) | [English](./README.en.md)

给 AI Agent 配上的视频运营与内容制作技能包。`luma-cli` 让 Agent 具备一组可调用的专业视频能力：识别素材、合成语音、生成数字人口播、制作字幕、增强画质、管理制作项目，并把产物整理成适合运营使用的短视频资产。

它不是一个面向开发者炫技的命令集合，而是一个面向创作者、运营团队和 Agent 使用者的专业能力包：你把需求交给 Agent，Agent 通过 `luma-cli` 调用视频制作能力，完成从脚本、配音、数字人到字幕和成片增强的工作。

[快速开始](#快速开始) · [适合做什么](#适合做什么) · [核心能力](#核心能力) · [给-agent-使用](#给-agent-使用) · [项目工作区](#项目工作区)

## 适合做什么

- **短视频批量制作**：把文案变成配音、数字人口播、字幕和可发布视频。
- **运营内容生产**：围绕直播、产品介绍、营销素材、知识讲解等场景快速生成视频资产。
- **素材整理和复用**：管理音色、数字人、原始视频、输出视频和中间产物。
- **字幕与包装**：生成适合短视频观看节奏的字幕，并支持高亮、样式和烧录。
- **Agent 视频技能增强**：让 Claude Code、Codex、Clawhub 等 Agent 不只是写代码，也能调用视频制作能力。

## 为什么需要 luma-cli？

- **让 Agent 真的能做视频**：Agent 可以直接调用 ASR、TTS、数字人口型、字幕、增强、素材管理等能力。
- **面向运营工作流**：不是单点 API 示例，而是围绕“做一条可用的视频”组织能力。
- **安装后即可发现能力**：`luma-cli tools list` 可以列出 Agent 能调用的所有工具。
- **适合团队沉淀经验**：视频制作方法、运营 SOP、字幕风格、数字人使用方式可以沉淀在 `skills/` 里。
- **后端能力托管**：注册、计费、模型执行、任务调度在 PikGeo 后端完成，CLI 只负责把能力带到用户和 Agent 身边。

## 核心能力

| 场景 | 命令 | 说明 |
| --- | --- | --- |
| 登录 | `auth login`, `auth status` | 保存和查看后端调用所需的 card key。 |
| 素材 | `asset upload`, `asset list` | 上传素材，查看音色、数字人等资源。 |
| 语音识别 | `asr` | 从音频或视频中识别文字。 |
| 语音合成 | `tts` | 把文案合成为指定音色的语音。 |
| 数字人口播 | `lipsync` | 用数字人形象和音频生成口播视频。 |
| 视频增强 | `enhance` | 对视频进行画质增强或超分。 |
| 字幕制作 | `subtitle` | 生成字幕，并可烧录到视频。 |
| 项目管理 | `project create/list/use/info/clean` | 管理本地视频项目、产物和处理历史。 |
| 任务查询 | `task status` | 查询云端任务状态。 |
| Agent 工具发现 | `tools list`, `tools describe` | 查看 Agent 可调用能力和参数说明。 |

## 快速开始

安装：

```bash
npm install -g @lumageo/luma-cli
```

登录：

```bash
luma-cli auth login <CARD_KEY>
luma-cli auth status
```

查看 Agent 可用能力：

```bash
luma-cli tools list
luma-cli tools describe tts.synthesize
```

第一次制作：

```bash
# 文案转语音
luma-cli tts "你好，欢迎来到直播间" --voice 男声3

# 数字人口播
luma-cli lipsync --avatar 数字人男 --audio tts_output.wav --output output.mp4

# 加字幕
luma-cli subtitle output.mp4 --output output_subtitled.mp4

# 增强画质
luma-cli enhance output_subtitled.mp4 --scale 2
```

## 给 Agent 使用

Agent 不需要猜命令怎么用，先让它发现能力：

```bash
luma-cli tools list
luma-cli --json tools describe asr.transcribe
```

典型 Agent 流程：

1. 根据用户目标选择视频制作技能。
2. 用 `tools list` 查看当前 CLI 支持什么。
3. 用 `tools describe <tool_id>` 查看参数、风险和输出。
4. 调用一个个原子能力完成制作。
5. 把多步任务产物放入 project 工作区。

## 内置 Skills

`skills/` 是给 Agent 看的专业说明书，用来告诉 Agent 如何把多个能力组合成运营和视频制作工作流。

| Skill | 说明 |
| --- | --- |
| `luma-video-workflow` | 从素材到成片的完整视频制作流程。 |
| `luma-digital-human` | 数字人口播、配音、口型同步相关流程。 |
| `luma-subtitle` | 字幕生成、切分、样式和烧录流程。 |
| `luma-assets` | 音色、数字人、素材上传和选择流程。 |

## 项目工作区

多步视频任务建议创建项目，方便整理素材和产物：

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
output/     最终成片
tmp/        临时文件
```

## 安全说明

`luma-cli` 可以被 AI Agent 调用，并通过当前配置的 card key 创建、上传、下载或修改媒体资源。所有 `risk: write` 的工具都应被视为有副作用的操作。

安全问题请参考 [SECURITY.md](./SECURITY.md)。
