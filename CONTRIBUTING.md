# Contributing

Thanks for helping improve `luma-cli`.

## Project Boundary

This repository contains the open-source CLI client:

- command adapters;
- atomic backend tool wrappers;
- agent tool metadata;
- agent skills;
- local project workspace helpers;
- documentation and release tooling.

The hosted backend remains closed-source and owns registration, billing, model execution, task scheduling, and internal operations.

## Development

```bash
go test ./...
go build ./...
go run . tools list
go run . --json tools describe asr.transcribe
```

## Backend Configuration

The public CLI uses the hosted PikGeo API by default:

```text
https://api.pikgeo.com
```

For development or private deployments, override it locally:

```bash
export LUMA_API_URL=https://your-api.example.com
```

PowerShell:

```powershell
$env:LUMA_API_URL = "https://your-api.example.com"
```

You can also inject credentials through the environment:

```bash
export LUMA_CARD_KEY=<CARD_KEY>
```

## Release Process

Releases are maintained by project maintainers.

1. Update `package.json`.
2. Run local checks:

   ```bash
   go test ./...
   go build ./...
   npm pack --dry-run
   ```

3. Commit and push.
4. Create a version tag:

   ```bash
   git tag v0.0.2
   git push origin v0.0.2
   ```

GitHub Actions builds the multi-platform binaries, creates the GitHub Release, and publishes the npm package.

## Adding a Tool

When adding a new atomic capability:

1. add backend interaction in `internal/atom/`;
2. add CLI adapter code in `internal/commands/`;
3. register agent metadata in `shortcuts/`;
4. update or add a skill in `skills/` if the capability participates in a workflow;
5. add focused tests.

Keep commands atomic. Put workflow glue in skills.

## Pull Request Checklist

- [ ] `go test ./...` passes.
- [ ] New commands are documented through `tools describe`.
- [ ] No secrets, card keys, npm tokens, internal endpoints, or private prompts are committed.
- [ ] README or skills are updated when user-facing behavior changes.
