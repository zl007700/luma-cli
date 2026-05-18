# luma-cli

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.23-blue.svg)](https://go.dev/)

[中文](./README.zh.md) | [English](./README.md)

Agent-native CLI for PikGeo/Luma media creation workflows. `luma-cli` is an open-source client for a hosted backend. It exposes video, audio, asset, subtitle, and digital-human capabilities as small atomic commands that are easy for AI agents and humans to call safely.

[Quick Start](#quick-start) · [Agent Usage](#agent-usage) · [Atomic Tools](#atomic-tools) · [Skills](#agent-skills) · [Architecture](#architecture) · [Publishing](#publishing) · [Development](#development)

## Why luma-cli?

- **Agent-native design**: atomic CLI commands are separated from workflow glue, so agents can compose capabilities without coupling business logic into the binary.
- **Discoverable tool contracts**: `luma-cli tools list` and `luma-cli tools describe <id>` expose machine-readable metadata for agent planners.
- **Skills-first workflows**: long-form orchestration guidance lives in `skills/`, while commands stay focused on single backend capabilities.
- **Structured output path**: commands that need agent integration can use `--json` and return a stable `{ ok, code, error, data }` envelope.
- **Project-aware media pipeline**: optional project workspaces organize source files, audio, subtitles, effects, outputs, and processing history.
- **Thin root entrypoint**: the repository root stays clean; command implementations live under `internal/commands`.
- **Hosted backend boundary**: model execution, scheduling, billing, registration, and account management live in the hosted backend, not in this repository.

## Capabilities

| Domain | Commands | Description |
| --- | --- | --- |
| Authentication | `auth login`, `auth status` | Store and inspect the Luma card key used for backend calls. |
| Assets | `asset upload`, `asset list` | Upload local media assets and list cloud asset groups such as voices and roles. |
| ASR | `asr` | Transcribe local audio or video through cloud ASR. |
| TTS | `tts` | Synthesize speech from text using a named voice asset. |
| Lip Sync | `lipsync` | Generate a digital-human lip-sync video from an avatar and audio. |
| Enhance | `enhance` | Enhance or upscale local video files. |
| Subtitle | `subtitle` | Generate styled ASS subtitles and optionally burn them into video. |
| Project | `project create/list/use/info/clean` | Manage local video project workspaces and processing history. |
| Task | `task status` | Inspect backend task status and result metadata. |
| Agent Tools | `tools list`, `tools describe` | List and inspect agent-callable atomic tools. |

## Quick Start

### Requirements

- Go `1.23` or newer
- A valid Luma card key

### Build From Source

```bash
git clone git@github.com:zl007700/luma-cli.git
cd luma-cli
go build -o luma-cli .
```

### Install From npm

After a release is published:

```bash
npm install -g @lumageo/luma-cli
luma-cli auth login <CARD_KEY>
luma-cli tools list
```

### Configure

```bash
luma-cli auth login <CARD_KEY>
luma-cli auth status
```

Configuration is stored in the local Luma config directory. For isolated tests or agent sandboxes, set:

```bash
export LUMA_CONFIG_DIR=/tmp/luma-cli-config
```

On PowerShell:

```powershell
$env:LUMA_CONFIG_DIR = "$env:TEMP\luma-cli-config"
```

The default API endpoint is:

```bash
https://api.pikgeo.com
```

For development or private deployments, override it with:

```bash
export LUMA_API_URL=https://your-api.example.com
```

On PowerShell:

```powershell
$env:LUMA_API_URL = "https://your-api.example.com"
```

### First Commands

```bash
# Transcribe media
luma-cli asr input.mp4 --language zh

# Synthesize speech
luma-cli tts "你好，欢迎来到直播间" --voice 男声3 --speech-rate 1.1

# Generate a lip-sync video
luma-cli lipsync --avatar 数字人男 --audio tts_output.wav --output output.mp4

# Enhance video
luma-cli enhance output.mp4 --scale 2
```

## Agent Usage

Agents should treat `luma-cli` as a collection of atomic backend tools. Do not hard-code multi-step business workflows into command calls; load the relevant skill instructions from `skills/` and compose tools from there.

### Discover Tools

```bash
luma-cli tools list
luma-cli tools describe asr.transcribe
```

Machine-readable mode:

```bash
luma-cli --json tools describe tts.synthesize
```

Example response shape:

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

### Recommended Agent Flow

1. Run `luma-cli tools list` to identify available atomic capabilities.
2. Run `luma-cli tools describe <tool_id>` before calling a tool for the first time.
3. Load the matching workflow skill from `skills/` when a task requires multiple tools.
4. Prefer explicit file paths and output paths.
5. Use `project create/use` for multi-step media jobs that need organized outputs.

## Atomic Tools

| Tool ID | Command | Risk | Skills |
| --- | --- | --- | --- |
| `asr.transcribe` | `luma-cli asr` | write | `luma-subtitle`, `luma-video-workflow` |
| `tts.synthesize` | `luma-cli tts` | write | `luma-digital-human`, `luma-video-workflow` |
| `lipsync.create` | `luma-cli lipsync` | write | `luma-digital-human`, `luma-video-workflow` |
| `video.enhance` | `luma-cli enhance` | write | `luma-video-workflow` |
| `subtitle.render` | `luma-cli subtitle` | write | `luma-subtitle` |
| `asset.upload` | `luma-cli asset upload` | write | `luma-assets`, `luma-digital-human` |
| `asset.list` | `luma-cli asset list` | read | `luma-assets` |
| `task.status` | `luma-cli task status` | read | `luma-video-workflow` |

The source of truth for these contracts is `shortcuts/`.

## Agent Skills

| Skill | Purpose |
| --- | --- |
| `luma-assets` | Asset discovery, upload conventions, voice/avatar lookup. |
| `luma-digital-human` | TTS + lip-sync workflow guidance for digital-human videos. |
| `luma-subtitle` | Subtitle generation, segmentation, styling, and burn-in workflow. |
| `luma-video-workflow` | End-to-end media workflow composition across ASR, TTS, lip sync, enhance, and tasks. |

Skills contain orchestration guidance. CLI commands should stay atomic.

## Project Workspaces

Projects help agents keep multi-step jobs organized.

```bash
luma-cli project create demo-video
luma-cli project use demo-video
luma-cli project info
```

Project directories:

```text
source/     source media files
audio/      extracted or generated audio
subtitles/  SRT and ASS subtitle files
effects/    effect overlay files
output/     final outputs
tmp/        temporary files
```

## Architecture

```text
main.go                 thin CLI entrypoint
internal/commands/      command routing and CLI adapters
internal/atom/          atomic backend capabilities
internal/cmdutil/       shared argument parsing helpers
internal/config/        local config and credential loading
internal/output/        machine-readable output envelope
shortcuts/              agent tool metadata and discovery
skills/                 agent workflow instructions
project/                local project workspace model
subtitle/               subtitle generation and rendering logic
cloud/                  backend API client helpers
```

Design rule:

```text
skills describe workflows
shortcuts describe callable tools
commands adapt CLI args to atom calls
atoms call backend capabilities
```

## Open Source Boundary

This repository is intended to be open-source client code. It is safe to expose the CLI adapter layer, tool metadata, local project workspace helpers, and agent skills. The hosted backend remains responsible for:

- user registration and account management;
- billing and entitlement checks;
- model execution and task scheduling;
- production prompts and internal workflow policy;
- private asset catalogs and operational tooling.

Do not commit production credentials, card keys, internal model endpoints, database URLs, bucket credentials, or private prompts. Use `LUMA_API_URL` for non-default backend environments.

## Publishing

The release flow follows the same pattern as Lark CLI:

```text
git tag v0.0.1
        ↓
GitHub Actions runs GoReleaser
        ↓
GitHub Release stores platform archives and checksums.txt
        ↓
npm publishes a lightweight installer package
        ↓
npm postinstall downloads the matching Go binary
```

Release requirements:

- GitHub repository permission to create releases.
- npm package access for `@lumageo/luma-cli`, or change `package.json` to a package name you own.
- GitHub Actions secret `NPM_TOKEN`.

Publish `0.0.1`:

```bash
git tag v0.0.1
git push origin v0.0.1
```

The workflow in `.github/workflows/release.yml` will:

1. run `go test ./...`;
2. build `darwin/linux/windows` archives for `amd64/arm64`;
3. upload the archives and `checksums.txt` to GitHub Release;
4. publish the npm package.

Local checks before tagging:

```bash
go test ./...
go build ./...
npm pack --dry-run
```

## Development

```bash
go test ./...
go build ./...
go run . help
go run . tools list
go run . --json tools describe asr.transcribe
```

On Windows with a local Go toolchain:

```powershell
$goRoot = Join-Path $env:USERPROFILE ".local\go\go1.25.5"
$env:GOROOT = $goRoot
$env:Path = (Join-Path $goRoot "bin") + ";" + $env:Path
$env:GOTOOLCHAIN = "local"
go test ./...
go build ./...
```

## Security Notes

`luma-cli` can be invoked by AI agents and may create, upload, download, or modify media assets through the configured backend identity. Treat commands with `risk: write` as side-effecting operations.

Recommended practices:

- Keep card keys private and avoid pasting them into shared logs.
- Use isolated `LUMA_CONFIG_DIR` values for test agents.
- Prefer explicit output paths so generated files are easy to inspect.
- Use `tools describe` before allowing an agent to call an unfamiliar command.
- Keep workflow policy in skills rather than embedding broad automation into command implementations.
