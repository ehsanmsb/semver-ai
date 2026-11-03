# Implementation mapping

The original architecture and implementation prompt are retained as specifications.
This document describes the actual MVP rather than treating every proposed file as
an implementation requirement.

| Package | Responsibility |
|---|---|
| `cmd/release-manager` | CLI, signals/timeouts, CI environment checks, output/logging |
| `internal/config` | Strict YAML defaults and validation |
| `internal/version` | Strict stable SemVer and deterministic classification policy |
| `internal/git` | Bounded Git subprocesses, reachable tags, complete ranges, API context |
| `internal/ai` | Provider contract and typed findings |
| `internal/ai/openai` | Compatible Chat Completions transport, embedded versioned prompt |
| `internal/analyzer` | Redaction, file chunks, evidence/coverage/confidence validation |
| `internal/scm` | Small release-store port independent of GitLab |
| `internal/gitlab` | Official client adapter, pagination, bounded safe retries |
| `internal/application` | Prediction and preparation orchestration |
| `internal/release` | Candidate schema/checksum/policy, notes, idempotent publisher |
| `internal/safejson` | Strict JSON including duplicate-key detection |

Output formatting stays in the CLI; deterministic notes stay in release. There are
no unused output/template packages or language-specific parser frameworks. The
classification prompt is embedded so a binary/image does not depend on a relative
prompt directory. Notes derive only from validated findings, without another AI call.

MR analysis uses GitLab metadata/diff/commit APIs and the complete local Git diff;
missing Git objects and GitLab collapsed/truncated diffs fail. The prediction uses
the last reachable stable tag at the MR base. Post-merge preparation analyzes the
whole unreleased range; it does not need to map every commit back to an MR. Notes
currently describe findings without inventing MR/issue references. Per-entry MR
provenance across squash/rebase histories is a documented follow-up, not a claimed
feature.

The candidate's SHA-256 is corruption detection, not a signature. Publication
requires a trusted protected GitLab pipeline and validates identity, policy and
freshness again. Failed release creation after tag creation is recoverable.
Concurrent writers must share GitLab's resource group; matching create conflicts
are also safely recognized by the publisher.

## Explicit MVP limits

- Stable versions only; prerelease/build-metadata tags are ignored.
- Chunks preserve complete file diffs; a single oversized file fails rather than
  losing context. Increase the configured budget after reviewing input size.
- Context tokens use a conservative byte bound; completion budget defaults to 4096 tokens and is configurable via `ai.max_completion_tokens` (including reasoning tokens).
- Repository context includes changed protobuf/SQL/OpenAPI/Swagger files; this is
  not a complete language-specific dependency or public API graph.
- Generic secret redaction and prompt boundaries reduce risk but do not prove
  absence of custom secrets or model prompt injection.
- All five release-note sections must be configured, with custom ordering allowed.
- `version` is local metadata; `predict` performs AI analysis for the next version.
- No external service credentials are needed for tests; live model accuracy and
  GitLab permission enforcement require an operator's staging validation.
