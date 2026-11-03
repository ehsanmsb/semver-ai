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

## Incomplete or refused AI responses

`ai.max_completion_tokens` defaults to 4096 (valid range 1..131072, subject to the
selected provider/model's own limits). This is separate from the input/context
budget. For reasoning models the completion budget includes both reasoning and
visible JSON output. If `finish_reason=length` is reported, raise this setting
explicitly, for example to 16384, or reduce the analyzed scope. Higher limits can
increase cost/latency; `ai.timeout` remains configurable (default 120s, max 10m).
There is no automatic paid retry or silent acceptance of partial JSON.

Errors now distinguish length exhaustion, refusal, content filtering, unsupported
finish reasons, tool calls and empty content. Length errors include the configured
limit and numeric completion/reasoning usage when returned by the provider. Raw
response bodies, refusal text and unknown finish-reason strings are not logged.
The old v1.0.0 message `AI response refused or incomplete` cannot identify which
of these happened; rerun with the updated image to obtain evidence.

See [the API definition of max_completion_tokens](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create).

The classification schema is included in the system prompt as well as
`response_format.json_schema` for compatible gateways that do not enforce the
response format. Local validation remains mandatory: responses such as
`{"files": [...]}` are rejected rather than reinterpreted as valid findings.
