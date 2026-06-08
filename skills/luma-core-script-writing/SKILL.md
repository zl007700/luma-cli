---
name: luma-core-script-writing
description: Draft or revise a Luma spoken-video script locally from an approved plan and material artifacts. Use when a workflow needs agent-authored script JSON, accurate evidence roles, ready material IDs, explicit section transitions, and a complete script-review payload without calling backend script.write.
---

# Luma Core Script Writing

Do not call backend `script.write`.

## Modes

- `draft`: write from approved plan and materials.
- `revise`: update the latest script using `script-review` feedback.

## Inputs

- `01_profile.json`
- optional `01_profile_extra.md`
- `03_topic_selection.json`
- accepted longform plan and plan review
- material plan, material assets, and deliverables manifest
- target duration

Revision also requires the latest script and script review.

## Step 1: Build Writing Brief

Run:

```bash
node <skill_dir>/scripts/build_script_writer_payload.js \
  --profile 01_profile.json \
  --profile-extra 01_profile_extra.md \
  --topic-selection 03_topic_selection.json \
  --topic-id <topic_id> \
  --longform-plan 03a_longform_plan_<topic_id>.json \
  --plan-review 03b_plan_review_<topic_id>.json \
  --material-plan 04_material_plan_<topic_id>.json \
  --material-assets 05_material_assets_<topic_id>.json \
  --deliverables materials/<topic_id>/final_assets/deliverables_manifest.json \
  --duration-sec <seconds> \
  --output 06_script_writer_payload_<topic_id>.json
```

## Step 2: Write Script

Write `07_script_<topic_id>.json`:

```json
{
  "topic_id": "<topic_id>",
  "title": "internal and cover title",
  "hook": "first spoken hook",
  "topic_reveal": "spoken subject sentence",
  "viewer_promise": "spoken payoff sentence",
  "sections": [
    {
      "section": "section label",
      "spoken_text": "complete spoken copy including transition",
      "claim_ids": ["claim_001"],
      "material_asset_ids": ["asset_001"],
      "evidence_role": "supporting_evidence",
      "bridge_to_next": "spoken transition logic",
      "visual_intent": "what the audience should see"
    }
  ],
  "full_script": "complete concatenated spoken copy",
  "evidence_notes": [],
  "risk_notes": [],
  "estimated_duration_sec": 240
}
```

Opening order:

```text
public_entry -> topic_reveal -> viewer_promise -> audience_filter_turn
```

## Step 3: Build Review Payload

Write `08_script_review_payload_<topic_id>.json`:

```json
{
  "input": {
    "profile": {},
    "topic_card": {},
    "longform_plan": {},
    "plan_review": {},
    "material_assets": {},
    "script": {}
  },
  "options": {
    "language": "zh-CN"
  }
}
```

## Gate

- script is understandable with the screen off
- `full_script` includes topic reveal and viewer promise
- every non-final section has an explicit bridge
- evidence roles are accurate
- `material_asset_ids` contains only ready deliverables
- revision mode addresses every actionable reviewer issue while preserving the approved thesis

## Outputs

- `06_script_writer_payload_<topic_id>.json`
- `07_script_<topic_id>.json`
- `08_script_review_payload_<topic_id>.json`
