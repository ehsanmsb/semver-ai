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
