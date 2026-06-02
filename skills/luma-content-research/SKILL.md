---
name: luma-content-research
version: 0.1.0
description: "Use when a Luma / 拾光 / 拾光智能体 / 拾光工具 agent needs content research, topic discovery, keyword tables, persona-based search, or Excel-friendly research outputs for short-video planning."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli tools describe research.run"
  category: "capability"
  entrypoint: true
  aliases: ["拾光选题", "拾光调研", "拾光工具", "Luma research", "内容调研", "选题"]
  relatedSkills: ["luma-shared", "luma-workflow-viral-remix"]
---

# Luma Content Research

Use this skill when the user wants to find short-video topics, analyze references, export search results, or prepare a keyword table for later script writing with Luma / 拾光.

Read `../luma-shared/SKILL.md` first for common auth, project, and artifact rules.

## When To Use

- The user wants topic ideas, competitor references, or viral angles.
- The user provides a role/persona and asks what content to make.
- The user asks for an Excel-style research table.
- A workflow needs `step0_content_research.json`, `step0_content_research.csv`, or `step0_keywords.json`.

Do not use this skill for script rewriting after the topic is already selected. Use script or workflow skills for that step.

## Standard Flow

1. Run backend content research:
   ```bash
   luma-cli research run --role "<role_or_persona_description>" --mode precise --date-range 7d --output step0_content_research.json
   ```

2. Export an Excel-friendly table:
   ```bash
   luma-cli research export --input step0_content_research.json --output step0_content_research.csv
   ```

3. Extract keywords and topic rows:
   ```bash
   luma-cli research keywords --input step0_content_research.json --output step0_keywords.json --csv step0_keywords.csv
   ```

## Persona Reuse

Save a reusable persona when the user describes a stable account or role:

```bash
luma-cli research persona save ai_founder --role "AI tool founder making short videos for operators"
luma-cli research run --persona ai_founder --output step0_content_research.json
```

## Agent Rules

- Keep the raw research JSON even when exporting CSV.
- Use CSV for human inspection; use JSON for agent decisions.
- Pick at most 3 high-potential references or one topic cluster explicitly before moving to rewrite.
- Research results are discovery data, not source scripts. For viral remix workflows, only download and ASR references that are likely to contain reusable spoken copy; skip non-口播 or low-signal videos. See `../luma-workflow-viral-remix/SKILL.md`.
- Do not fabricate metrics that are not present in the research result.
- If research returns too few results, retry with `--mode expanded` or a broader role description.
