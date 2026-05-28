---
name: robotx
description: Use the robotx CLI to deploy, manage versions, and check status for RobotX applications.
metadata:
  short-description: RobotX deployment CLI skill
---

# RobotX Deployment Skill

Use this skill when an agent needs to deploy or manage project versions on RobotX using the `robotx` CLI.

## Quick start

- Check CLI availability: `which robotx || which robotx_cli`
- Install (binary-first, no Go required):
  - `curl -fsSL https://raw.githubusercontent.com/haibingtown/robotx_cli/main/scripts/install.sh | bash`
  - In mainland China or slow GitHub networks, prefer the default RobotX mirror relay with a pinned version:
    - `curl -fsSL https://mr.robotx.xin/https://raw.githubusercontent.com/haibingtown/robotx_cli/main/scripts/install.sh | env ROBOTX_VERSION=v0.5 ROBOTX_GITHUB_PROXY=https://mr.robotx.xin bash`
  - If the default relay returns 502 or times out while downloading GitHub release assets, use the fallback release proxy:
    - `curl -fsSL https://mr.robotx.xin/https://raw.githubusercontent.com/haibingtown/robotx_cli/main/scripts/install.sh | env ROBOTX_VERSION=v0.5 ROBOTX_GITHUB_PROXY=https://gh-proxy.com bash`
- Fallback install (if you explicitly want source install):
  - `go install github.com/haibingtown/robotx_cli/cmd/robotx@latest`
  - Or auto PATH setup: `curl -fsSL https://raw.githubusercontent.com/haibingtown/robotx_cli/main/scripts/go-install.sh | bash`

## Configure

Set credentials by config file (`~/.robotx.yaml`) or env vars:

- `ROBOTX_BASE_URL`
- `ROBOTX_API_KEY`

## Auth pre-check and default login

Before running any API command (`deploy`, `access`, `projects`, `projects delete`, `versions`, `status`, `publish`),
verify local auth first.

Recommended quick check:

```bash
robotx doctor --output json
```

If you see auth-related errors (`missing_base_url`, `missing_api_key`, `401`, `403`), always try `robotx login` first, then fall back only if login fails:

1. Default (interactive, browser-based): run `robotx login` and retry the original command.
   - `robotx login --base-url https://robotx.xin`
   - The CLI prints a verification URL + user code, then auto-opens your browser for authorization.
   - Complete the login in the browser; the CLI polls and saves credentials to `~/.robotx.yaml`.
   - Headless/remote mode: add `--no-browser` and open the printed URL manually.
   - For RobotX hosted login authorization, use `robotx.xin` (not `api.robotx.xin`).
2. Fallback (only if login is not possible or fails): manual API key setup via console and configure locally.
   - `export ROBOTX_BASE_URL=https://your-robotx-server.com`
   - `export ROBOTX_API_KEY=your-api-key`
   - Or write `~/.robotx.yaml`:

```yaml
base_url: https://your-robotx-server.com
api_key: your-api-key
```

For CI/non-interactive environments, prefer env vars over `robotx login`.

## Machine-readable output

For agents and workflows, always use structured output:

- `robotx deploy . --name my-app --output json`
- `robotx projects --limit 50 --output json`
- `robotx versions --project-id proj_123 --output json`
- `robotx status --project-id proj_123 --output json`
- `robotx publish --project-id proj_123 --build-id build_456 --output json`
- `robotx access status --project-id proj_123 --output json`
- `robotx projects delete --project-id proj_123 --yes --output json`

JSON is written to stdout. Progress logs are written to stderr.

## Common commands

### Deploy

```bash
robotx deploy [path] --create --target main --name my-app --publish=true --wait=true --output json
```

Use `--access open` only when the user explicitly wants the production URL to be anonymously accessible. `--visibility public` alone does not mean anonymous access.

```bash
robotx deploy [path] --update --target main --access open --verify-url --output json
```

Use `--upsert --name my-app` only when the user explicitly wants legacy create-or-update behavior by name.

### Versions

```bash
robotx versions --project-id proj_123 [--limit 20]
```

`versions` alias: `robotx builds --project-id proj_123`.

### Projects

```bash
robotx projects [--limit 50]
robotx projects delete --project-id proj_123 --yes
```

`projects delete` deletes the remote RobotX project and requires explicit user approval before running. It does not remove local `.robotx/targets.json` records; use `robotx targets remove <name>` for local target records.

### Access

```bash
robotx access status --project-id proj_123
robotx access open --project-id proj_123
robotx access login --project-id proj_123
robotx access private --project-id proj_123
```

`open` means anonymous access is allowed. `login` means RobotX login is required. `private` means allowlist access.

### Status

```bash
robotx status --project-id proj_123 [--build-id build_456]
```

`status` accepts `--project-id`, `--build-id`, or both. Build logs are no longer available because RobotX no longer runs remote builds.

### Publish

```bash
robotx publish --project-id proj_123 --build-id build_456
```

## MCP note

`robotx mcp` is currently a placeholder and not available for production use. Use shell/CLI mode for agent integration.
