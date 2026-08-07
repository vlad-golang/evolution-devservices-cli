# eds — Evolution DevServices CLI for cloud.ru developer tools

`eds` is a small, agent-friendly command-line client for cloud.ru
developer tools products. It currently covers two products:

- **Repo** (`eds repo`) — git repositories in the cloud.
- **Workflow Studio** (`eds wf`) — wires a repository + branch to a deploy
  pipeline and publishes it.

Repo and Workflow Studio are independent products with independent
credentials; `eds` is just the platform CLI they're both driven through.
Every command is designed to be safely driven by automation and AI
agents: stable `--json` output, environment variables for secrets.

- Single static binary (Go, no runtime dependencies).
- Reads Repo's API key from `EDS_REPO_API_KEY`, Workflow Studio's key
  id/secret from `EDS_WF_KEY_ID`/`EDS_WF_SECRET`, or from
  `~/.config/eds/config.json`.
- Outputs JSON when piped, pretty tables on a TTY (`--json` to force).
- Repo is authenticated with `X-API-KEY` and uses the local `git` CLI for
  clone. Workflow Studio is authenticated with a Bearer token that `eds`
  mints on your behalf (see "Workflow Studio auth" below).

## Installation

### From a GitHub Release (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/cloud-ru/evolution-devservices-cli/main/scripts/install.sh | bash
export PATH="$HOME/.local/bin:$PATH"
eds version
```

The installer detects the platform, downloads the matching static binary
from the latest [GitHub Release](https://github.com/cloud-ru/evolution-devservices-cli/releases),
places it in `~/.local/bin/eds`, and verifies the install.

You can also grab a binary directly from the
[releases page](https://github.com/cloud-ru/evolution-devservices-cli/releases/latest):

```bash
# pick eds-<os>-<arch> for your platform, e.g. eds-darwin-arm64, eds-linux-amd64
curl -fsSL -o eds \
  https://github.com/cloud-ru/evolution-devservices-cli/releases/latest/download/eds-darwin-arm64
chmod +x eds && sudo mv eds /usr/local/bin/eds
```

### From source

```bash
git clone git@github.com:cloud-ru/evolution-devservices-cli.git
cd evolution-devservices-cli
go install github.com/cloud-ru/evolution-devservices-cli@latest
# or
make build           # ./bin/eds
make build-all       # cross-compile darwin/linux × amd64/arm64 into ./dist/
```

> Target platforms: **Linux + macOS** (developers locally + CI).
> Override the matrix if you ever need to: `make build-all OSES="linux darwin" ARCHS="amd64 arm64"`.

## Configuration

The CLI looks for configuration in this order (later wins):

1. Built-in defaults: Repo API URL `https://devtools.api.cloud.ru/repo/api/v1`,
   git host `https://repo.cloud.ru/`, Workflow Studio API URL
   `https://pipeline.cloud.ru/public-api/v1`, IAM URL
   `https://iam.api.cloud.ru/api/v1/auth/token`.
2. File `~/.config/eds/config.json` (overridable via `EDS_CONFIG` or
   `XDG_CONFIG_HOME`) — a single platform-level file shared by both products.
3. Environment variables (see table below).
4. Per-command flags.

Env vars and flags are namespaced by product — `EDS_REPO_*`/`--repo-*` for
Repo, `EDS_WF_*`/`--wf-*` for Workflow Studio — except for the truly
platform-level settings (`EDS_PROJECT_ID`/`--project`, `EDS_IAM_URL`/`--iam-url`),
which are shared.

Example config file (`~/.config/eds/config.json`):

```json
{
  "api_url": "https://devtools.api.cloud.ru/repo/api/v1",
  "project_id": "3232b2d0-1063-41e6-b2fa-13df767f4a0a",
  "api_key": "...",
  "git_host": "https://repo.cloud.ru/",
  "workflow_api_url": "https://pipeline.cloud.ru/public-api/v1",
  "workflow_key_id": "...",
  "workflow_secret": "...",
  "iam_url": "https://iam.api.cloud.ru/api/v1/auth/token",
  "workflow_access_token": "... (cached, managed by the CLI)",
  "workflow_token_expires_at": "... (cached, managed by the CLI)"
}
```

### Production vs dev (Repo product)

- **Production** — `https://devtools.api.cloud.ru/repo/api/v1`, git host
  `https://repo.cloud.ru/`.
- **Dev** — `https://devtools.dev.api.internal.cloud.ru/repo/api/v1`.

Override with `--repo-api-url` or `EDS_REPO_API_URL`.

### Workflow Studio auth

