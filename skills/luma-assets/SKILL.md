---
name: luma-assets
version: 0.1.0
description: "Manage Luma / 拾光 cloud assets used by generation tools, including voices, avatars, fonts, media inputs, and named groups."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli tools describe asset.upload"
  category: "capability"
  entrypoint: true
  relatedSkills: ["luma-shared", "luma-material", "luma-digital-human"]
  aliases: ["拾光素材", "拾光资产", "拾光工具", "Luma assets", "云端素材", "云端字体"]
---

# Luma Assets

Use this skill when an agent needs to inspect, upload, name, classify, or reuse media assets for Luma workflows.

For local material libraries and PIP material matching, prefer `../luma-material/SKILL.md`.

## Uploaded Video Intent

When the user uploads a video, do not guess its purpose. A video can be a voice sample, an avatar/source-role video, a PIP material, a reference for ASR/rewrite, or a video to process with subtitles/BGM/enhancement.

If the user did not clearly say the purpose, ask one short confirmation before running any destructive or paid workflow:

```text
这段视频准备用来做什么？
1. 提取声音做音色
2. 当视频素材使用
3. 识别文案/仿写
4. 给视频加字幕/处理
```

Only proceed without asking when the wording is explicit:

- Voice clone: "用这个声音", "克隆这个声音", "照这个人的声音", "用视频里的音频".
- Avatar/source role: "用这个人出镜", "这个做数字人", "用这个视频当形象".
- PIP/material: "当素材", "插到视频里", "素材库".
- Reference/ASR/rewrite: "识别这段", "仿写这个", "提取文案".
- Video processing: "加字幕", "加 BGM", "增强", "处理这个视频".

If the user wants voice clone from a video, treat the video as an audio source, not as a digital-human avatar. Extract or upload the audio sample for `voice.clone`; do not upload the video to `roles` unless the user explicitly asks to use the visual person as an avatar/source role.

## Friendly Asset Names

Never expose hash-like object keys as the primary reusable name. Keep object keys internally, but give the user a short display name.

When a new video asset has a hash-like name or no useful filename:

1. Run asset/material understanding when available:
   ```bash
   luma-cli asset understand <object_name_or_key> --group <group> --output asset_meta.json
   ```
2. Generate a natural Chinese display name from the visual/audio summary, 5-10 Chinese characters.
3. Prefer concrete names that describe the person/scene/use, for example `老板在家里`, `女声装修讲解`, `门店口播素材`, `窗帘安装现场`.
4. Upload or save the asset with `--name <display_name>` when the command supports it.
5. Tell the user the display name and keep the object key only as technical detail.

If the generated name is uncertain, ask:

```text
我先把它叫「老板在家里」，可以吗？
```

## Common Groups

- `voice`: voice samples or generated voice assets
- `roles`: digital human avatar videos. Use this group for avatars/source-role videos; `avatar` and `avatars` are conversation aliases, not the canonical command group.
- `asr_input`: uploaded media for transcription
- `enhance_input`: uploaded videos for enhancement
- `lipsync_input`: uploaded audio for lip-sync
- `default`: generic assets

Default assets:

- Users should have common/default assets for voices and digital-human roles.
- `luma-cli asset list roles` is the correct way to inspect available digital-human avatars and should include default/common roles.
- Do not assume default roles are missing because `asset list avatar` fails or returns nothing. Use `asset list roles`.
- Do not create avatar assets from generated images. A digital-human role must be a video asset in `roles`.

## Commands

```bash
luma-cli asset list voice
luma-cli asset list roles
luma-cli asset upload avatar.mp4 --group roles --name 老板在家里
luma-cli asset upload voice.wav --group voice --name 女声装修讲解
luma-cli asset delete upload8393 --group voice
luma-cli voice clone ./sample.wav --name my_voice
luma-cli material group list --output material_groups.json
luma-cli material group describe vlm_ai --output materials.json
luma-cli material search --materials materials.json --query "AI assistant" --limit 5 --output material_matches.json
```

## Agent Rules

- Prefer friendly names from `asset list` when available.
- Prefer stable asset names or IDs returned by CLI commands.
- For digital-human avatars, always inspect `asset list roles`; do not use `asset list avatar`.
- For uploaded videos, confirm intent before choosing voice clone, avatar, material, ASR, or video-processing workflows.
- Do not treat uploaded videos as digital-human avatar/source-role assets unless the user explicitly asks for the visual person to appear.
- Do not upload a user-provided video with only a hash-like name. If no friendly name is available, generate one from understanding or ask the user to confirm a proposed name.
- When uploading reusable assets, pass `--name <display_name>` and report that display name first.
- Use a short display name for reusable assets; do not ask the user to remember hash-like object keys.
- Ask for confirmation before deleting user assets, then use `luma-cli asset delete <name_or_stem> --group <group>`.
- True in-place rename is not available yet. To change a name, re-upload or recreate the asset with `--name <display_name>` and delete the old one after user confirmation.
- Upload local files before referencing them in cloud-only workflows.
- Keep asset upload separate from creative workflow planning.
- For local material libraries, prefer `material group describe` over hand-building a materials file.
- Use `material search` before PIP planning when the script is long or the material group has many candidates.
