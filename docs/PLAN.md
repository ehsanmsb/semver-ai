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
| P4 | Prediction, MR analysis, candidates, freshness checks, idempotent publish | P3 | Pending |
| P5 | GitLab template, Docker, GitHub semantic releases, operational docs, checks | P4 | Pending |

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

Commit hashes can be obtained with `git log --oneline -- docs/PLAN.md`; the table
uses exact subjects so a commit never needs to contain its own hash.

## Deferred beyond MVP

Monorepos, dashboards, additional AI providers, databases, unattended GitLab
publication, and deployment automation are outside the supplied MVP scope.
