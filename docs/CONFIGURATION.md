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
