---
name: luma-find-material
description: Find evidence and visual materials for one selected Luma topic and approved content plan. Use after local topic selection and plan review to search web/image sources, capture screenshots, filter assets, and record claim coverage.
---

# Luma Find Material

Read `../luma-shared/SKILL.md` first. Use this skill only after one topic/plan has been selected and
reviewed. This skill finds materials for that plan; it does not manage a reusable material library
and it must not change the topic.

## Output Chain

```text
03_topic_selection.json
  -> 03a_longform_plan_<topic_id>.json
  -> 03b_plan_review_<topic_id>.json
  -> 04_material_plan_<topic_id>.json
  -> 05_material_assets_<topic_id>.json
```

Before finding materials, the agent must write a minimal `longform_plan` and submit it to backend
`plan.review`. Only proceed to material collection after the review gate passes or the user explicitly
accepts the risk.

`material.plan` decides what must be proved and what may be generated. `material.collect` executes cheap searches, prepares browser tasks, and records claim coverage. Do not rewrite the topic or script here.

Use the shared workflow step names:

- `plan.review`: backend gate before material work.
- `material.plan`: local skill plan file from the approved topic/plan.
- `material.collect`: bounded collection, capture preparation, local filtering, and coverage accounting.
- `content.search.websearch` / `content.search.image`: cloud search atoms invoked by collection.
- `material.review`: cloud VLM review atom for screenshots/images.

Credit estimates for planning only: `content.search.websearch` is 5 credits per query,
`content.search.image` is 8 credits per query, `material.review` is 5 credits per reviewed asset,
and `plan.review` is 5 credits with `basic_model` or 80 credits with `pro_model`. Actual backend
usage/metering is authoritative.

## Ownership

Cloud capabilities:

- `websearch`
- `social`
- `social-account`
- `image_search`
- asset upload and metering

Local skill scripts:

- capture deterministic browser screenshots
- download and inspect image files
- verify real image format and dimensions
- classify web provenance and distinguish primary from third-party evidence
- write project artifacts

Agent capabilities:

- approve the material plan
- consume structured coverage results
- attribute secondary evidence and downgrade unsupported claims during script writing

Keep browser screenshots client-side because access, IP, cookies, and rendering vary by environment.

## 0. Plan Review Gate

After selecting one local topic card and before collecting materials, write a compact
`03a_longform_plan_<topic_id>.json` for backend review.

The review payload must include only:

```json
{
  "profile": {},
  "topic_card": {},
  "longform_plan": {
    "plan_id": "longplan_<topic_id>",
    "target_duration_sec": 240,
    "topic": "selected topic title",
    "public_entry": "the first spoken entry sentence for ordinary strangers",
    "topic_reveal": "the next spoken sentence that explicitly says what this video is about",
    "viewer_promise": "what the viewer will understand or be able to do after watching",
    "core_thesis": "one clear thesis",
    "stance": "the creator's judgment or belief",
    "audience_filter_turn": "how the topic narrows back to the target audience after the public entry",
    "outline": [
      {
        "section": "short section label",
        "claim": "the section's main claim",
        "points": ["supporting point"],
        "bridge_to_next": "why the next section logically follows",
        "evidence_role": "none | example | analogy | supporting_evidence | direct_proof"
      }
    ],
    "fact_boundary": ["what must be softened or verified later"]
  }
}
```

Required `longform_plan` fields:

- `plan_id`
- `target_duration_sec`
- `topic`
- `public_entry`
- `topic_reveal`
- `viewer_promise`
- `core_thesis`
- `stance`
- `audience_filter_turn`
- `outline[]`
- `fact_boundary[]`

Do not include `opening_sentence`, `opening_scene`, `primary_case`, `material_requirements`,
`visual_plan`, `script_sections`, `pip`, screenshots, image ideas, or invented examples as fixed
cases. Material has not been collected yet, so the plan may name evidence needs only inside
`fact_boundary`.

