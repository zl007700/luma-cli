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
