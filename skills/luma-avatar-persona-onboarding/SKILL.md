---
name: luma-avatar-persona-onboarding
description: "Create or improve a Luma avatar persona for original scripts and video workflows. Use when the user has no avatar_persona_id, wants to set up the account/business persona, or needs a persona before running luma-original-script. This replaces legacy profile onboarding."
metadata:
  category: "capability"
  entrypoint: true
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli avatar-persona create"
  relatedSkills: ["luma-shared", "luma-original-script", "luma-workflow-original-ip-talk", "luma-digital-human"]
---

# Luma Avatar Persona Onboarding

Use this skill to create one reusable avatar persona. In the new Luma content flow, the avatar
persona is the durable subject for both content strategy and media defaults.

Read `../luma-shared/SKILL.md` first for common CLI and auth rules.

## Boundary

Use this skill when:

- no `avatar_persona_id` exists
- the user wants to create or improve account/business positioning
- `luma-original-script` or a video workflow needs a persona

Do not write scripts, choose topics, or produce videos here. Stop after saving a usable persona.

## Minimal Questions

Ask at most two short questions before drafting:

```text
这个账号/角色主要代表什么业务？卖什么产品或服务？目标客户是谁？
内容要帮你解决什么问题，比如获客、信任、转化、复购、招商、招聘？
```

If the user already gave enough context, infer the draft.

## Persona Fields

Use the current avatar-persona schema:

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

## Flow

1. Check existing personas:

   ```bash
   luma-cli --json avatar-persona list
   ```

2. If assets are needed, inspect options:

   ```bash
   luma-cli --json avatar-persona options
   ```

3. Draft one persona. If direction is ambiguous, show at most three concise direction cards and recommend one.

4. Save:

   ```bash
   luma-cli --json avatar-persona create "<avatar_name>" \
     --role-description "<role_description>" \
     --audience "<audience>"
   ```

   If default media assets are known:

   ```bash
   luma-cli --json avatar-persona create "<avatar_name>" \
     --role-description "<role_description>" \
     --audience "<audience>" \
     --voice <voice_asset_id> \
     --role <role_asset_id>
   ```

5. Verify:

   ```bash
   luma-cli --json avatar-persona get <avatar_persona_id>
   ```

## Handoff

Report:

- `avatar_persona_id`
- avatar name
- one-line positioning summary
- missing requirements, if any
- next command:

  ```bash
  luma-cli --json content original-script run --avatar-persona <avatar_persona_id> --output runs/<run_id>
  ```
