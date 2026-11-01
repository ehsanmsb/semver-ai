# Semver AI

Your AI Release Engineer: a Go CLI for AI-assisted, human-approved GitLab releases.
AI classifies changes; Go calculates versions deterministically.

Implementation is tracked in [docs/PLAN.md](docs/PLAN.md). The supplied product
specification is in [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Development

Requires Go 1.26+ and Git. Run `make test lint build validate`.

```sh
cp examples/.release-manager.yml .release-manager.yml
# Set your AI model and GitLab API URL in the file.
go run ./cmd/release-manager validate-config
go run ./cmd/release-manager version
```

Current milestone: CLI foundation and strict configuration validation. See the
plan for implemented commands and remaining release functionality.
