# Implementation plan

Started: 2026-11-02 (Asia/Tehran). Owner: ehsanmsb.

## Assessment

The local and remote repositories were empty (unborn `main`). There was no code,
dependency manifest, CI, or existing work to preserve. The supplied architecture
and implementation prompt are preserved alongside this plan. The GitHub release
workflow will follow the conventional-commit/semantic-release pattern in mkdocs-mcp.

The product targets GitLab; GitHub Actions releases this tool itself. These are
separate lifecycles: GitLab publication always requires authorized human approval.

## Milestones

| Phase | Deliverable / acceptance | Dependencies | State |
|---|---|---|---|
| P0 | Preserve specifications, assessment, execution ledger | — | Complete |
| P1 | Go CLI, strict YAML config, defaults and validation, build tooling | P0 | Complete |
| P2 | Deterministic SemVer, classifications, real Git history/tag/diff tests | P1 | Complete |
| P3 | Validated AI classification, redaction, bounded context; GitLab adapter | P2 | Complete |
| P4 | Prediction, MR analysis, candidates, freshness checks, idempotent publish | P3 | Complete |
| P5 | GitLab template, Docker, GitHub semantic releases, operational docs, checks | P4 | Complete |

Each implementation commit updates this file and relevant operational docs.
Completed rows mean implementation and tests exist and were run. External checks
are recorded separately; mock tests are not evidence of a live deployment.

## Decisions

- Use Go's standard CLI/logging/HTTP/process packages; no application framework.
- Use Masterminds/semver for strict SemVer and the official GitLab Go client.
- Keep source control, AI, version policy, and release lifecycle separated.
- Fail closed on incomplete history, uncertain analysis, or stale candidates.
- Stable releases only for the MVP; prerelease channels are a later extension.
- Use one local checkout, small Conventional Commits, and the requested Git identity.
- Publish initial work to the empty `main`; no existing branch needs protection.

## Execution ledger

| Date | Phase / commit subject | Verification | Outcome |
|---|---|---|---|
| 2026-11-02 | P0 — `docs: record architecture and implementation plan` | Empty repository and remote inspected; GitHub login verified as ehsanmsb | Specifications and plan recorded |
| 2026-11-02 | P1 — `feat: bootstrap Go CLI and strict configuration` | go test ./..., go vet ./..., go build ./..., example validation | Passed; Docker daemon is not running locally |
| 2026-11-02 | P2 — `feat: add deterministic version engine and Git history analysis` | SemVer table tests; temporary Git repositories, annotated/unreachable tags and shallow clone tests; go vet | Passed |
| 2026-11-02 | P3 — `feat: integrate validated AI analysis and GitLab API` | go test ./...; mocked HTTP contracts, pagination, redaction, malformed/uncertain analysis, chunk aggregation | Passed after adapting to official client v3 release commit value type |
| 2026-11-02 | P4 — `feat: implement approved release preparation and retry-safe publication` | go test -race ./..., go vet ./..., go build ./...; lifecycle and CLI tests | Passed: tampering/staleness rejected, partial publication recovered, retries/concurrency idempotent |
| 2026-11-02 | P5 — `ci: automate semantic releases and document deployment` | Full race tests, vet, binary build, example config, actionlint, shell/JSON checks; initial Docker build passed | Passed locally and on GitHub; v1.0.0 and multi-architecture image published |
| 2026-11-02 | Verification — `docs: record successful CI and first release [skip ci]` | GitHub run, release assets, GHCR manifest and clean local checkout checked | P0–P5 implementation complete; live consuming-project setup remains below |

Commit hashes can be obtained with `git log --oneline -- docs/PLAN.md`; the table
uses exact subjects so a commit never needs to contain its own hash.

## Verification results (2026-11-02)

- `go test -race ./...`: passed, including real-Git fixtures, HTTP test servers,
  complete CLI preparation/publication, no-release cleanup and concurrency/recovery.
- `go vet ./...`, `go build`, example config validation: passed.
- `actionlint` v1.7.11, release script syntax and JSON configuration: passed.
- npm lockfile: all resolved packages use public `registry.npmjs.org`.
- Final local Docker build (Linux arm64): passed. `version --build`, config validation,
  and repository metadata with a read-only mount/non-root user: passed.
