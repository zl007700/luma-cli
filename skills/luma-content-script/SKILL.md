---
name: luma-content-script
description: Run the complete Luma original spoken-script workflow as a strict step-by-step SOP. Use when an agent must load a creator profile, inspect content history, mine Douyin and web topic signals, select a topic locally, draft and review a longform plan, collect materials, write the script locally, review it, and upload final artifacts.
---

# Luma Content Script SOP

Follow every checkpoint in order. Do not replace a named command with a similar-looking capability.
All required commands, schemas, gates, and failure branches are contained in this file.

## Non-Negotiable Route

```text
profile get
-> content history
-> content topic mine
-> select topic locally
-> write longform plan locally
-> agent plan-review
-> luma-find-material plan and collect
-> build local writing brief
-> write script locally
-> agent script-review
-> revise until accepted
-> content artifact upload
```

`research.run` is not a substitute for `content topic mine`. It may only suggest additional search
language. Do not call backend `script.write`; this skill requires agent-authored local writing.

## Files

Create these files in the active project:

```text
01_profile.json
01_profile_extra.md
02_content_history.json
02_raw_signals.json
03_topic_selection.json
03a_longform_plan_<topic_id>.json
03b_plan_review_<topic_id>.json
04_material_plan_<topic_id>.json
05_material_assets_<topic_id>.json
materials/<topic_id>/final_assets/deliverables_manifest.json
06_script_writer_payload_<topic_id>.json
07_script_<topic_id>.json
08_script_review_payload_<topic_id>.json
09_script_review_<topic_id>.json
```

Never overwrite an accepted artifact while revising. Add `_v2`, `_v3`, and so on, then promote the
accepted version to the stable filename.

## Checkpoint 0: Prepare

1. Confirm authentication and active project:

   ```bash
   luma-cli auth status
   luma-cli project artifact list
   ```

2. If no project is active:

   ```bash
   luma-cli project create <project_name>
   luma-cli project use <project_name>
   ```

3. Resolve the profile:

   ```bash
   luma-cli --json profile current
   ```

4. If no usable profile exists, stop this SOP and run `luma-profile-onboarding`.

Pass condition: one active project and one explicit `<profile_id>`.

## Checkpoint 1: Save Profile Snapshot

1. Run:

   ```bash
   luma-cli --json profile get <profile_id>
   ```

2. Save the JSON response `data` object, without the CLI envelope, as `01_profile.json`.
3. Save `data.extra` as `01_profile_extra.md`. Keep the remaining profile fields in
   `01_profile.json`.
4. Verify that identity, audience, and stance are non-empty. If any is empty, update the profile
   before continuing.

Pass condition: the agent can state who is speaking, to whom, and what judgments this creator owns.

## Checkpoint 2: Inspect Used Topics

1. Run and save the JSON result as `02_content_history.json`:

   ```bash
   luma-cli --json content history --profile <profile_id>
   ```

2. Build a used-topic set from:

   - `meta.topic_id`
   - `meta.topic_title`
   - `meta.content_fingerprint`
   - downloadable script, script-review, topic-selection, and longform-plan artifacts

3. Treat equivalent thesis and angle combinations as duplicates even when titles differ.

Pass condition: write a short local list of excluded topic IDs, titles, and near-duplicate angles.
If history cannot be loaded, record that limitation and continue with stricter freshness checks.

## Checkpoint 3: Build Search Matrix

Derive non-duplicate searches from the profile and used-topic set:

| Dimension | Required |
| --- | ---: |
| category terms | 2 |
| audience pains or objections | 2 |
| workflow or use-case terms | 2 |
| decision, misconception, or contrarian terms | 1-2 |
| current product, policy, or market terms | 1-2 when relevant |

Produce:

- 6-10 concise Douyin/social keywords
- 3-5 natural-language web queries

Do not use one broad phrase for every query. Do not search only the topic the agent already wants to
write.

Optional recovery only:

```bash
luma-cli research run \
  --role "<profile identity, audience, and domain>" \
  --mode expanded \
  --date-range 7d \
  --output keyword_expansion.json
```

Use useful phrases from that result in the matrix, then continue to `content topic mine`.

