# luma-cli

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.23-blue.svg)](https://go.dev/)

[中文](./README.zh.md) | [English](./README.md)

面向 AI Agent 的 PikGeo/Luma 媒体生产 CLI。`luma-cli` 是托管后端的开源客户端，把视频、音频、素材、字幕、数字人能力包装成稳定的原子命令，让 Agent 可以像调用工具一样组合工作流。

[快速开始](#快速开始) · [Agent 使用方式](#agent-使用方式) · [原子工具](#原子工具) · [Skills](#agent-skills) · [架构](#架构) · [分发与发布](#分发与发布) · [开发](#开发)

## 为什么需要 luma-cli？

- **Agent 原生**：CLI 只提供原子能力，复杂业务流程放在 `skills/` 说明书里，避免把业务胶水写死在命令里。
- **工具可发现**：`luma-cli tools list` 和 `luma-cli tools describe <id>` 可以输出 Agent 可读的工具契约。
- **工作流可维护**：Agent 的编排策略写在 Skill 中，CLI 命令保持小而稳定。
- **结构化输出**：支持 `--json`，面向 Agent 返回稳定的 `{ ok, code, error, data }` 结构。
- **项目工作区**：支持 project，把 source、audio、subtitles、effects、output、tmp 和处理历史组织起来。
- **根目录清爽**：仓库根目录只保留入口和项目文件，命令实现放在 `internal/commands/`。
- **托管后端边界**：模型执行、任务调度、计费、注册和账号体系放在后端，不放在这个开源仓库。

## 能力概览

| 领域 | 命令 | 说明 |
| --- | --- | --- |
| 鉴权 | `auth login`, `auth status` | 保存和查看后端调用使用的 card key。 |
| 素材 | `asset upload`, `asset list` | 上传本地素材，查看 voice、roles 等云端素材分组。 |
| 语音识别 | `asr` | 对本地音频或视频做云端 ASR。 |
| 语音合成 | `tts` | 使用指定音色把文本合成为语音。 |
| 数字人 | `lipsync` | 使用头像视频和音频生成口型同步视频。 |
| 视频增强 | `enhance` | 对本地视频做增强或超分。 |
| 字幕 | `subtitle` | 生成 ASS 字幕，并可烧录到视频。 |
| 项目 | `project create/list/use/info/clean` | 管理本地视频项目工作区和处理历史。 |
| 任务 | `task status` | 查询云端任务状态和结果。 |
| Agent 工具 | `tools list`, `tools describe` | 查看 Agent 可调用的原子工具契约。 |

## 快速开始

### 环境要求

- Go `1.23` 或更高版本
- 一个有效的 Luma card key

### 从源码构建

```bash
git clone git@github.com:zl007700/luma-cli.git
cd luma-cli
go build -o luma-cli .
```

Windows PowerShell：

```powershell
go build -o luma-cli.exe .
```

### 配置登录

```bash
luma-cli auth login <CARD_KEY>
luma-cli auth status
```

也可以通过环境变量传入，适合 CI、Agent 沙箱或临时运行：

```bash
export LUMA_CARD_KEY=<CARD_KEY>
```

PowerShell：

```powershell
$env:LUMA_CARD_KEY = "<CARD_KEY>"
```

如果要隔离配置目录：

```bash
export LUMA_CONFIG_DIR=/tmp/luma-cli-config
```

PowerShell：

```powershell
$env:LUMA_CONFIG_DIR = "$env:TEMP\luma-cli-config"
```

默认 API 地址：

```bash
https://api.pikgeo.com
```

开发环境或私有部署可以覆盖：

```bash
export LUMA_API_URL=https://your-api.example.com
```

PowerShell：

```powershell
$env:LUMA_API_URL = "https://your-api.example.com"
```

### 第一次调用

```bash
# 语音识别
luma-cli asr input.mp4 --language zh

# 语音合成
luma-cli tts "你好，欢迎来到直播间" --voice 男声3 --speech-rate 1.1

# 数字人口型同步
luma-cli lipsync --avatar 数字人男 --audio tts_output.wav --output output.mp4

# 视频增强
luma-cli enhance output.mp4 --scale 2
```

## Agent 使用方式

Agent 应该把 `luma-cli` 当成一组原子工具，而不是一个写死业务流程的大命令。

推荐流程：

1. 先运行 `luma-cli tools list` 查看可用工具。
2. 第一次调用某个工具前，运行 `luma-cli tools describe <tool_id>` 查看参数、风险和输出。
3. 如果任务需要多步编排，读取 `skills/` 里对应的 Skill。
4. 优先传入明确的输入路径和输出路径。
5. 对多步骤媒体任务，优先使用 `project create/use` 组织产物。

示例：

```bash
luma-cli tools list
luma-cli tools describe asr.transcribe
luma-cli --json tools describe tts.synthesize
```

结构化输出示例：

```json
{
  "ok": true,
  "data": {
    "id": "tts.synthesize",
    "service": "tts",
    "command": "luma-cli tts",
    "risk": "write",
    "flags": [],
    "outputs": []
  }
}
```

## 原子工具

| Tool ID | 命令 | 风险 | 相关 Skill |
| --- | --- | --- | --- |
| `asr.transcribe` | `luma-cli asr` | write | `luma-subtitle`, `luma-video-workflow` |
| `tts.synthesize` | `luma-cli tts` | write | `luma-digital-human`, `luma-video-workflow` |
| `lipsync.create` | `luma-cli lipsync` | write | `luma-digital-human`, `luma-video-workflow` |
| `video.enhance` | `luma-cli enhance` | write | `luma-video-workflow` |
| `subtitle.render` | `luma-cli subtitle` | write | `luma-subtitle` |
| `asset.upload` | `luma-cli asset upload` | write | `luma-assets`, `luma-digital-human` |
| `asset.list` | `luma-cli asset list` | read | `luma-assets` |
| `task.status` | `luma-cli task status` | read | `luma-video-workflow` |

这些工具契约的代码来源在 `shortcuts/`。

## Agent Skills

| Skill | 用途 |
| --- | --- |
| `luma-assets` | 素材查询、上传规范、音色和数字人素材选择。 |
| `luma-digital-human` | TTS + lipsync 的数字人视频工作流。 |
| `luma-subtitle` | 字幕生成、切分、样式、烧录工作流。 |
| `luma-video-workflow` | ASR、TTS、数字人、增强、任务查询等端到端媒体编排。 |

原则：Skill 写流程，CLI 做动作。

## 项目工作区

项目工作区适合多步媒体生产任务：

```bash
luma-cli project create demo-video
luma-cli project use demo-video
luma-cli project info
```

目录结构：

```text
source/     原始媒体文件
audio/      提取或生成的音频
subtitles/  SRT、ASS 字幕文件
effects/    特效叠加文件
output/     最终输出
tmp/        临时文件
```

## 架构

```text
main.go                 极薄 CLI 入口
internal/commands/      命令路由和 CLI 适配层
internal/atom/          后端原子能力封装
internal/cmdutil/       参数解析等命令工具
internal/config/        本地配置和 card key 加载
internal/output/        结构化输出 envelope
shortcuts/              Agent 工具元数据
skills/                 Agent 工作流说明书
project/                本地项目工作区模型
subtitle/               字幕生成和渲染逻辑
cloud/                  后端 API client
```

边界规则：

```text
skills 负责描述业务工作流
shortcuts 负责描述可调用工具
commands 负责把 CLI 参数适配到 atom
atom 负责调用后端原子能力
```

## 开源边界

这个仓库适合作为开源客户端：CLI 适配层、工具元数据、本地 project 工作区、Agent skills 都可以开放。

闭源后端继续负责：

- 用户注册和账号体系；
- 计费和权益校验；
- 模型执行和任务调度；
- 生产 prompt 和内部 workflow 策略；
- 私有素材库和运营后台工具。

不要提交生产 card key、内部 token、私有模型地址、数据库连接串、bucket 凭证或未公开 prompt。非默认后端环境请使用 `LUMA_API_URL`。

## 分发与发布

当前仓库已经按 Lark CLI 的方式补了 GitHub Release + npm 安装器。

发布链路：

```text
git tag v0.0.1
        ↓
GitHub Actions 触发 release workflow
        ↓
GoReleaser 构建多平台二进制
        ↓
GitHub Release 托管压缩包和 checksums.txt
        ↓
npm 发布轻量安装器
        ↓
用户 npm install 时自动下载对应平台的 Go 二进制
```

已经新增的文件：

```text
package.json
scripts/install.js
scripts/run.js
.goreleaser.yml
.github/workflows/release.yml
```

发布前需要准备：

```text
GitHub 仓库有创建 release 的权限
npm 上拥有 package.json 里的包名 `@lumageo/luma-cli`，或者把它改成你自己的 npm 包名
GitHub Actions 配置 NPM_TOKEN
```

发布 `0.0.1`：

```bash
git tag v0.0.1
git push origin v0.0.1
```

workflow 会自动：

1. 运行 `go test ./...`；
2. 构建 `darwin/linux/windows` + `amd64/arm64`；
3. 上传 release archives 和 `checksums.txt`；
4. 发布 npm 包。

发布前本地检查：

```bash
go test ./...
go build ./...
npm pack --dry-run
```

用户安装：

```bash
npm install -g @lumageo/luma-cli
luma-cli auth login <CARD_KEY>
luma-cli tools list
```

如果 npm scope 或包名需要调整，修改：

```text
package.json name
README 安装命令
```

例如改成 scoped 包：

```json
{
  "name": "@your-scope/luma-cli"
}
```

### Clawhub / Agent 分发

建议发布内容：

```text
安装命令       npm install -g @lumageo/luma-cli
健康检查       luma-cli version / luma-cli auth status
工具发现       luma-cli tools list
工具详情       luma-cli --json tools describe <tool_id>
Skills 目录    skills/
安全说明       哪些 tool 是 read，哪些 tool 是 write
```

后续可以增加一个 `manifest`：

```bash
luma-cli tools manifest --json
```

返回 CLI 名称、版本、skills、tools、认证方式、环境变量、风险等级，方便 Clawhub 自动接入。

当前阶段暂时不做 Clawhub 接入。

## 注册与登录

目前 CLI **还不能自助注册 key**。

现在已有能力：

```bash
luma-cli auth login <CARD_KEY>
luma-cli auth status
```

也支持：

```bash
LUMA_CARD_KEY=<CARD_KEY> luma-cli tools list
```

当前 `card_key` 会作为 `X-User-Id` 请求头传给后端。代码里还没有注册接口、设备码登录、浏览器登录或购买/开通流程。

如果要做到用户级分发，建议后端补一个正式的注册/登录协议：

```text
POST /v1/auth/register        创建用户并返回 card_key，或返回下一步验证信息
POST /v1/auth/login           邮箱/手机号/验证码登录
POST /v1/auth/device-code     给 Agent/CLI 用的浏览器确认登录
GET  /v1/auth/status          校验当前 key 是否有效
```

CLI 对应命令：

```bash
luma-cli auth register
luma-cli auth login
luma-cli auth login <CARD_KEY>
luma-cli auth status
luma-cli auth logout
```

在后端注册协议出来之前，CLI 只能支持“已有 key 的用户登录”。

## 开发

```bash
go test ./...
go build ./...
go run . help
go run . tools list
go run . --json tools describe asr.transcribe
```

Windows 本地 Go 工具链示例：

```powershell
$goRoot = Join-Path $env:USERPROFILE ".local\go\go1.25.5"
$env:GOROOT = $goRoot
$env:Path = (Join-Path $goRoot "bin") + ";" + $env:Path
$env:GOTOOLCHAIN = "local"
go test ./...
go build ./...
```

## 安全说明

`luma-cli` 可以被 AI Agent 调用，并通过当前配置的 card key 上传、下载、创建或修改媒体资源。所有 `risk: write` 的工具都应该被视为有副作用的操作。

建议：

- 不要把 card key 写进公开日志。
- Agent 沙箱使用独立的 `LUMA_CONFIG_DIR`。
- 多步任务显式传入输出路径。
- Agent 调用陌生工具前先运行 `tools describe`。
- 把流程策略放进 Skill，不要把复杂业务自动化硬编码进 CLI。
