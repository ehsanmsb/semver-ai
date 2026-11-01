# Security

Repository content and AI output are untrusted. A fixed embedded, versioned system
prompt separates instructions from JSON repository data. The provider cannot call
tools. Its structured response is validated locally: unknown/duplicate fields,
missing evidence, unknown files, uncertainty and low confidence fail closed.
This reduces prompt-injection risk; it cannot prove an AI classification correct.
Review the candidate and evidence before approving publication.

Known credential values and common private-key, token, password, authorization
and credential-bearing URL patterns are redacted before requests and artifact
output. Pattern matching cannot identify every custom secret format. Never commit
secrets; use your existing secret scanner and an approved AI endpoint. Changes
containing sensitive business data still require an approved provider/data policy.

HTTP errors expose status/operation only, never response bodies, headers or tokens.
Redirects are disabled. GitLab retries bounded read/idempotent requests only;
mutating calls are not blindly retried. AI calls are not automatically retried.
A failed analysis never falls back to an invented version.

Release authorization and artifact integrity are documented with the publication
milestone. A local artifact checksum alone is not an authentication mechanism.

Candidates are schema-versioned, tied to the GitLab API URL/project ID/path,
release branch, exact base/target commits, full configuration hash, and an
expiry of seven days. Publication recalculates the version and release notes,
checks remote branch/tag freshness and verifies final state. SHA-256 detects
accidental corruption; it is not a signature. A writer can recompute it. Trust
comes from protected CI configuration, restricted artifacts, and the human gate.

The CLI requires a manual GitLab job on a protected ref in `production-release`,
matching CI project/commit and local checkout. These environment checks are
additional safeguards, not authentication. Authorized approvers, protected tags,
environment-scoped publication credentials, and a project-wide resource group
must be configured in GitLab. Anyone controlling the runner, CI config or token
can bypass a local program. See GITLAB_CI.md before enabling publication.
