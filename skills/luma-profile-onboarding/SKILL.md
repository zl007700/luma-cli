---
name: luma-profile-onboarding
description: Guide a low-friction Luma profile onboarding flow when the user has no profile or wants to create/improve one; infer a usable creator profile from simple business facts, confirm with lightweight choices, and save it with luma-cli profile commands.
metadata:
  relatedSkills: ["luma-shared", "luma-original-script", "luma-workflow-original-ip-talk"]
---

# Luma Profile Onboarding

Read `../luma-shared/SKILL.md` first. This skill creates one usable Luma profile with the least
possible user burden. The profile is the reusable source for topic discovery, plan review, script
writing, and original IP video workflows.

## Boundary

Use this skill when:

- no current profile exists
- the user asks to create a profile, build an account persona, or start content from scratch
- another Luma workflow needs a profile before it can continue

Do not choose topics, write scripts, find materials, or make videos here. Stop after saving and
selecting a usable profile.

## Principle

Ask for business facts, not positioning language.

The user should not have to answer "who do you help solve what problem?" or fill a strategy form.
Ask what they already know:

```text
你是做什么行业的？主要卖什么产品或服务？做了多久？
随便说几句就行，不用整理。
```

If useful, add one optional sentence:

```text
现在最想通过内容解决什么问题？比如获客、信任、转化、复购、招商、招人。
```

Never ask more than two questions before drafting. Infer the rest.

## Required Profile Fields

The saved profile must contain only these structured fields:

- `id`: short stable slug, ASCII when possible, e.g. `local_beauty_owner`
- `identity`: creator identity in one sentence
- `audience`: 2-4 audience labels
- `stance`: 3-5 strong beliefs or content judgments
- `avoid`: 3-5 things the account should avoid

Put all softer context into `profile_extra.md`, not extra JSON fields:

- business background inferred from the user's words
- content style and voice
- product/service details
- likely audience pains
- conversion intent
- examples of topics the account should and should not talk about

## Onboarding Flow

1. Check the current profile.

   ```bash
   luma-cli --json profile current
   ```

   If a current profile exists, ask whether to improve it or create a new one only when the user
   explicitly wants onboarding. Otherwise keep the current profile.

2. Ask for the minimal facts.

   Use the exact short prompt from `Principle`. Let the user answer messily.

3. Infer three direction cards.

   Each card should be concrete and selectable:

   ```text
   A. 老板实战型：讲自己怎么做、怎么踩坑、怎么拿结果
   B. 顾问分析型：帮同行看清问题、判断方案、少花冤枉钱
   C. 用户视角型：从消费者真实体验切入，再带到专业判断
   ```

   Adapt the card labels to the industry. Do not present many choices. If one direction is clearly
   best, recommend it and let the user confirm or change.

4. Draft the profile.

   Create:

   - a short `profile_id`
   - one `identity`
   - concise `audience`, `stance`, and `avoid` lists
   - `profile_extra.md`

   Use strong, useful beliefs. Avoid generic identities such as "行业知识分享者" unless the user truly
   has no business context.

5. Confirm with one lightweight choice.

   Show only the meaningful draft fields and ask:

   ```text
   这个方向可以保存吗？你也可以让我改得更实战 / 更专业 / 更接地气。
   ```

   If the user does not object, save it.

6. Save and select the profile.

   Write `profile_extra.md` in the active project/workspace, then run:

   ```bash
   luma-cli profile create <profile_id> --identity "<identity>" --audience "<a,b,c>" --stance "<a,b,c>" --avoid "<a,b,c>" --extra-file profile_extra.md --use
   ```

   If the profile id already exists, ask before overwriting unless the user explicitly requested an
   update. For updates, use:

   ```bash
   luma-cli profile update <profile_id> --identity "<identity>" --audience "<a,b,c>" --stance "<a,b,c>" --avoid "<a,b,c>" --extra-file profile_extra.md
   luma-cli profile use <profile_id>
   ```

7. Verify.

   ```bash
   luma-cli --json profile get <profile_id>
   ```

   Confirm that `identity`, `audience`, `stance`, `avoid`, and `extra` are present.

## Quality Bar

A usable profile should make the next agent able to answer:

- this account should sound like whom
- who the first 20 videos are trying to attract
- what the account believes strongly
- what topics or tones would damage the account
- what business outcome content should eventually support

If those answers are still vague, revise the profile before saving.

## Handoff

After saving, report:

- profile id
- current profile status
- one-line positioning summary
- suggested next skill: `luma-original-script` for a script, or `luma-workflow-original-ip-talk` for a full PPT + digital-human video