- GitHub run: [41396329119](https://github.com/ehsanmsb/semver-ai/actions/runs/41396329119), **successful** (test and release jobs).
- All implementation commits are authored by `ehsanmsb <ehsan.mosaieby@gmail.com>`
  and pushed to `main` using the authenticated `ehsanmsb` account.

## Published release

- [v1.0.0](https://github.com/ehsanmsb/semver-ai/releases/tag/v1.0.0), generated
  automatically by semantic-release from the initial Conventional Commits.
- Assets: Linux/macOS binaries for amd64/arm64 and `SHA256SUMS` (all verified present).
- Image: `ghcr.io/ehsanmsb/semver-ai:1.0.0` with verified Linux amd64/arm64 manifests.
- Image index digest: `sha256:fc3af04e4c6e51231cb5ab87b11fb8d6e33a77c9f81ea0ce21952617c3421064`.
- Local checkout fetched the published `v1.0.0` tag.

The final documentation-only verification commit uses `[skip ci]` to avoid
rebuilding unchanged code; the successful workflow above tested and published the
final implementation at `ff393843bb2e80eda94c7313b229b3a41e44076e`.

## Remaining operator validation

- Configure an approved AI model/endpoint and verify classification quality on a staging project.
- Configure GitLab protected environment/tag permissions, scoped tokens, and CI Lint.
- Run a staging release with an authorized approver and a denied-approver test.
- Per-entry MR provenance for post-merge notes is a follow-up; notes currently use validated findings without invented links.

## Deferred beyond MVP

Monorepos, dashboards, additional AI providers, databases, unattended GitLab
publication, and deployment automation are outside the supplied MVP scope.

## 2026-11-03 — AI completion failure diagnostics

GitLab MR pipeline reached the AI provider, but v1.0.0 reported only
`AI response refused or incomplete`. The log does not distinguish refusal from
length exhaustion; root cause remains unconfirmed without the provider finish reason.

- Add safe finish-reason diagnostics and optional numeric token usage.
- Make the previously fixed 4096 completion-token budget configurable, keeping the default.
- Preserve strict rejection of partial/refused responses; no automatic AI retries.
- Commit: `fix: diagnose incomplete AI responses and configure completion budget`.
- Verification: go test -race ./..., go vet ./..., and binary build passed. Tests distinguish truncation/refusal/filter/tool calls/empty or unsupported responses and verify configurable budget forwarding without leaking provider content.
- User endpoint/model: internal gateway, kimi/kimi-k3.1-pro. Live finish reason remains unknown; no model compatibility assumption made.
- Release v1.0.1 published; GitHub run 41589508144 passed test and release jobs.

### Gateway schema compatibility follow-up

A live test on the Python sandbox returned finish_reason=stop and valid JSON,
but used the wrong top-level key (`files` instead of `changes`). The strict
validator correctly rejected it. Sending response_format.json_schema alone did
not produce a schema-conforming response on this gateway route.

- Include the same JSON Schema explicitly in the system prompt as well as the
  API response_format; retain strict local validation and no automatic retries.
- Commit: `fix: include classification schema in gateway prompts`.
- Full race tests, go vet and build passed. Live CLI prediction on the sandbox passed after explicit-schema prompting: 12 validated findings in about 130 seconds, using a 16384 completion budget and 300s timeout. No credentials or response bodies persisted.
- Original v1.0.0 pipeline finish reason remains unknown; the observed schema mismatch and latency were reproduced separately. No tag/release was created during live tests.

- Published [v1.0.2](https://github.com/ehsanmsb/semver-ai/releases/tag/v1.0.2); [workflow 41590427241](https://github.com/ehsanmsb/semver-ai/actions/runs/41590427241) passed test and release jobs. Linux amd64/arm64 image manifests verified.
- GitLab sandbox branch updated to image 1.0.2 with default completion budget 16384 and timeout 300s; 11 Python tests passed. A new MR pipeline and approved staging publication remain operator validation.
