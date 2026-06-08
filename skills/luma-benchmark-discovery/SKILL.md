---
name: luma-benchmark-discovery
version: 0.1.0
description: "Use when a Luma / 拾光 agent needs to discover, evaluate, and shortlist benchmark or competitor social accounts for content planning. Guides the agent to call luma-cli profile/content search commands, aggregate authors from social raw signals, score account fit against a profile, and produce A/B/C/reject tracking recommendations."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli profile get; luma-cli content search social; luma-cli content search social-account; luma-cli content topic mine"
  category: "capability"
  entrypoint: true
  aliases: ["对标账号", "找对标账号", "竞品账号", "benchmark accounts", "competitor accounts", "账号挖掘"]
  relatedSkills: ["luma-shared", "luma-content-script"]
---

# Luma Benchmark Discovery

Use this skill to find useful benchmark accounts before topic mining and script writing.

Read `../luma-shared/SKILL.md` first for auth, JSON output, and artifact rules.

## Goal

Build a small, explainable benchmark account pool for one active profile:

- A-class seed accounts: 5-10, track daily.
- B-class observation accounts: 10-30, track every 2-3 days or weekly.
- C-class candidates: keep for weekly review.
- Reject: noisy, irrelevant, repost, or pure marketing accounts.

Do not chase a huge realtime competitor system. V1 should be small, incremental, and easy for the user to inspect.

## Inputs

Prefer the current global profile:

```bash
luma-cli --json profile current
luma-cli --json profile get <profile_id>
```

If no profile is active, ask the user which profile to use or create one with `luma-cli profile create`.

Start from 5-10 industry keywords. Derive them from:

- profile identity
- audience
- stance
- user-provided niche
- known category terms

For the current AI SaaS Agent founder profile, good seed keywords are:

```text
AI工具, AI智能体, AI获客, AI营销, AI创业, AI副业, 私域运营, 短视频获客, 中小企业AI, 普通人用AI
```

## Standard Flow

### 1. Collect Raw Social Signals

Run social search first:

```bash
luma-cli content search social \
  --keywords "AI工具,AI智能体,AI获客,AI营销,AI创业" \
  --date-range 7d \
  --limit-per-keyword 20 \
  --output benchmark_social_signals.json
```

For a broader first pass, use `topic mine` with only social keywords:

```bash
luma-cli content topic mine \
  --social-keywords "AI工具,AI智能体,AI获客,AI营销,AI创业,短视频获客,中小企业AI" \
  --date-range 7d \
  --limit-per-keyword 20 \
  --max-raw 200 \
  --output benchmark_raw_signals.json
```

Use `--json` when the agent needs to parse command output. Always read the saved JSON file; stdout may contain notices.

### 2. Aggregate Authors

Read `raw_signals` from the search output. Group by:

```text
platform + author_id
```

If `author_id` is missing, use a weak key:

```text
platform + author_name + sample_url
```

Mark weak-key candidates as lower confidence.

For each candidate author, calculate:

- `matched_keywords`
- `matched_video_count`
- `avg_engagement`
- `max_engagement`
- `recent_video_titles`
- `sample_urls`
- `latest_published_at`
- `confidence`: `high | medium | low`

Use available stats such as likes/comments/shares if present. Do not invent metrics.

### 3. First-Pass Filtering

Reject or down-rank before deep thinking:

- no author identity
- only one weak low-engagement hit
- obvious ads, reposts, course spam, or pure tool affiliate accounts
- mixed-content accounts with weak verticality
- topics unrelated to the active profile audience
- titles with no reusable topic structure

Keep roughly the top 20 candidate accounts for scoring.

### 4. Fetch Recent Videos For Top Candidates

For the top candidate authors, fetch recent account videos before final scoring. Keep this capped: V1 should fetch at most 20 candidate accounts, usually one page per account.

```bash
luma-cli content search social-account \
  --accounts "account_id_or_sec_user_id_or_profile_url" \
  --max-pages 1 \
  --count 20 \
  --output benchmark_account_recent_videos.json
```

Accepted account inputs include numeric UID, `sec_user_id`, unique/short id, Douyin profile URL, or a Douyin short link when the backend can resolve it.

By default, `social-account` returns slim normalized signals. Use `--include-raw` only when debugging provider payloads, because raw provider data can make files large and slow down batch runs.

Use these recent videos to improve:

- verticality judgment
- stable topic direction
- originality/repost detection
- recurring audience language
- whether the account has enough useful recent output to track

If account lookup fails for a candidate, keep the first-pass evidence and lower confidence rather than blocking the entire discovery run.

### 5. Score Against Profile

Score each candidate from 0-10:

- `vertical_score`: consistently about the target domain.
- `engagement_score`: meaningful interaction relative to observed videos.
- `topic_fit_score`: fits the profile's audience and stance.
- `originality_score`: appears original, not repost/mass marketing.
- `content_density_score`: titles imply useful arguments, objections, workflows, or audience language.
- `benchmark_value_score`: likely to improve topic discovery if tracked.

Do not choose accounts merely because they are big. Choose accounts that repeatedly surface useful topics, angles, objections, or audience language.

