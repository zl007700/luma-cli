---
name: luma-profile-onboarding
description: "Create or improve a Luma profile for original scripts and video workflows, implemented through avatar-persona commands. Use when the user has no avatar_persona_id/profile, has no avatar/voice defaults, wants to set up the account/business persona, or needs onboarding before running luma-original-script or persona-to-video workflows."
metadata:
  category: "capability"
  entrypoint: true
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli avatar-persona create"
  relatedSkills: ["luma-shared", "luma-original-script", "luma-workflow-original-ip-talk", "luma-digital-human"]
---

# Luma Profile Onboarding

Use this skill to create one reusable Luma profile. The user-facing concept is "profile"; the current
implementation stores it as an avatar persona through `luma-cli avatar-persona`.

Read `../luma-shared/SKILL.md` first for common CLI and auth rules.

## Boundary

Use this skill when:

- no `avatar_persona_id` exists
- the user wants to create or improve account/business positioning
- `luma-original-script` or a video workflow needs a persona

Do not write scripts, choose topics, or produce videos here. Stop after saving a usable persona and,
when possible, binding default voice/role assets.

## Minimal Questions

Ask at most two short questions before drafting:

```text
这个账号/角色主要代表什么业务？卖什么产品或服务？目标客户是谁？
内容要帮你解决什么问题，比如获客、信任、转化、复购、招商、招聘？
```

If the user already gave enough context, infer the draft.

## Profile Fields

Use the current avatar-persona schema to store the profile:

- `avatar_name`: short display name
- `role_description`: content positioning, voice, beliefs, conversion intent, and business context
- `audience`: target audience labels or a concise audience description
- optional `voice`: default TTS voice asset id
- optional `role`: default lip-sync role asset id

Because `role_description` currently carries most content guidance, make it concrete:

- who this persona is
- what it sells or represents
- what the audience cares about
- what the persona believes strongly
- what it should avoid
- what business outcome content should support

## Cold Start Rules

New users may have no avatar persona and may not know whether they have usable media assets.

- Content script generation only requires a saved avatar persona. It can run without voice or role assets.
- Full video generation requires a voice and a lip-sync role video.
- Always check `avatar-persona options` before saying the user has no avatar.
- Prefer platform/default voices and roles if available.
- If no suitable role exists, ask the user to upload a local role/avatar video. Do not generate an image and pretend it is a lip-sync avatar.
- If the user provides a video, ask whether it is for voice clone, visual avatar/role, PIP material, ASR/reference, or ordinary processing before uploading.

## Flow

1. Check existing personas:

   ```bash
   luma-cli --json avatar-persona list
   ```

2. Inspect available defaults:

   ```bash
   luma-cli --json avatar-persona options
   ```

   If options are empty or unclear, also inspect the legacy compatibility lists:

   ```bash
   luma-cli asset list voice
   luma-cli asset list roles
   ```

3. Draft one persona. If direction is ambiguous, show at most three concise direction cards and recommend one.

4. Save:

   ```bash
   luma-cli --json avatar-persona create "<avatar_name>" \
     --role-description "<role_description>" \
     --audience "<audience>"
   ```

   If default media assets are available:

   ```bash
   luma-cli --json avatar-persona create "<avatar_name>" \
     --role-description "<role_description>" \
     --audience "<audience>" \
     --voice <voice_asset_id> \
     --role <role_asset_id>
   ```

5. If media defaults are missing, bind them later.

   Voice:

   ```bash
   luma-cli avatar-persona bind-voice <avatar_persona_id> <voice_asset_id> --usage default_tts
   ```

   Role/avatar video:

   ```bash
   luma-cli avatar-persona bind-role <avatar_persona_id> <role_asset_id> --usage default_lipsync
   ```

   If the user provides a local avatar/role video, upload it to `roles` first:

   ```bash
   luma-cli asset upload avatar.mp4 --group roles --name "<friendly_name>"
   luma-cli avatar-persona bind-role <avatar_persona_id> <role_asset_id> --usage default_lipsync
   ```

   If the user provides a voice sample:

   ```bash
   luma-cli voice clone ./voice.wav --name "<friendly_name>"
   luma-cli avatar-persona bind-voice <avatar_persona_id> <voice_asset_id> --usage default_tts
   ```

6. Verify:

   ```bash
   luma-cli --json avatar-persona get <avatar_persona_id>
   ```

   Check `missing_requirements`. If it still lists media requirements, script generation can continue,
   but video generation should wait until the missing assets are bound.

## Handoff

Report:

- `avatar_persona_id`
- avatar name
- one-line positioning summary
- missing requirements, if any
- next script command:

  ```bash
  luma-cli --json content original-script run --avatar-persona <avatar_persona_id> --output runs/<run_id>
  ```

- next video step only if voice and role defaults are present
