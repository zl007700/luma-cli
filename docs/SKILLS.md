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

The npm package should install and run `luma-cli`. It does not need to be the only distribution channel for skills.

Recommended distribution model:

1. GitHub Release remains the source of versioned CLI binaries.
2. The same release should also publish a separate `luma-skills-vX.Y.Z.zip` artifact.
3. Skills platforms can ingest the zip, repo path, or individual skill folders directly.
4. A future CLI command can install skills from the current release:
   ```bash
   luma-cli skills list
   luma-cli skills install --target ~/.codex/skills
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
