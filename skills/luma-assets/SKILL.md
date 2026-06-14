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

## Asset Registry V2

Use Asset Registry V2 for reusable cloud assets that an Agent chooses semantically: templates, fonts, BGM, SFX, avatars, voices, persona images, material images, and material videos.

Prefer system assets when selecting platform defaults:

```bash
luma-cli assets groups --kind template --scope system
luma-cli assets search --kind template --group hook_portrait --scope system --limit 8
luma-cli assets search --kind bgm --scope system --limit 8
luma-cli assets search --kind sfx --scope system --limit 8
luma-cli assets search --kind font --scope system --limit 8
luma-cli assets resolve <asset_id>
```

Use the returned `asset_id` as the stable reference. Read `GROUP`, `NAME`, `PROBE`, and `CAPTION` before choosing. For templates, inspect `metadata.manifest.format_support`, `metadata.manifest.agent_should_fill`, `metadata.semantic.use_when`, and `metadata.semantic.avoid_when`.

Do not sign or cache Remotion template assets as a normal client. Template source is private and only render workers can request it.

## Common Groups

- `voice`: voice samples or generated voice assets
- `roles`: digital human avatar videos. Use this group for avatars/source-role videos; `avatar` and `avatars` are conversation aliases, not the canonical command group.
- `asr_input`: uploaded media for transcription
- `enhance_input`: uploaded videos for enhancement
- `lipsync_input`: uploaded audio for lip-sync
- `default`: generic assets

Default assets:

- Registry V2 system assets use `scope=system`; user uploads use `scope=user`.
- Template groups include shape and purpose, for example `hook_portrait`, `hook_landscape`, `metrics_landscape`, `subtitles_portrait`.
- For the legacy digital-human lip-sync path, `luma-cli asset list roles` is still the compatibility command for available avatar videos.
- Do not create digital-human avatar assets from generated images. A digital-human role must be a video asset in `roles`.

## Commands

```bash
luma-cli assets groups --kind template --scope system
luma-cli assets search --kind template --group hook_portrait --scope system --limit 8
luma-cli assets search --kind bgm --scope system --limit 8
luma-cli assets upload image.png --kind material_image --group references --name 门店外景
luma-cli assets delete <asset_id>
luma-cli voice clone ./sample.wav --name my_voice
luma-cli asset list roles
luma-cli material group list --output material_groups.json
luma-cli material group describe vlm_ai --output materials.json
luma-cli material search --materials materials.json --query "AI assistant" --limit 5 --output material_matches.json
```

## Agent Rules

- Prefer Asset Registry V2 (`luma-cli assets ...`) for cloud assets that will be chosen by an Agent.
- Prefer stable `asset_id` values returned by `assets search` or `assets resolve`.
- Use `--scope system` when selecting platform default assets so user uploads and smoke-test leftovers do not affect the choice.
- For digital-human avatars in the legacy lip-sync path, inspect `asset list roles`; do not use `asset list avatar`.
- For uploaded videos, confirm intent before choosing voice clone, avatar, material, ASR, or video-processing workflows.
- Do not treat uploaded videos as digital-human avatar/source-role assets unless the user explicitly asks for the visual person to appear.
- Do not upload a user-provided video with only a hash-like name. If no friendly name is available, generate one from understanding or ask the user to confirm a proposed name.
- When uploading reusable assets, pass `--name <display_name>` and report that display name first.
- Use a short display name for reusable assets; do not ask the user to remember hash-like object keys.
- Ask for confirmation before deleting user assets, then use `luma-cli assets delete <asset_id>` for Registry V2 assets.
- True in-place rename is not available yet. To change a name, re-upload or recreate the asset with `--name <display_name>` and delete the old one after user confirmation.
- Upload local files before referencing them in cloud-only workflows.
- Keep asset upload separate from creative workflow planning.
- For local material libraries, prefer `material group describe` over hand-building a materials file.
- Use `material search` before PIP planning when the script is long or the material group has many candidates.