Workflow Studio has its own base URL (`--wf-api-url` / `EDS_WF_API_URL`) and
its own credential pair (`--wf-key-id`/`--wf-secret` or
`EDS_WF_KEY_ID`/`EDS_WF_SECRET`) — it is **not** authenticated with
`--repo-api-key`/`EDS_REPO_API_KEY`. The key id/secret are exchanged for a
short-lived Bearer access token via the cloud.ru IAM service
(`POST /api/v1/auth/token` at `--iam-url` / `EDS_IAM_URL`, default
`https://iam.api.cloud.ru/api/v1/auth/token`). The CLI does this
automatically before the first `eds wf ...` call and caches the resulting
token (and its expiry) in the config file, re-exchanging it once it
expires — you never need to call the IAM endpoint yourself.

## Commands

```text
eds login                                  save credentials to the local config
eds config                                 show effective configuration
eds version                                print the CLI version

eds repo list [--search S] [--sort ...]    list git repositories
eds repo create <name>                     create a repository
eds repo show <id-or-name>                 show repository details
eds repo delete <id-or-name> [--force]     delete a repository
eds repo clone <id-or-name> [dir] [--ssh]  clone via local git CLI

eds wf app create <name> --repository R|--repository-url URL --branch B   create a Workflow Studio application
eds wf app list                                       list applications
eds wf app show <id>                                  show application details
eds wf app update <id> --branch B [--name N]          update name/branch
eds wf app delete <id> [--force]                       delete an application
eds wf app deploy <id>                                run the pipeline and publish
eds wf app deployments <id>                            list publish history
eds wf app status <id>                                 run/stage/job status + live URL

eds wf run show <id>                          show a run's status, stages and jobs
eds wf run list [--pipeline-id ID]             list runs
eds wf run stop <id>                          stop a running run

eds wf job show <id>                          show job details
eds wf job list --run-id ID                   list jobs for a run
eds wf job logs <id>                          stream a job's logs
eds wf job retry <id>                         retry a failed/canceled job
eds wf job stop <id>                          stop a running job
```

### Login

```bash
eds login --repo-api-key "$EDS_REPO_API_KEY" --project <project-id>
eds login --repo-api-url https://devtools.dev.api.internal.cloud.ru/repo/api/v1 \
          --repo-api-key "$EDS_REPO_API_KEY" --project <project-id>

# Workflow Studio uses a separate credential pair, exchanged for a Bearer
# token via cloud.ru IAM on first use (cached automatically afterwards)
eds login --wf-key-id "$WF_KEY_ID" --wf-secret "$WF_SECRET" \
          --project <project-id>
```

### For an AI agent

The CLI is designed to be safely scripted. Recommended pattern:

```bash
export EDS_REPO_API_KEY="..."
export EDS_PROJECT_ID="..."

# List repos as JSON, pipe into jq
eds repo list --json | jq '.repositories[].name'

# Create a repo, capture the new id
eds repo create demo --json | jq -r '.id'

# Inspect
eds repo show demo --json | jq '.clone.https'

# Clone (uses the system's git)
eds repo clone demo ./work/demo
```

All commands return exit code 0 on success, non-zero on failure, and
write errors to stderr in a single line:

```
Error: api error 404: {"error_msg":"404 Not Found"}
```

See [`skill/SKILL.md`](skill/SKILL.md) for the canonical Agent Skills
description that can be attached to an AI agent.

## Workflow Studio (deploy & publish)

Workflow Studio is a "developer tools" product on cloud.ru: it wires a
repository + branch to a deploy pipeline. An **application** is that wiring;
running the application's pipeline (a **deployment**) publishes it and
produces a live URL. This is the CLI's main "vibe-coded a site, now ship it"
path — `eds repo` gets your code hosted, `eds wf app` publishes it.

```bash
export EDS_WF_KEY_ID="..."     # separate credential pair from EDS_REPO_API_KEY
export EDS_WF_SECRET="..."     # exchanged for a Bearer token via cloud.ru IAM automatically
export EDS_PROJECT_ID="..."

# 1. Create the repo and push code (see "File upload / push" below)
eds repo create my-site
eds repo clone my-site && cd my-site
# ...add code + Dockerfile...
git add . && git commit -m "init" && git push origin main

# 2. Wire it to a Workflow Studio application
APP_ID=$(eds wf app create my-site --repository my-site --branch main --json | jq -r '.id')

# 3. Publish it
eds wf app deploy "$APP_ID" --json | jq -r '.run_id'

# 4. Poll status until it's done, then read the live URL
eds wf app status "$APP_ID" --json | jq '{status: .application.status, url: .latest_deployment.url}'
```

