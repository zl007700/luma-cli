---
name: luma-core-topic-selection
description: Select one original Luma topic locally from mined signals, creator profile, and content history. Use when a workflow needs agent judgment to cluster signals, reject duplicate or weak candidates, and output 03_topic_selection.json without calling a topic reviewer.
---

# Luma Core Topic Selection

There is no topic reviewer. Select locally.

## Inputs

- `01_profile.json`
- optional `01_profile_extra.md`
- `02_content_history.json`
- `02_raw_signals.json`

## Procedure

1. Cluster related social and web signals into candidate themes.
2. Remove candidates duplicating historical title, thesis, or angle.
3. Score remaining candidates on audience relevance, creator stance, freshness, conflict,
   material availability, and ability to support a full argument.
4. Compare at least three candidates.
5. Select exactly one and record rejection reasons for at least two alternatives.

## Output

Write `03_topic_selection.json`:

```json
{
  "topic_cards": [
    {
      "topic_id": "topic_001",
      "status": "selected",
      "title": "internal topic title",
      "theme": "topic cluster",
      "angle": "specific angle",
      "public_entry": "candidate spoken hook",
      "core_opinion": "creator-specific thesis",
      "common_misunderstanding": "what the audience gets wrong",
      "audience_value": "why the audience should care",
      "why_selected": "comparison-based selection reason",
      "evidence_signals": [
        {
          "source": "douyin",
          "title": "signal title",
          "url": "source URL"
        },
        {
          "source": "websearch",
          "title": "signal title",
          "url": "source URL"
        }
      ],
      "rejected_alternatives": [
        {
          "title": "candidate title",
          "reason": "why it lost"
        }
      ]
    }
  ]
}
```

## Gate

- exactly one card has `status=selected`
- at least three evidence signals span social and web sources
- at least two rejected alternatives are recorded
- selected topic is not a historical near-duplicate
- hook, thesis, audience value, and evidence path are explicit
