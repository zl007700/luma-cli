---
name: luma-core-topic-discovery
description: Discover canonical Luma topic signals from a creator profile and content history. Use when a workflow needs to generate a diverse search matrix, call content topic mine with Douyin/social and websearch, retry weak searches, and output 02_raw_signals.json.
---

# Luma Core Topic Discovery

## Inputs

- `01_profile.json`
- optional `01_profile_extra.md`
- `02_content_history.json`

## Procedure

1. Read identity, audience, stance, avoided topics, and used topic history.
2. Build a search matrix:
   - 2 category terms
   - 2 audience pain or objection terms
   - 2 workflow or use-case terms
   - 1-2 misconception or decision terms
   - 1-2 current product, policy, or market terms when relevant
3. Produce 6-10 non-duplicate social keywords and 3-5 natural-language web queries.
4. Run:

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

5. Inspect `counts` and `raw_signals`.

## Gate

Pass only when:

- at least 15 unique signals exist; target 30 or more
- `counts.social_raw > 0`
- `counts.web_raw > 0`
- at least 8 signals are plausibly relevant after removing ads, traffic bait, and off-profile items
- the searches cover at least three matrix dimensions

If the gate fails, replace weak queries and rerun. `research.run --mode expanded` may suggest better
language, but its output must never replace `02_raw_signals.json`.

## Output

- `02_raw_signals.json`
