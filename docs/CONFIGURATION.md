# Configuration

Start with `examples/.release-manager.yml`. YAML fields are strict: unknown keys,
multiple documents, malformed values, invalid URLs and invalid policies fail.
Missing fields use defaults shown in the example; `ai.model` must be supplied.
Secrets are read only from named environment variables at execution time.
`validate-config` does not require secrets or call external services.

MVP restrictions: human approval and GitLab Release creation cannot be disabled;
versions must be strict, stable SemVer. All five notes sections must be present,
but their ordering is configurable. MR comments default to off.
HTTPS is required except loopback test endpoints; URL credentials are rejected.

Version selection uses the highest reachable stable SemVer tag, ignoring unrelated,
prerelease and build-metadata tags. Pre-1.0 breaking changes bump minor by default.
No prior tag uses initial_version exactly; docs-only changes produce no release.
Shallow clones fail; use full history and fetch all tags.

AI endpoints must support Chat Completions with strict `json_schema` output and
`max_completion_tokens`. Provider failures, refusals, incomplete JSON, UNKNOWN,
uncertainty, omitted changed files, fabricated evidence and low confidence fail.
No fallback to commit-message guessing is used.

`max_diff_bytes` bounds the complete diff. Chunks preserve whole files; an
oversized single-file diff fails explicitly. `max_context_tokens` uses a
conservative byte upper bound including serialized request overhead, rather than
a model-specific tokenizer. Metadata and API/schema context are never truncated
silently. Increase limits after reviewing data size if necessary. Git context
includes complete changed `.proto`, `.sql`, OpenAPI and Swagger files at the
analyzed commit. No arbitrary working-tree file is uploaded.

API contracts: [OpenAI structured outputs](https://developers.openai.com/api/docs/guides/structured-outputs?api-mode=chat),
[GitLab Go client](https://gitlab.com/gitlab-org/api/client-go),
[SemVer library](https://github.com/Masterminds/semver).

Preparation requires the local configured release branch to match its remote
head, and local stable release tags to match GitLab. Publication additionally
checks the candidate's exact configuration hash; changing any configuration
requires preparing a fresh candidate. Candidate validity is seven days.
`version --build` reports the executable version without configuration;
`version` shows the current tag and target; `predict` calculates the next version.

`--output` selects the candidate path for prepare/publish and a report file for
other commands. `--dry-run` suppresses files and remote mutations, but still
performs analysis/validation and may incur AI cost. Publication dry runs require
the same protected manual job context. Preparation removes previous candidate
and notes files before attempting a new analysis, preventing stale artifact reuse.
