# Luma Skills

Luma skills are agent-facing instructions for composing `luma-cli` atoms into production workflows.

## Design

Skills are split into three layers:

1. `luma-shared`: common rules for auth, projects, artifacts, runtime, and failures.
2. Capability skills: focused instructions for one capability domain.
3. Workflow skills: user-goal workflows that compose capability skills.

Current core skills:

| Skill | Layer | Purpose |
| --- | --- | --- |
| `luma-shared` | shared | Common rules every Luma agent should know |
| `luma-content-research` | capability | Research, keyword tables, persona-based search |
| `luma-material` | capability | Local material groups, material search, PIP matching |
| `luma-digital-human` | capability | Voice clone, TTS, avatar, lip-sync |
| `luma-subtitle` | capability | Text segmentation and subtitle rendering |
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
npx -y skills add zl007700/luma-cli -g -y
```

Selective install is a maintainer/platform feature, not the default user path:

```bash
luma-cli skills sync -s luma-workflow-viral-remix
npx -y skills add zl007700/luma-cli -s luma-workflow-viral-remix -g -y
```

User-facing docs should present the full skill pack install. Selective install is useful for marketplace validation, troubleshooting, or future platform ingestion rules.

Update flow:

```bash
luma-cli update
```

This updates the CLI through npm and then syncs skills through the skills installer. A local stamp is written to `~/.luma/skills.stamp.json`; if the CLI version changes while the synced skills version does not, the CLI prints a short notice asking the user or agent to run `luma-cli update`.

Distribution channels:

1. npm installs the CLI shell and native binary launcher.
2. `npx skills add zl007700/luma-cli -g -y` installs public skills from the repo.
3. `https://pikgeo.com/skills/luma` can host the same skills through the well-known website format. See [SKILLS_HOSTING.md](./SKILLS_HOSTING.md).
4. GitHub Release still uploads `luma-skills-vX.Y.Z.zip` as a backup/import artifact for platforms that prefer zip ingestion.
5. Skills platforms can ingest the repo path, website source, release zip, or individual skill folders.

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

Do not couple skill publishing to npm only. Skills should be independently packageable and platform-friendly.

## Authoring Rules

- Keep `SKILL.md` concise and goal-oriented.
- Put long examples or domain details into `references/`.
- Use stable intermediate file names so agents can resume work.
- Reference `luma-cli tools describe <tool_id>` instead of duplicating every flag.
- Keep prompt-heavy and closed-source logic on the backend.
