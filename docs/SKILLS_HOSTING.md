# Skills Website Hosting

`skills` can install from a normal website through the well-known discovery format. For Luma, the intended public URL is:

```bash
npx -y skills add https://pikgeo.com/skills/luma -g -y
```

## Required File Layout

Host these files under the `/skills/luma` path:

```text
https://pikgeo.com/skills/luma/.well-known/agent-skills/index.json
https://pikgeo.com/skills/luma/.well-known/agent-skills/luma-shared/SKILL.md
https://pikgeo.com/skills/luma/.well-known/agent-skills/luma-content-research/SKILL.md
https://pikgeo.com/skills/luma/.well-known/agent-skills/luma-material/SKILL.md
...
```

The `index.json` can use the simple v1 format:

```json
{
  "skills": [
    {
      "name": "luma-workflow-viral-remix",
      "description": "Use when the user wants a complete viral-remix short-video workflow: research, rewrite, TTS, digital human, PIP materials, subtitles, BGM, and cover.",
      "files": ["SKILL.md"]
    }
  ]
}
```

## Build The Directory

From this repository:

```bash
node scripts/build-skills-site.js
```

The generated directory is:

```text
dist/skills-site
```

Copy the contents of `dist/skills-site` to the website route that serves `https://pikgeo.com/skills/luma`.

## Validate

After deployment:

```bash
npx -y skills add https://pikgeo.com/skills/luma -l
npx -y skills add https://pikgeo.com/skills/luma -g -y
```

When this is live and stable, `luma-cli` can switch its default skills source from `zl007700/luma-cli` to `https://pikgeo.com/skills/luma`.
