---
name: luma-content-script
description: Orchestrate the Luma original profile-to-script workflow as pure glue. Use when an agent must call topic discovery, local topic selection, longform planning, plan review, material collection, local script writing, script review, and artifact upload in the required order.
---

# Luma Content Script Workflow

This skill is glue only. Do not perform topic selection, plan writing, material reasoning, or script
writing inside this file. Delegate those steps to the named core skills.

## Inputs

- active Luma project
- `<profile_id>`
- target duration in seconds

## Flow

### Step 1: Load Profile

Run:

```bash
luma-cli --json profile get <profile_id>
luma-cli --json content history --profile <profile_id>
```

Save:

- profile data -> `01_profile.json`
- profile extra text -> `01_profile_extra.md`
- content history -> `02_content_history.json`

If the profile is unusable, call `luma-profile-onboarding`, then repeat Step 1.

### Step 2: Discover Topic Signals

Call `luma-core-topic-discovery`.

Inputs:

- `01_profile.json`
- `01_profile_extra.md`
- `02_content_history.json`

Required output:

- `02_raw_signals.json`

Do not continue unless that skill's discovery gate passes.

### Step 3: Select One Topic

Call `luma-core-topic-selection`.

Inputs:

- `01_profile.json`
- `01_profile_extra.md`
- `02_content_history.json`
- `02_raw_signals.json`

Required output:

- `03_topic_selection.json`

Read the selected `topic_id` and use it in every later filename.

### Step 4: Create Longform Plan

Call `luma-core-longform-plan` in `draft` mode.

Inputs:

- `01_profile.json`
- `01_profile_extra.md`
- selected card from `03_topic_selection.json`
- target duration

Required output:

- `03a_longform_plan_<topic_id>.json`

### Step 5: Review Longform Plan

Run:

```bash
luma-cli agent plan-review \
  --input 03a_longform_plan_<topic_id>.json \
  --model basic_model \
  --output 03b_plan_review_<topic_id>.json
```

If the decision is not accepted, call `luma-core-longform-plan` in `revise` mode with:

- latest plan
- latest plan review

Save versioned files and repeat Step 5. Stop after three failed review cycles and report the repeated
blocker.

### Step 6: Find Materials

Call `luma-find-material`.

Inputs:

- `03_topic_selection.json`
- accepted `03a_longform_plan_<topic_id>.json`
- accepted `03b_plan_review_<topic_id>.json`

Required outputs:

- `04_material_plan_<topic_id>.json`
- `05_material_assets_<topic_id>.json`
- `materials/<topic_id>/final_assets/deliverables_manifest.json`

Do not continue with empty deliverables.

### Step 7: Write Script

Call `luma-core-script-writing` in `draft` mode.

Inputs:

- `01_profile.json`
- `01_profile_extra.md`
- `03_topic_selection.json`
- accepted longform plan and plan review
- material plan, material assets, and deliverables manifest
- target duration

Required outputs:

- `06_script_writer_payload_<topic_id>.json`
- `07_script_<topic_id>.json`
- `08_script_review_payload_<topic_id>.json`

### Step 8: Review Script

Run:

```bash
luma-cli agent script-review \
  --input 08_script_review_payload_<topic_id>.json \
  --model basic_model \
  --output 09_script_review_<topic_id>.json
```

If the decision is not accepted, call `luma-core-script-writing` in `revise` mode with:

- latest script
- latest script review
- the same approved plan and material inputs

Save versioned files and repeat Step 8. Stop after three failed review cycles and report the repeated
blocker.

### Step 9: Upload History

Run:

```bash
luma-cli content artifact upload \
  --profile <profile_id> \
  --type longform_plan \
  --input 03a_longform_plan_<topic_id>.json \
  --name longform_plan.current.json \
  --topic-id <topic_id> \
  --topic-title "<title>"

luma-cli content artifact upload \
  --profile <profile_id> \
  --type script \
  --input 07_script_<topic_id>.json \
  --name script.current.json \
  --topic-id <topic_id> \
  --topic-title "<title>"

luma-cli content artifact upload \
  --profile <profile_id> \
  --type script_review \
  --input 09_script_review_<topic_id>.json \
  --name script_review.current.json \
  --topic-id <topic_id> \
  --topic-title "<title>"
```

Verify:

```bash
luma-cli --json content history --profile <profile_id>
```

## Output Contract

Return these files to the caller:

- accepted `07_script_<topic_id>.json`
- accepted `09_script_review_<topic_id>.json`
- `05_material_assets_<topic_id>.json`
- deliverables manifest
- selected topic ID and title

Do not start TTS or video production before all five outputs exist.
