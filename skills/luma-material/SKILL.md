---
name: luma-material
version: 0.1.0
description: "Use when an agent needs to inspect local material libraries, describe ZA-AGENT style material groups, upload or understand materials, search candidates, or prepare PIP matching inputs."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli tools describe material.group.describe"
  category: "capability"
  entrypoint: true
  relatedSkills: ["luma-shared", "luma-workflow-viral-remix"]
---

# Luma Material

Use this skill for local material libraries, cloud material understanding, material search, and PIP matching preparation.

Read `../luma-shared/SKILL.md` first for common project and output rules.

## When To Use

- The user provides a local material group, for example `data/material_library/groups/vlm_ai`.
- A workflow needs `step4_materials_enriched.json` or `step4_material_matches.json`.
- The agent needs to inspect which materials are available before PIP rendering.
- The user wants to upload or understand a material through the backend.

## Local Material Group Flow

List available groups:

```bash
luma-cli material group list ./material_library/groups --output material_groups.json
```

Describe one group into a standard materials file:

```bash
luma-cli material group describe ./material_library/groups/vlm_ai --output step4_materials_enriched.json
```

Search candidate materials before planning:

```bash
luma-cli material search --materials step4_materials_enriched.json --query "<script_or_scene_text>" --limit 8 --output step4_material_matches.json
```

## Cloud Understanding

If local metadata is missing or weak, understand useful materials one by one:

```bash
luma-cli material understand ./materials/a.jpg --output a.meta.json --descriptor-output a.material.json
luma-cli material merge --materials step4_materials.json --meta ./materials_meta --output step4_materials_enriched.json
```

## PIP Matching

After subtitles are segmented and scene units are ready:

```bash
luma-cli pip scene --segments step4_segments.json --output step4_scene_units.json
luma-cli pip match --scenes step4_scene_units.json --materials step4_materials_enriched.json --mode auto --output step4_material_matches.json
```

Use `--mode cloud` when semantic matching must be backend-only. Use `--mode local` only as a deterministic fallback.

## Agent Rules

- Prefer `material group describe` for ZA-AGENT style local libraries.
- Do not upload an entire local library unless the user explicitly asks; upload only materials that need cloud understanding.
- If matching returns zero inserts, report that no suitable materials were found and continue without PIP.
- Keep `materials.json`, scene units, matches, and PIP plan as separate artifacts.
- Do not expose backend object storage implementation details to the user unless a command explicitly requires them.
