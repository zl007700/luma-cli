---
name: luma-core-longform-plan
description: Draft or revise a Luma longform spoken-video plan from a selected topic. Use when a workflow needs agent-authored 03a_longform_plan JSON with an explicit spoken topic reveal, viewer promise, logical section bridges, evidence roles, and revision from plan-review feedback.
---

# Luma Core Longform Plan

## Modes

- `draft`: create a plan from profile and selected topic.
- `revise`: update a prior plan using `plan-review` feedback.

## Inputs

Draft:

- `01_profile.json`
- optional `01_profile_extra.md`
- selected card from `03_topic_selection.json`
- target duration

Revise also requires:

- latest `03a_longform_plan_<topic_id>.json`
- latest `03b_plan_review_<topic_id>.json`

## Output

Write `03a_longform_plan_<topic_id>.json`:

```json
{
  "input": {
    "profile": {},
    "topic_card": {},
    "longform_plan": {
      "plan_id": "longplan_<topic_id>",
      "target_duration_sec": 240,
      "topic": "spoken subject",
      "public_entry": "first spoken hook",
      "topic_reveal": "next spoken sentence naming the subject",
      "viewer_promise": "what the viewer will gain",
      "core_thesis": "one defensible thesis",
      "stance": "creator judgment",
      "audience_filter_turn": "how the opening narrows to the audience",
      "outline": [
        {
          "section": "section label",
          "claim": "one main claim",
          "points": ["supporting point"],
          "bridge_to_next": "why the next section follows",
          "evidence_role": "none"
        }
      ],
      "fact_boundary": ["claim requiring verification or softer wording"]
    }
  },
  "options": {
    "language": "zh-CN"
  }
}
```

Allowed `evidence_role`:

- `none`
- `example`
- `analogy`
- `supporting_evidence`
- `direct_proof`

Omit `bridge_to_next` only on the final section.

## Gate

- hook, topic reveal, and viewer promise work as three consecutive spoken sentences
- topic is understandable without seeing a title card
- each section contains one claim
- every adjacent section has explicit logic
- analogy is never labeled as direct proof
- revision mode addresses every actionable reviewer issue without silently changing the selected
  topic
