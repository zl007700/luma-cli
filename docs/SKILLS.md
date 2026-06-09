# Luma Skills

Luma skills are agent-facing instructions for composing `luma-cli` atoms into production workflows.

## Design

Skills are split into four layers:

1. `luma-shared`: common rules for auth, projects, artifacts, runtime, and failures.
2. Core skills: one agent-authored reasoning or writing operation with a stable file contract.
3. Capability skills: focused instructions for one CLI/media domain.
4. Workflow skills: pure glue that composes core skills, capability skills, and CLI atoms.

Current core skills:

| Skill | Layer | Purpose |
| --- | --- | --- |
| `luma-shared` | shared | Common rules every Luma agent should know |
| `luma-maintenance` | maintenance | Update Luma CLI and sync agent skills |
| `luma-profile-onboarding` | capability | Guide users to create and save a usable profile |
| `luma-content-ip-writing` | workflow | Profile-based IP content research and writing with dual review gates |
| `luma-benchmark-discovery` | capability | Discover and score benchmark accounts |
| `luma-find-material` | capability | Find evidence and visual materials for an approved content plan |
| `luma-material` | capability | Local material groups, material search, PIP matching |
| `luma-digital-human` | capability | Voice clone, TTS, avatar, lip-sync |
| `luma-ppt-video` | capability | Render align.json into PPT-style MP4, with optional avatar PiP |
| `luma-subtitle` | capability | Text segmentation and subtitle rendering |
| `luma-workflow-original-ip-talk` | workflow | Original profile-to-video workflow with digital human and PPT visuals |
| `luma-workflow-viral-remix` | workflow | Research-to-video viral remix workflow |

## Distribution

The npm package installs and runs `luma-cli`. Skills are distributed separately: CLI binaries come from npm/GitHub Release, while agent skills are installed through the skills installer.

Recommended user flow:

```bash
npm install -g @lumageo/luma-cli
luma-cli skills sync
```

`luma-cli skills sync` runs:

```bash
luma-cli skills sync
```

Update flow:

```bash
luma-cli update
```

This updates the CLI through npm and then syncs skills through the skills installer. A local stamp is written to `~/.luma/skills.stamp.json`; if the CLI version changes while the synced skills version does not, the CLI prints a short notice asking the user or agent to run `luma-cli update`.

Distribution channels:

1. npm installs the CLI shell and native binary launcher.
2. `luma-cli skills sync` installs the public Luma skill pack.
3. GitHub Release uploads a versioned skills bundle for compatible agent runtimes.
4. Skills platforms can ingest the repo path, release bundle, or individual skill folders.

Useful commands:

   ```bash
   luma-cli skills list
   luma-cli skills status
   luma-cli skills sync
   luma-cli update
   ```

This gives two paths:

- Users who discover Luma through npm install the CLI first, then install or sync skills.
- Users who discover Luma from a skills marketplace install the skills first, then follow the skill instructions to install `luma-cli`.

Skills are independently packageable and platform-friendly.

## Authoring Rules

- Keep workflow `SKILL.md` files concise and limited to step order, invoked command/skill, input,
  output, and failure routing.
- Put agent-authored schemas and quality rules in core skills, not workflow skills.
- Keep every required contract in `SKILL.md`; agents must not depend on proactively loading a
  separate reference file.
- Put deterministic repeated transformations in bundled scripts.
- Use stable intermediate file names so agents can resume work.
- Reference `luma-cli tools describe <tool_id>` instead of duplicating every flag.
- Use Luma cloud services for advanced material understanding and semantic matching.