### 6. Assign Tracking Tier

Use these rules:

- `A`: high profile fit, strong verticality, useful topics, stable direction. Track daily.
- `B`: relevant but not central; useful occasional topics. Track 2-3 days or weekly.
- `C`: interesting but insufficient evidence. Recheck weekly.
- `reject`: noisy, irrelevant, spam, repost, low topic density, or not useful for this profile.

For V1, recommend no more than 10 A-class accounts.

## Output Contract

Write strict JSON to `benchmark_accounts.json` unless the user gives another path:

```json
{
  "profile_id": "ai_saas_agent_founder",
  "platform": "douyin",
  "source_keywords": ["AI工具", "AI智能体"],
  "generated_at": "2026-06-05T09:00:00Z",
  "updated_at": "2026-06-05T09:00:00Z",
  "next_refresh_after": "2026-06-12T09:00:00Z",
  "refresh_policy": {
    "interval_days": 7,
    "refresh_when_stale": true
  },
  "limits": {
    "raw_signal_limit": 200,
    "candidate_author_limit": 50,
    "scored_account_limit": 20,
    "seed_account_limit": 10
  },
  "recommended_seed_accounts": [],
  "observation_accounts": [],
  "weekly_candidates": [],
  "rejected_accounts": [],
  "notes": []
}
```

Each account item must use:

```json
{
  "platform": "douyin",
  "account_id": "",
  "nickname": "",
  "profile_url": "",
  "matched_keywords": [],
  "matched_video_count": 0,
  "avg_engagement": 0,
  "max_engagement": 0,
  "recent_video_titles": [],
  "sample_urls": [],
  "vertical_score": 0,
  "engagement_score": 0,
  "topic_fit_score": 0,
  "originality_score": 0,
  "content_density_score": 0,
  "benchmark_value_score": 0,
  "tier": "A",
  "confidence": "medium",
  "keep": true,
  "reason": ""
}
```

If the CLI output does not include enough author fields, still produce a candidate file, but:

- set `confidence` to `low`
- explain the missing fields in `notes`
- ask for either richer social search output or manual seed account URLs before finalizing A-class accounts

## Profile-Bound Cloud Persistence

Benchmark accounts are long-term profile memory. After producing `benchmark_accounts.json`, bind it to the active profile instead of leaving it as a one-off local file.

Benchmark accounts should usually refresh weekly, not daily. Treat the approved benchmark JSON as a cached profile-bound memory.

Freshness policy:

- If no `kind=benchmark` asset exists, run discovery.
- If a benchmark asset exists and its `updated_at` or profile asset `created_at` is less than 7 days old, reuse it.
- If it is 7+ days old, run a refresh discovery and propose account additions, removals, and tier changes.
- If the user explicitly asks for a refresh, run discovery even before 7 days.
- Do not refresh silently when the user only needs today's topic; report that the benchmark pool is stale and ask before updating if the refresh may cost credits.

Default persistence flow:

1. Present the candidate pool to the user for review.
2. Ask before promoting accounts into the long-term benchmark pool.
3. Set `updated_at` to the current time and `next_refresh_after` to 7 days later.
4. After approval, upload the JSON to the profile's cloud asset group:

   ```bash
   luma-cli profile asset upload <profile_id> benchmark_accounts.json \
     --kind benchmark \
     --name benchmark_accounts
   ```

This stores the file in the profile asset group, usually:

```text
profile_<profile_id>
```

and records the uploaded object in the profile's `assets` list.

Before future benchmark discovery or topic mining, check whether the profile already has a benchmark asset:

```bash
luma-cli --json profile asset list <profile_id>
```

If a `kind=benchmark` asset exists, treat it as the current benchmark memory and check its freshness before doing any search. Use new discovery runs to propose additions, removals, or tier changes rather than replacing the pool blindly.

Recommended versioning:

- Use `benchmark_accounts` for the current approved pool.
- Use `benchmark_candidates_<date>` for unreviewed discovery output.
- Keep raw search files local or as project artifacts unless the user wants to preserve them.

Do not upload rejected raw signals as profile memory. Only upload curated benchmark account JSON.

## Cost Discipline

Default V1 limits:

```text
keywords: 5-10
results per keyword: 20
raw signals: max 200
candidate authors: max 50
scored accounts: max 20
final A-class seeds: max 10
```

Do not fetch or analyze every account deeply. If future CLI account-search exists, fetch recent videos only for the top 20 candidates.
`social-account` now exists, so use it only after first-pass filtering. It is priced like search by account/query count, so keep it capped. Prefer batches of 2-3 accounts if the provider is slow.

## Human Review

Do not silently overwrite a long-term benchmark pool. Present:

- which accounts to add to A
- which to keep in B/C
- which to reject
- why each A account helps this profile

Ask for confirmation before treating new A-class accounts as the long-term seed pool.

## Quality Bar

A useful benchmark account helps answer:

```text
What is the market currently talking about,
and how can this profile say something sharper?
```

If an account cannot improve topic discovery, audience language, or angle selection, reject it even if its engagement is high.