Pass condition: the matrix covers at least three distinct intent dimensions.

## Checkpoint 4: Mine Canonical Raw Signals

Run exactly this capability:

```bash
luma-cli content topic mine \
  --profile <profile_id> \
  --social-keywords "<6-10 comma-separated keywords>" \
  --web-queries "<3-5 comma-separated queries>" \
  --date-range 7d \
  --limit-per-keyword 20 \
  --web-num 6 \
  --max-raw 200 \
  --output 02_raw_signals.json
```

Validate:

```bash
node <skill_dir>/scripts/validate_artifact.js \
  --type discovery \
  --input 02_raw_signals.json
```

Required:

- `raw_signals` contains at least 15 unique signals; target 30 or more
- `counts.social_raw > 0`
- `counts.web_raw > 0`
- at least 8 signals remain plausibly relevant after obvious ads, generic traffic bait, and
  off-profile items are removed

Failure branch:

1. Identify which query dimension failed.
2. Replace weak queries; do not merely add synonyms.
3. Rerun `content topic mine`.
4. Do not manually wrap `research.run` output as `02_raw_signals.json`.

## Checkpoint 5: Select Topic Locally

There is no topic reviewer in this workflow. Read `02_raw_signals.json`, cluster related signals,
compare them with profile/history, and choose the topic as the agent.

Score candidate clusters on:

- audience relevance
- creator stance fit
- freshness
- conflict or misconception
- material availability
- ability to support a complete argument rather than one isolated headline

Reject a candidate when:

- it duplicates content history
- its evidence is only generic engagement
- its public entry does not lead naturally to the actual topic
- it has no creator-specific stance
- available materials cannot support its factual claims

Write `03_topic_selection.json` yourself:

```json
{
  "topic_cards": [
    {
      "topic_id": "topic_001",
      "status": "selected",
      "title": "internal topic title",
      "theme": "topic cluster",
      "angle": "specific angle",
      "public_entry": "candidate first spoken hook",
      "core_opinion": "creator-specific thesis",
      "common_misunderstanding": "what the audience usually gets wrong",
      "audience_value": "why the audience should care",
      "why_selected": "comparison-based selection reason",
      "evidence_signals": [
        {
          "source": "social or websearch",
          "title": "signal title",
          "url": "source URL",
          "author_name": "optional",
          "published_at": "optional"
        }
      ],
      "rejected_alternatives": [
        {
          "title": "candidate title",
          "reason": "duplicate, weak evidence, low fit, or weak argument"
        }
      ]
    }
  ]
}
```

Keep exactly one `status=selected` card. Include at least three evidence signals spanning social and
web sources. Record at least two rejected alternatives so selection is comparative rather than
arbitrary.

Validate:

```bash
node <skill_dir>/scripts/validate_artifact.js \
  --type topic-selection \
  --input 03_topic_selection.json
```

Pass condition: the selected topic has a clear hook direction, thesis, audience value, evidence
path, and no near-duplicate in history.

## Checkpoint 6: Draft Longform Plan Locally

Create `03a_longform_plan_<topic_id>.json` yourself:

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
      "topic_reveal": "second spoken sentence naming the subject",
      "viewer_promise": "what the viewer will gain",
      "core_thesis": "one defensible thesis",
      "stance": "creator judgment",
      "audience_filter_turn": "how the opening narrows to the target audience",
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

Allowed `evidence_role`: `none`, `example`, `analogy`, `supporting_evidence`, `direct_proof`.
Omit `bridge_to_next` only on the final outline item.

Required logic:

- `public_entry`: first spoken hook
- `topic_reveal`: next sentence explicitly naming the subject
- `viewer_promise`: why the viewer should continue
- each outline item has one claim
- each non-final item has `bridge_to_next`
- each item declares `evidence_role`
- analogies are not labeled as direct proof

Validate:

```bash
node <skill_dir>/scripts/validate_artifact.js \
  --type plan \
  --input 03a_longform_plan_<topic_id>.json
```

Pass condition: a viewer hearing only the first three fields knows the topic and expected payoff,
and every adjacent section has a stated logical relationship.

## Checkpoint 7: Review Plan

Run:

