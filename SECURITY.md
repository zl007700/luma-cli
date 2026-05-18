# Security Policy

## Reporting a Vulnerability

Please report security issues privately instead of opening a public issue.

Email: security@pikgeo.com

Include:

- affected version or commit;
- steps to reproduce;
- impact assessment;
- any relevant logs with secrets removed.

## Secrets

Do not commit:

- card keys;
- npm tokens;
- API tokens;
- private backend URLs;
- database connection strings;
- cloud bucket credentials;
- private production prompts.

Use environment variables for local testing:

```bash
LUMA_CARD_KEY=<CARD_KEY>
LUMA_API_URL=https://api.pikgeo.com
```

## Backend Boundary

`luma-cli` is an open-source client for a hosted backend. The backend is responsible for account registration, billing, entitlement checks, task scheduling, and model execution.

Treat all `risk: write` tools as side-effecting operations.