`public_entry` is the first spoken sentence and the reviewer gate. It must be judged for ordinary
strangers who just swiped into the video, not for target users who already know they need AI or
customer acquisition help. Avoid entries that depend on the viewer being a boss, owning customers,
buying tools, knowing AI, or running a store.

`topic_reveal` must immediately follow the hook in the spoken script, normally within the first
10 seconds. A viewer cannot see the internal title or plan, so this sentence must name the actual
subject in plain language. `viewer_promise` states why staying is worthwhile.

Every outline item except the last must include `bridge_to_next`. The bridge must explain the
argument, question, or contrast that makes the next section necessary. `evidence_role` prevents an
analogy or product example from being presented as proof. For example, CRM scoring can illustrate
signal-based prioritization, but it does not directly prove a custom red/yellow/green framework.

Submit the plan:

```bash
luma-cli agent plan-review \
  --input 03a_longform_plan_topic_002.json \
  --output 03b_plan_review_topic_002.json \
  --model basic_model
```

Use `--model pro_model` when the user asks for the stronger reviewer or when repeated basic reviews
are too lenient. If `decision` is not `pass`, revise only the minimal `longform_plan` fields and
resubmit. Do not start `material.plan` until the plan gate is accepted.

## 1. Build The `material.plan`

```bash
node <skill_dir>/scripts/plan_from_topic_review.js \
  --review 03_topic_selection.json \
  --topic-id topic_002 \
  --longform-plan 03a_longform_plan_topic_002.json \
  --max-web-queries 2 \
  --max-image-queries 1 \
  --output 04_material_plan_topic_002.json
```

`--longform-plan` is required after plan review. It binds material claims and chapter visuals to the
approved local plan instead of an older topic-card outline.

Defaults deliberately limit cost:

- web searches: at most 2
- image searches: at most 1
- image search estimate: 8 credits per query
- browser captures: at most 3

For 3-5 minute videos, plan by chapter. Use 1-2 meaningful visuals per chapter, not one asset per sentence.

Inspect:

- `core_claims[].claim_type`
- `core_claims[].evidence_level`
- `collection_tasks`
- `collection_budget`
- `evidence_policy`

## 2. Run `material.collect`

First run without network calls:

```bash
node <skill_dir>/scripts/collect_from_material_plan.js \
  --plan 04_material_plan_topic_002.json \
  --output 05_material_assets_topic_002.json
```

The collector performs a pre-check. Existing search files under the results directory are reused.

To execute the full bounded collection pipeline:

```bash
node <skill_dir>/scripts/collect_from_material_plan.js \
  --plan 04_material_plan_topic_002.json \
  --execute-collection \
  --results-dir materials/topic_002 \
  --output 05_material_assets_topic_002.json
```

`--execute-collection` performs only missing work:

- runs missing `websearch` and `image_search` calls
- captures up to three ranked web candidates per evidence task
- reviews every capture with `luma-cli material review --purpose evidence`
- downloads image-search results, applies local hard filters, then reviews survivors
- reuses existing capture and review artifacts when URL and review context match
- labels accepted web captures as primary, institutional, secondary, or social evidence

It calls:

```bash
luma-cli content search websearch ...
luma-cli content search image ...
```

Use `--query "<single query>"` for one query that may contain commas. Use `--queries` only for an intentional comma-separated query list.
For `content.search.websearch`, `--date-range` accepts only `24h` or `7d`. Default to `7d`; never
invent a longer window.

The collector:

- generates signal and Remotion component specs
- ranks web source candidates
- filters duplicate or undersized image candidates
- creates `capture_url` tasks for browser verification
- records failed and retryable tasks
- does not mark search snippets as proof

## 3. Capture Evidence Manually

For every pending `capture_url` task:

1. Prefer official pages, documentation, primary research, then reputable media.
2. Run the bundled screenshot script. Do not manually automate the browser.
3. Inspect `matched_text` and the screenshot to confirm that it supports the claim.
4. Add only verified captures to the captured manifest.