```bash
luma-cli agent plan-review \
  --input 03a_longform_plan_<topic_id>.json \
  --model basic_model \
  --output 03b_plan_review_<topic_id>.json
```

Inspect `decision`, score, issues, and revision instructions. A backend pass does not override local
topic-reveal, continuity, or evidence-role failures.

Failure branch:

1. Revise only the plan.
2. Save a versioned plan.
3. Rerun plan review.
4. Do not start material collection before acceptance.

## Checkpoint 8: Plan And Collect Materials

Use the bundled material-planning and collection scripts:

```bash
node <luma-find-material>/scripts/plan_from_topic_review.js \
  --review 03_topic_selection.json \
  --topic-id <topic_id> \
  --longform-plan 03a_longform_plan_<topic_id>.json \
  --max-web-queries 2 \
  --max-image-queries 1 \
  --output 04_material_plan_<topic_id>.json

node <luma-find-material>/scripts/collect_from_material_plan.js \
  --plan 04_material_plan_<topic_id>.json \
  --execute-collection \
  --results-dir materials/<topic_id> \
  --deliverables-dir materials/<topic_id>/final_assets \
  --output 05_material_assets_<topic_id>.json
```

Validate:

```bash
node <skill_dir>/scripts/validate_artifact.js \
  --type materials \
  --input 05_material_assets_<topic_id>.json
```

Required:

- every chapter has a ready asset or generated-component spec
- factual claims have verified evidence or softened wording
- rejected and failed assets remain recorded
- `materials/<topic_id>/final_assets/deliverables_manifest.json` exists
- search snippets never count as proof
- official pages and primary documentation outrank media and social signals
- social signals show attention, not factual truth
- generated components explain the argument but never prove it

Do not continue with an empty material result.

## Checkpoint 9: Build Writing Brief

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

This file is a local writing brief. Do not submit it to backend `script.write`.

Pass condition: the brief contains the selected topic, approved plan, claim constraints, material
coverage, and section briefs.

## Checkpoint 10: Write Script Locally

Write `07_script_<topic_id>.json` yourself:

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

`full_script` must contain the same spoken content as the ordered opening and sections. The title is
not spoken context unless its words also appear in `full_script`.

Mandatory opening order:

```text
public_entry -> topic_reveal -> viewer_promise -> audience_filter_turn
```

Mandatory section rules:

- spoken text, not outline prose
- one argument per section
- explicit bridge between adjacent sections
- source role stated accurately: example, analogy, supporting evidence, or direct proof
- only ready asset IDs in `material_asset_ids`
- no title-card-dependent context

Validate:

```bash
node <skill_dir>/scripts/validate_artifact.js \
  --type script \
  --input 07_script_<topic_id>.json
```

Pass condition: the full script can be understood with the screen turned off.

## Checkpoint 11: Review And Revise Script

Create `08_script_review_payload_<topic_id>.json`:

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

Use the accepted profile snapshot, selected topic card, approved plan, latest plan review, material
coverage summary, and complete local script. Do not submit only `full_script`.

Then run:

```bash
luma-cli agent script-review \
  --input 08_script_review_payload_<topic_id>.json \
  --model basic_model \
  --output 09_script_review_<topic_id>.json
```

If the decision is `revise` or `major_revise`:

1. Map every issue to a script location.
2. Revise the local script, not the approved thesis unless the review identifies a plan defect.
3. Save versioned script, payload, and review files.
4. Rerun local validation and backend review.

Stop after three failed cycles and report the repeated blocker. Do not silently lower the standard.

Pass condition:

- backend decision accepted
- local topic-reveal, continuity, evidence, and material-ID gates pass
- `full_script` is complete spoken copy

## Checkpoint 12: Upload Durable History

Upload the accepted artifacts:

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

Verify with:

```bash
luma-cli --json content history --profile <profile_id>
```

Final handoff to `luma-workflow-original-ip-talk`:

- accepted `07_script_<topic_id>.json`
- accepted `09_script_review_<topic_id>.json`
- `05_material_assets_<topic_id>.json`
- deliverables manifest
- explicit selected topic ID and title

Do not begin TTS or video production without this handoff.
