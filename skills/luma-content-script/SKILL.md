---
name: luma-content-script
description: Create one original Luma spoken-video script from profile memory: choose a fresh topic, review the plan, find materials, write locally, and submit script review.
metadata:
  relatedSkills: ["luma-shared", "luma-find-material", "luma-workflow-original-ip-talk"]
---

# Luma Content Script

Read `../luma-shared/SKILL.md` first. This is the main skill for creating one original content
script. It starts at topic selection and ends at a reviewed final spoken script. The CLI provides
atomic cloud/resource commands; the agent does the orchestration, judgment, iteration, and writing.

## Ownership

Cloud/TOS owns memory and history:

- profile resources: `profile_<profile_id>/profile.current.json`, `profile_extra.current.md`
- content artifacts: `content_<profile_id>/raw_signals.current.json`, `topic_review.current.json`
- project artifacts: approved longform plan, material plan/assets, final script, review result
- reviewer APIs: `topic.review`, `plan.review`, `script.review`
- search/resource atoms: social search, websearch, image search, screenshot workers, VLM review

Skill/agent owns content intelligence:

- decide what history matters
- aggregate and interpret signals
- select or draft the topic/plan
- revise based on reviewer feedback
- collect and downgrade material evidence
- write the spoken script
- decide when the result is good enough

CLI owns thin operations only:

- upload/download/list cloud resources
- call search and reviewer atoms
- save local project artifact copies
- never become the long-term authority for profile/content memory

## Standard Chain

```text
profile.load
  -> content.history
  -> content.discovery
  -> topic.review
  -> plan.draft
  -> plan.review
  -> material.plan
  -> material.collect
  -> script.write.local
  -> script.review
  -> content.artifact.upload
```

Use these step names in artifact metadata, notes, and local filenames. Prefixes mean ownership:

- `content.*`: CLI/cloud resource or search atom.
- `topic.review`, `plan.review`, `script.review`: backend reviewer atom.
- `plan.draft`, `script.write.local`: agent/skill-authored content, no backend writer call.
- `material.plan`, `material.collect`: local skill planning/collection around cloud search/review atoms.

## Credit Planning

These are planning estimates only; backend usage/metering is the billing source of truth.

| Step | Default estimate |
| --- | ---: |
| `profile.load` / `content.history` / `content.artifact.upload` | 0 credits |
| `content.search.social` / `content.search.social-account` / `content.search.websearch` | 5 credits |
| `content.search.image` | 8 credits per query |
| `topic.review` / `plan.review` / `script.review` with `basic_model` | 5 credits |
| `topic.review` / `plan.review` / `script.review` with `pro_model` | 80 credits |
| `material.review` | 5 credits per asset |
| `plan.draft` / `script.write.local` | 0 backend credits |

Do not mention provider/model names to end users. Use only `basic_model` and `pro_model`.

## Required Files

Use stable names in the project workspace:

- `01_profile.json`
- `01_profile_extra.md`
- `02_raw_signals.json`
- `03_topic_review.json`
- `03a_longform_plan_<topic_id>.json`
- `03b_plan_review_<topic_id>.json`
- `04_material_plan_<topic_id>.json`
- `05_material_assets_<topic_id>.json`
- `06_script_writer_payload_<topic_id>.json`
- `07_script_<topic_id>.json`
- `08_script_review_payload_<topic_id>.json`
- `09_script_review_<topic_id>.json`

Upload durable JSON/MD artifacts back to TOS through CLI resource helpers whenever a command supports
cloud artifact persistence. Keep local copies because downstream video production works from the
project directory.

## Procedure

1. Load profile:

   ```bash
   luma-cli --json profile get <profile_id>
   luma-cli --json content history --profile <profile_id>
   ```

2. Build the used-topic set before discovering new topics. Treat any historical artifact with one of
   these signals as already used:

   - `meta.topic_id`
   - `meta.topic_title`
   - `meta.content_fingerprint`
   - a downloadable `script`, `script_review`, `topic_review`, or `longform_plan` artifact whose JSON
     contains the same topic title, topic id, core thesis, or substantially equivalent angle

   Do not reuse an already used topic or near-duplicate angle unless the user explicitly asks to
   rerun that topic. If history is unavailable, say so and prefer fresher signals instead of
   reusing local old test artifacts.

3. Discover or select a topic using search atoms and agent judgment. Submit promising topic cards to
   `topic.review` when ranking needs protected reviewer judgment.

4. Draft a compact `longform_plan` with only the required fields from `luma-find-material`.
   Mark this local step as `plan.draft`.
   Submit it to:

   ```bash
   luma-cli agent plan-review --input 03a_longform_plan_<topic_id>.json --output 03b_plan_review_<topic_id>.json --model basic_model
   ```

   Use `pro_model` only when the user asks or the gate is ambiguous.

5. Run `material.plan` / `material.collect` through `luma-find-material`. Do not let material collection
   change the topic.

6. Write the script directly as the agent. Mark this local step as `script.write.local`. Preserve
   `longform_plan.public_entry` as the opening direction unless reviewer feedback explicitly requires
   changing it.

7. Submit script review:

   ```bash
   luma-cli agent script-review --input 08_script_review_payload_<topic_id>.json --output 09_script_review_<topic_id>.json --model basic_model
   ```

8. If review says `revise` or `major_revise`, revise the agent-written script and resubmit. Do not
   call backend `script.write` as fallback.

9. Save final durable artifacts to cloud history with topic metadata:

   ```bash
   luma-cli content artifact upload --profile <profile_id> --type script --input 07_script_<topic_id>.json --name script.current.json --topic-id <topic_id> --topic-title "<title>"
   luma-cli content artifact upload --profile <profile_id> --type script_review --input 09_script_review_<topic_id>.json --name script_review.current.json --topic-id <topic_id> --topic-title "<title>"
   ```

   This is what prevents the next workflow run from selecting the same topic again.

## Quality Gates

- Public entry must make ordinary strangers want to stay before narrowing to the target audience.
- Plan/script review gates are not averages; opening survival caps the decision.
- Material evidence constrains facts. Unsupported claims must be softened or removed.
- Final script must be complete spoken copy, not an outline.
