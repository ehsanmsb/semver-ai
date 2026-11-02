# Development

Go 1.26+, Git, and Docker (for image validation) are required. Use `make fmt`,
`make test`, `make lint`, `make build`, and `make validate`. Default tests use local
fixtures and HTTP test servers and never need GitLab or AI credentials.

Use Conventional Commits: `feat:`, `fix:`, `perf:`, `docs:`, `test:`, `ci:`.
Update `docs/PLAN.md` and affected operational docs with every milestone.
Commit identity: `ehsanmsb <ehsan.mosaieby@gmail.com>`.

## Dependencies

Masterminds/semver v3 provides strict parsing and comparisons. The official
GitLab client v3 provides maintained API types and transport. YAML v3 from
`go.yaml.in` provides known-field decoding. Everything else uses the standard
library. `go.sum` locks Go modules; `package-lock.json` locks development-only
semantic-release tooling. Node is not part of the runtime image.

## GitHub workflow

`.github/workflows/release.yml` follows the mkdocs-mcp release pattern with Go
validation instead of Python. PRs have read-only permissions and never publish.
The release job alone receives contents/packages write permissions. Git identity
variables are set to the repository owner's requested name/email; API actions use
GitHub's built-in workflow token. No PAT is required.

Enable Actions and allow the workflow token to create repository releases and
GHCR packages. Make the GHCR package public if anonymous GitLab runners need to
pull it. Otherwise configure registry credentials in GitLab. A private package
cannot be pulled anonymously just because its repository is public.

A release builds four static binaries plus checksums and a multi-architecture
image before creating its tag. If preparation fails, fix the problem and rerun.
If the GitHub release upload fails after the tag exists, inspect the semantic-release
logs and GitHub release before recovery; do not blindly delete or move a published
tag. `workflow_dispatch` permits reruns, but is not an automatic repair of a tagged
release with missing assets. Image aliases may have advanced before a later
GitHub API failure; use immutable version tags/digests in consuming projects.

Run `npm ci --ignore-scripts` to install release tooling locally if needed.
Do not run a real semantic-release against main merely to validate configuration:
it publishes. Use workflow checks or a dry run with a read-only testing context.