Keyword-region capture:

```bash
node <skill_dir>/scripts/capture_webpage.js \
  --url "https://example.com/official-release" \
  --mode keyword \
  --keywords "MiniMax M3|1M context" \
  --output materials/topic_002/official_release.png
```

Use `--mode first-screen` for a product page, announcement hero, or source identity shot. Keyword mode
falls back to a first-screen capture and reports `fallback_used: true` when no readable match exists.
The script owns browser launch, viewport, waiting, timeout, PNG validation, and cleanup.

## 4. Download And Filter Images Manually

Run image search through `luma-cli`, then pass its JSON to the bundled local collector:

```bash
node <skill_dir>/scripts/collect_images.js \
  --input materials/topic_002/image_search_mat_image_001.json \
  --output-dir materials/topic_002/images \
  --min-width 800 \
  --min-height 450 \
  --limit 3 \
  --review-topic "老板用 AI 没效果，问题通常不在工具" \
  --review-claim "AI 需要进入业务流程才能产生稳定价值" \
  --review-purpose auxiliary
```

The script downloads candidates, detects their real file format and dimensions, removes placeholders
and duplicate content, then calls `luma-cli material review` for every surviving image. It writes
`images_manifest.json` containing only VLM-approved images under `accepted`. Search-result dimensions
are advisory only. Use `evidence` only when the image must prove a factual claim.

Captured manifest:

```json
{
  "assets": [
    {
      "asset_id": "capture_mat_web_001_01",
      "type": "website_screenshot",
      "evidence_role": "fact_evidence",
      "status": "ready",
      "path": "materials/topic_002/official_release.png",
      "source_url": "https://example.com/official-release",
      "claim_ids": ["claim_001"]
    }
  ]
}
```

Reconcile it:

```bash
node <skill_dir>/scripts/collect_from_material_plan.js \
  --plan 04_material_plan_topic_002.json \
  --results-dir materials/topic_002 \
  --captured-manifest materials/topic_002/captured_manifest.json \
  --output 05_material_assets_topic_002.json
```

Only a ready `fact_evidence` asset with a local path, source URL, claim IDs, and a primary or
institutional source tier may produce `covered_by_fact_evidence`.
For strict primary-source matching, material plans may set `source_policy.preferred_domains`; matching
domains are ranked first and can produce primary fact coverage.

## Coverage Meanings

- `generated_only`: visual explanation only.
- `fallback_only`: claim still lacks evidence.
- `market_signal_only`: proves attention or discussion, not truth.
- `evidence_candidate_found`: source found, screenshot verification pending.
- `covered_by_secondary_evidence`: a verified third-party capture supports the claim; attribute and qualify it.
- `covered_by_fact_evidence`: a verified primary or institutional capture supports the claim.

The script writer must attribute claims backed only by `covered_by_secondary_evidence`, and soften or
remove factual claims that have neither secondary nor primary evidence.

## Collection Priority

For AI commentary:

1. Official webpage or documentation screenshot
2. Search result or reputable media screenshot
3. Social signal card for market attention
4. Generated comparison, process, chapter, or action card
5. Image search for auxiliary visuals
6. Paper/PDF screenshot only for research-heavy claims

Image-search results are auxiliary unless their source is official. Never use an attractive image as factual evidence merely because it matches the topic.

## Rules

- Never let material collection choose a new topic.
- Do not proceed to script writing with an empty material result. Each chapter must have either a
  verified ready asset or an explicit generated-component spec. Search failures and rejected assets
  must remain visible in `05_material_assets_<topic_id>.json`.
- Keep evidence assets and generated components separate.
- Preserve every JSON artifact for `script.write` and `storyboard.plan`.
- Do not invent benchmarks, prices, dates, or capabilities.
- If evidence is unavailable, downgrade the wording instead of forcing a weak source.
