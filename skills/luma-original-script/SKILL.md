---
name: luma-original-script
description: "Run the Luma original video script pipeline from a creator profile. Use when the user asks for an original spoken short-video script, topic-to-script content production, profile-based copywriting, or wants a reviewed final.md from Luma memory, research, topic refinement, writing, and final review."
metadata:
  category: "workflow"
  entrypoint: true
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli content original-script run"
  relatedSkills: ["luma-shared", "luma-profile-onboarding"]
---

# Luma Original Script

Use this skill as glue only. Do not choose topics, run ad hoc research, write hooks, or review the
article inside the agent. The production logic lives in the CLI/backend pipeline.

Read `../luma-shared/SKILL.md` first for auth, output, and safe CLI rules.

## Boundary

This skill owns:

```text
profile_id -> original-script pipeline -> final.md + final_review.json + run_state.json
```

It does not produce TTS, digital-human video, PPT visuals, subtitles, or covers. Use
`luma-workflow-original-ip-talk` after this skill when the user wants a complete video.

## Required Input

- `profile_id`
- optional `topic_hint`
- optional output directory

If no usable profile exists, use `luma-profile-onboarding` first.

## Run

Prefer JSON output:

```bash
luma-cli --json content original-script run \
  --profile <profile_id> \
  --output runs/<run_id>
```

With a user-provided direction:

```bash
luma-cli --json content original-script run \
  --profile <profile_id> \
  --topic-hint "<topic or direction>" \
  --output runs/<run_id>
```

## Output Contract

Always inspect and report these files:

- `final.md`: the produced spoken script. This exists whenever the workflow status is `done`.
- `final_review.json`: final reviewer result and route.
- `run_state.json`: stage usage, warnings, cloud artifact ids, and promotion status.

Useful audit files:

- `03_research_rounds.json`: web/social search results.
- `04_detail_expansion_plan.json`: selected sources for detail expansion.
- `05_expanded_details.json`: URL reads and video ASR attempts, including failures.
- `06_topic_selection.json`: topic refinement result.
- `07_article_v1.md`: first draft before any rewrite.

## Status Semantics

`run_state.status` describes whether the pipeline produced an output:

- `done`: the workflow completed and `final.md` is the best available script.
- `failed`: a system or network boundary prevented script production.

`run_state.promotion.status` describes whether the script passed the publication/history gate:

- `promoted`: the script passed final review and was written to cloud history.
- `blocked`: the script was produced but not promoted. This is not a workflow failure.

When `promotion.status` is `blocked`, still show the user `final.md`. Also summarize the reviewer
reason from `run_state.promotion.reason` and `final_review.json`.

## Agent Rules

- Do not rewrite `final.md` locally unless the user explicitly asks for manual editing.
- Do not call separate search/write/review commands to replace the pipeline.
- Do not treat `promotion.status=blocked` as failure. It means “produced, but not auto-promoted.”
- Keep intermediate files. They are the audit trail for research, detail expansion, and review.
- Never expose API keys, card keys, backend tokens, or private object credentials.