If a deployment fails, drill into the failing job's logs:

```bash
eds wf app status "$APP_ID" --json | jq -r '.application.run.stages[].jobs[] | select(.status=="failed") | .id' \
  | xargs -I{} eds wf job logs {}
```

`eds wf run` and `eds wf job` are the lower-level primitives behind `eds wf
app status` — use them directly when you need to inspect or control a
particular run/job (e.g. `eds wf run stop`, `eds wf job retry`).

## File upload / push

This CLI does **not** implement a custom upload path. To push code, use
the standard git workflow after `eds repo clone`:

```bash
eds repo clone demo
cd demo
git add . && git commit -m "init" && git push origin main
```

`git push` works out of the box because the API key is used as
basic-auth credentials on the smart-HTTP endpoint exposed by the server.

## Distribution / publishing

Releases are published on
[GitHub Releases](https://github.com/cloud-ru/evolution-devservices-cli/releases).

```bash
# cross-compile everything into ./dist/ + sha256 checksums
make release

# tag, then cut the release with the built artifacts
git tag v0.2.0 && git push origin v0.2.0
gh release create v0.2.0 dist/* --generate-notes
```

`scripts/install.sh` downloads from the latest GitHub Release by default.
The `Makefile` also has `make upload`/`make upload-latest` targets for
optionally mirroring builds to an S3-compatible bucket (e.g. for internal
environments without GitHub access) — pass `EDS_CLI_BASE_URL` to
`install.sh` to install from a mirror instead of GitHub:

```bash
export AWS_ENDPOINT_URL=https://storage.cloud.ru
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
make upload BUCKET=my-bucket PREFIX=evolution-devservices-cli VERSION=v0.2.0

curl -fsSL https://storage.cloud.ru/my-bucket/evolution-devservices-cli/install.sh | \
  EDS_CLI_BASE_URL=https://storage.cloud.ru/my-bucket/evolution-devservices-cli bash
```

## Development

```bash
make vet           # go vet ./...
make test          # go test ./...
make build         # current platform into ./bin/
make build-all     # full matrix into ./dist/
make clean         # remove ./bin and ./dist
make help          # list targets
```

Project layout:

```
main.go                         # entry point + ldflags-driven version
cmd/
  root.go                       # cobra root command (`eds`) + global flags
  helpers.go                    # config resolution, runtime context
  login.go                      # `eds login`
  config.go                     # `eds config`
  version.go                    # `eds version`
  repo.go                       # `eds repo list|create|show|delete|clone`
  wf.go                         # `eds wf` parent command (groups app/run/job)
  app.go                        # `eds wf app create|list|show|update|delete|deploy|deployments|status`
  run.go                        # `eds wf run show|list|stop`
  job.go                        # `eds wf job show|list|logs|retry|stop`
internal/
  config/                       # disk config + env overrides
  output/                       # JSON / table formatting
  repoapi/                      # thin HTTP client for the Repo product API
  workflowapi/                  # thin HTTP client for the Workflow Studio product API
  iam/                          # exchanges wf-key-id/secret for a Bearer token
scripts/
  install.sh                    # one-liner installer (GitHub Releases by default)
skill/
  SKILL.md                      # Agent Skills description
Makefile                        # build, build-all, release, upload, …
```

## Environment variables

| Variable          | Description                                                 |
| ----------------- | ------------------------------------------------------------ |
| `EDS_PROJECT_ID`  | Default project ID (shared across products)                  |
| `EDS_IAM_URL`     | cloud.ru IAM token endpoint (shared, overrides config)        |
| `EDS_REPO_API_URL`  | Repo product API base URL (overrides config)                |
| `EDS_REPO_API_KEY`  | Repo product X-API-KEY value                                 |
| `EDS_REPO_GIT_HOST` | Repo product git smart-HTTP host                             |
| `EDS_WF_API_URL`  | Workflow Studio product API base URL (overrides config)      |
| `EDS_WF_KEY_ID`   | Workflow Studio key id (separate from Repo's API key)         |
| `EDS_WF_SECRET`   | Workflow Studio secret (paired with key id)                   |
| `EDS_CONFIG`      | Path to config file (overrides default)                      |
| `XDG_CONFIG_HOME` | Respected when locating the config file                      |
| `AWS_ENDPOINT_URL` | S3-compatible endpoint for `make upload` (optional mirror)  |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development setup and PR
process, and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for community
guidelines. Report security issues to opensource@cloud.ru rather than a
public issue.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
