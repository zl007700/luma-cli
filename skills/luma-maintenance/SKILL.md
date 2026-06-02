---
name: luma-maintenance
version: 0.1.0
description: "Update or repair Luma / 拾光 / 拾光智能体 / 拾光工具 / 拾光运营套装 by updating luma-cli and syncing agent skills."
metadata:
  requires:
    bins: ["luma-cli"]
  cliHelp: "luma-cli update"
  category: "maintenance"
  entrypoint: true
  aliases: ["更新拾光", "更新拾光智能体", "更新拾光工具", "更新拾光运营套装", "更新 Luma", "update luma", "luma-cli update"]
  relatedSkills: ["luma-shared"]
---

# Luma Maintenance

Use this skill when the user asks to update, repair, refresh, or resync 拾光, 拾光智能体, 拾光工具, 拾光运营套装, Luma, or luma-cli.

## Update

Run:

```bash
luma-cli update
```

This updates the npm CLI package and then syncs the Luma agent skills.

## Check Status

If the user only wants to check whether skills are current:

```bash
luma-cli skills status
```

If the CLI is already current but skills need repair:

```bash
luma-cli skills sync
```

## After Updating

Confirm the installed version and backend:

```bash
luma-cli version
luma-cli auth status
```

If a command still fails because parameters changed, inspect the current tool contract:

```bash
luma-cli --json tools describe <tool_id>
```
