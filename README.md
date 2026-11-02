# Semver AI

**Your AI Release Engineer.** A Go CLI that analyzes changes across programming
languages, predicts Semantic Versions and prepares human-approved GitLab releases.
AI classifies changes with evidence; a deterministic Go engine calculates versions.
There is no database, daemon or dashboard.

## Quick start

Requires Go 1.26+ and Git with full history/tags. An OpenAI-compatible endpoint
supporting strict JSON Schema output is required for analysis.

```sh
go build -trimpath -o bin/release-manager ./cmd/release-manager
cp examples/.release-manager.yml .release-manager.yml
# Edit ai.model, gitlab.api_url and release.branch.
export SEMVER_AI_API_KEY='your-provider-key'
export GITLAB_TOKEN='your-project-token'
export SEMVER_PROJECT='group/project'  # CI uses CI_PROJECT_ID instead
bin/release-manager validate-config
bin/release-manager predict --json
bin/release-manager prepare-release --json
```

Preparation writes `semver-release-candidate.json` and `RELEASE_NOTES.md`. It exits
successfully with `NO_RELEASE` and no candidate for documentation-only changes.
Publication is run only from an authorized manual protected GitLab job after a
human reviews the candidate. Follow [GitLab setup](docs/GITLAB_CI.md); a manual
button without protected environments/permissions is insufficient.

## Commands

| Command | Behavior |
|---|---|
| `analyze-mr` | Advisory MR analysis; optional configured GitLab comment |
| `predict` | Analyze unreleased changes and report the calculated version |
| `prepare-release` | Verify branch/tags, analyze, generate a validated candidate |
| `publish-release` | Validate approved candidate, reject staleness, publish/recover |
| `version` | Show tool build, current stable tag and checkout commit |
| `version --build` | Print build version without configuration or network |
| `validate-config` | Validate strict YAML without credentials/network |

Flags follow the command: `--config`, `--output`, `--json`, `--verbose`, `--dry-run`.
`--output` is the candidate path for prepare/publish, and a report path otherwise.
Dry runs still perform analysis/validation but do not write files, comments, tags
or releases. Exit codes: `0` success/no release, `1` operational/validation error,
`2` CLI usage error. Diagnostics go to stderr as JSON; reports go to stdout/files.

## Version policy

`BREAKING > FEATURE > FIX/PERFORMANCE > NONE`. Before 1.0, breaking changes bump
minor by default (configurable to major). No prior stable tag uses `initial_version`
exactly, default `0.1.0`. Docs, tests, chores and refactors default to no release.
Unknown/uncertain classifications, low confidence, incomplete history or truncated
context stop preparation. Stable tags only; prerelease channels are not implemented.

## Docker

```sh
docker build -t semver-ai:local .
docker run --rm semver-ai:local version --build
docker run --rm --user "$(id -u):$(id -g)" \
  -v "$PWD:/workspace" -e SEMVER_AI_API_KEY -e GITLAB_TOKEN -e SEMVER_PROJECT \
  semver-ai:local prepare-release --json
```

The image runs as a non-root user and contains Git/CA certificates. Match your
workspace UID/GID when bind-mounting a writable repository. Published images are
`ghcr.io/ehsanmsb/semver-ai:<version>` for Linux amd64/arm64.

## Development and releases of this tool

```sh
make fmt
make test lint build validate
```

Tests use temporary Git repositories and local HTTP servers; no real service
credentials are needed. GitHub Actions runs checks and a Docker smoke test before
semantic-release. On `main`, Conventional Commits create version tags/GitHub Releases,
four Go binaries (Linux/macOS, amd64/arm64), SHA-256 checksums and GHCR images.
`feat:` bumps minor, `fix:`/`perf:` patch, `!` or `BREAKING CHANGE:` major. Docs/chore
commits alone do not publish. The first tool release is `1.0.0` under semantic-release's
initial-release behavior. This is independent of the `0.1.0` default for managed projects.
No GitLab publication approval policy is bypassed by this GitHub workflow.

## Documentation and status

- [Implementation plan and verification ledger](docs/PLAN.md)
- [Architecture specification](docs/ARCHITECTURE.md) and [implementation mapping](docs/IMPLEMENTATION.md)
- [Configuration](docs/CONFIGURATION.md)
- [GitLab CI and permissions](docs/GITLAB_CI.md)
- [Security and trust boundaries](docs/SECURITY.md)
- [Development and release operations](docs/DEVELOPMENT.md)

The implementation has mock-service and real-Git tests. Production readiness also
requires testing your chosen AI model/endpoint and your GitLab authorization setup.
No claim is made that model classifications or protected-environment settings have
been verified against your live services.
