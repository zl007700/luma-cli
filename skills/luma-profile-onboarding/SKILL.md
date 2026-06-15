---
name: luma-profile-onboarding
description: "Create or improve a Luma content profile for original scripts and content workflows. Use when the user has no profile_id, wants to set up an account/business positioning, or needs a creator profile before running luma-original-script. This is for content strategy profiles, not avatar-persona digital-human roles."
metadata:
  category: "capability"
  entrypoint: true
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli profile create"
  relatedSkills: ["luma-shared", "luma-original-script", "luma-workflow-original-ip-talk"]
---

# Luma Content Profile Onboarding

Use this skill to create one usable content profile with minimal user burden. A content profile is
the reusable source for original topic selection, writing, review, and history memory.

Read `../luma-shared/SKILL.md` first for common CLI and auth rules.

## Terminology

- `profile`: content account positioning used by `luma-cli profile` and `luma-cli content original-script run`.
- `avatar-persona`: digital-human presentation persona used by `luma-cli avatar-persona`; it binds voices,
  roles, and avatar assets. Do not create or edit avatar personas in this skill.
- `profile_extra.md`: longer business context for content generation.

Avoid saying “人设” when it could mean the visual digital human. Prefer “内容账号定位” or “content profile”.

## Boundary

Use this skill when:

- no current content profile exists
- the user wants to create or improve account positioning
- another content workflow needs a `profile_id`

Do not choose topics, write scripts, find materials, make videos, bind voices, or configure digital-human avatars here.
Stop after saving and selecting a usable content profile.

## Minimal Questions

Ask for business facts, not strategy jargon. Use at most two short questions before drafting:

```text
你现在主要做什么业务？卖什么产品或服务？目标客户大概是谁？
现在希望内容帮你解决什么问题，比如获客、信任、转化、复购、招商、招聘？
```

If the user already gave enough context, do not ask again. Infer a draft.

## Profile Fields

Save only these structured fields:

- `id`: stable ASCII slug, for example `ai_saas_agent_founder`
- `identity`: one-sentence creator/business identity
- `audience`: 2-4 audience labels
- `stance`: 3-5 strong beliefs or content judgments
- `avoid`: 3-5 concrete things the account should avoid

Put softer context into `profile_extra.md`:

- business background
- product/service details
- style and voice
- conversion intent
- likely audience pains
- examples of topics to pursue or avoid

## Flow

1. Check current profile:

   ```bash
   luma-cli --json profile current
   ```

   If one exists, keep it unless the user explicitly wants to improve or replace it.

2. Ask the minimal questions if needed.

3. Draft one recommended profile. If the direction is ambiguous, show at most three concise direction cards and recommend one.

4. Confirm lightly:

   ```text
   我会按这个内容账号定位保存，可以吗？你也可以让我改得更实战、更专业或更接地气。
   ```

5. Save and select:

   ```bash
   luma-cli profile create <profile_id> \
     --identity "<identity>" \
     --audience "<a,b,c>" \
     --stance "<a,b,c>" \
     --avoid "<a,b,c>" \
     --extra-file profile_extra.md \
     --use
   ```

   If the id already exists, ask before overwriting unless the user explicitly requested an update.
   For updates:

   ```bash
   luma-cli profile update <profile_id> \
     --identity "<identity>" \
     --audience "<a,b,c>" \
     --stance "<a,b,c>" \
     --avoid "<a,b,c>" \
     --extra-file profile_extra.md

   luma-cli profile use <profile_id>
   ```

6. Verify:

   ```bash
   luma-cli --json profile get <profile_id>
   ```

## Quality Bar

A usable content profile should make the next workflow clear on:

- what the account/business is
- who the first 20 videos should attract
- what the account believes strongly
- what tones or topics would damage trust
- what business outcome content should support

If those answers are vague, revise before saving.

## Handoff

Report:

- `profile_id`
- whether it is now the current profile
- one-line content positioning summary
- next command:

  ```bash
  luma-cli --json content original-script run --profile <profile_id> --output runs/<run_id>
  ```
