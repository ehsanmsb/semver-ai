# Semver — Architecture & Product Specification

**Project:** Semver  
**Tagline:** Your AI Release Engineer  
**Status:** MVP Architecture  
**Primary language:** Go  
**Deployment:** Docker / GitLab CI/CD  
**Initial release strategy:** AI-assisted Semantic Versioning with mandatory human approval

---

## 1. Product Overview

Semver is an AI-powered release management tool that analyzes software changes, predicts version increments, generates release notes, and manages controlled releases through GitLab.

Its primary goal is to automate the reasoning and operational work involved in software releases while keeping version calculation deterministic and release publication under human control.

Semver is designed to support multiple GitLab projects and programming languages.

### Core capabilities

1. Analyze Merge Requests before they are merged.
2. Classify changes based on their semantic impact.
3. Predict the next version and provide evidence for that prediction.
4. Analyze all unreleased changes after they reach the default branch.
5. Calculate the final release version using deterministic SemVer rules.
6. Generate release notes.
7. Create a release candidate artifact.
8. Wait for human approval.
9. Publish a Git tag and GitLab Release after approval.

### Important design principle

AI must never directly decide the final version number.

AI analyzes changes and returns structured semantic classifications. A deterministic Go version engine applies the configured versioning policy and calculates the next version.

---

## 2. Goals and Non-Goals

### MVP Goals

- Multi-project support.
- Multi-language repository analysis.
- GitLab integration.
- AI-based semantic change classification.
- Deterministic SemVer calculation.
- Pre-merge version prediction.
- Post-merge release preparation.
- Human-approved release publication.
- Release notes generation.
- Docker-based execution.
- GitLab CI/CD integration.
- Idempotent release publication.
- No database or external stateful service requirement.

### Out of Scope for MVP

- Monorepo and independently versioned packages.
- Web dashboard.
- Multi-user management.
- Release analytics.
- Automatic release publication without approval.
- Multiple AI providers beyond the initial provider implementation.
- Microservices architecture.
- Persistent database.
- Automatic deployment after release.

These capabilities may be introduced in later versions without compromising the MVP architecture.

---

## 3. High-Level Architecture

Semver should be implemented as a modular monolithic Go application.

The application consists of a single executable with clearly separated internal packages.

```text
                  GitLab CI/CD
                       |
                       v
              Semver CLI Application
                       |
             Application Use Cases
                       |
       +---------------+---------------+
       |               |               |
       v               v               v
   GitLab Client   Git Repository   Configuration
       |               |               |
       +---------------+---------------+
                       |
                       v
                Change Analyzer
                       |
              +--------+--------+
              |                 |
              v                 v
          AI Provider      Context Builder
              |
              v
       Structured Analysis
              |
              v
       Semantic Aggregator
              |
              v
       Deterministic SemVer Engine
              |
       +------+----------------+
       |                       |
       v                       v
 Release Candidate        Release Notes
       |
       v
 Human Approval (GitLab CI)
       |
       v
 Release Publisher
       |
       +------> Git Tag
       |
       +------> GitLab Release
```

### Architectural principles

- Modular monolith, not microservices.
- Clean separation of business logic and external integrations.
- Dependency inversion for AI providers and GitLab clients.
- Deterministic version calculation.
- Explicit configuration and predictable defaults.
- No hidden persistent state.
- Idempotent operations.
- Testable business logic independent of GitLab and AI services.

---

## 4. Application Architecture

Use a lightweight hexagonal architecture.

### Application layer

Contains orchestration and use cases.

```text
internal/application/
    analyze_mr.go
    predict.go
    prepare_release.go
    publish_release.go
```

Responsibilities:

- Coordinate repository analysis.
- Retrieve GitLab information.
- Invoke the analyzer.
- Calculate versions.
- Generate release candidates.
- Publish approved releases.

### GitLab integration

```text
internal/gitlab/
    client.go
    merge_requests.go
    commits.go
    tags.go
    releases.go
```

Responsibilities:

- Authenticate with GitLab.
- Retrieve Merge Requests and metadata.
- Retrieve commit information.
- List and create tags.
- Create and retrieve GitLab Releases.
- Handle pagination, retries, rate limits, and API errors.

Use a mature Go GitLab client library where appropriate instead of implementing the API from scratch.

### Git repository integration

```text
internal/git/
    repository.go
    diff.go
    history.go
```

Responsibilities:

- Inspect repository metadata.
- Retrieve commits and diffs.
- Determine the base tag.
- Identify unreleased changes.
- Compare commit ranges.
- Detect the target commit.
- Support shallow-clone limitations and report insufficient history clearly.

### Analyzer

```text
internal/analyzer/
    analyzer.go
    classifier.go
    context.go
    aggregator.go
    models.go
```

Responsibilities:

- Build analysis context.
- Analyze changed files and diffs.
- Classify changes.
- Aggregate individual findings.
- Preserve evidence and confidence.
- Produce normalized semantic analysis results.

### AI abstraction

```text
internal/ai/
    provider.go
    openai/
```

Define a provider interface that allows future providers to be added without changing application or business logic.

Initial implementation: OpenAI-compatible API.

Future implementations may include Anthropic, Gemini, Ollama, and self-hosted models.

### Version engine

```text
internal/version/
    semver.go
    calculator.go
    policy.go
```

Responsibilities:

- Parse and validate SemVer versions.
- Calculate version increments.
- Apply configurable release policies.
- Handle pre-1.0 versioning.
- Support configurable tag prefixes.
- Handle initial versions when no previous tag exists.

The version engine must not depend on AI.

### Release management

```text
internal/release/
    candidate.go
    notes.go
    publisher.go
```

Responsibilities:

- Create release candidate JSON.
- Generate Markdown release notes.
- Validate candidate integrity.
- Publish tags and GitLab Releases.
- Ensure retry-safe publication.

### Configuration and output

```text
internal/config/
    config.go
    loader.go
    validation.go

internal/output/
    json.go
    markdown.go
```

---

## 5. Semantic Change Classification

AI should classify each meaningful change into one of these categories:

| Classification | Meaning | Default version impact |
|---|---|---|
| BREAKING | Backward-incompatible change | MAJOR |
| FEATURE | New functionality | MINOR |
| FIX | Bug fix | PATCH |
| PERFORMANCE | Performance improvement | Configurable |
| REFACTOR | Internal restructuring | NONE |
| DOCS | Documentation-only changes | NONE |
| CHORE | Maintenance and tooling | NONE |
| TEST | Test-only changes | NONE |
| UNKNOWN | Insufficient evidence | Requires policy handling |

### Version priority

```text
MAJOR > MINOR > PATCH > NONE
```

Examples:

- One bug fix → PATCH.
- One feature and one bug fix → MINOR.
- One breaking change and multiple features → MAJOR.
- Documentation-only changes → NONE.
- Internal refactoring without behavioral changes → NONE.

The default policy should be conservative.

### AI output requirements

The AI must return structured JSON, not free-form text.

Each classification must include:

- Classification.
- Short explanation.
- Evidence from the analyzed change.
- Affected files.
- Confidence score.
- Potential breaking-change details.
- Uncertainty or missing context.

The system must validate the response against a strict schema.

AI-generated claims must be treated as untrusted data until validated.

### Confidence policy

Introduce a configurable confidence threshold.

If the analysis confidence is below the configured threshold:

- Do not silently guess.
- Mark the analysis as uncertain.
- Fail release preparation by default or require an explicit configured override.
- Include the reason in the output.

---

## 6. Multi-Language Analysis

Semver must not be coupled to a single programming language.

Analysis should use multiple information sources:

1. Git diff.
2. Commit messages.
3. Merge Request title.
4. Merge Request description.
5. Changed file paths.
6. Repository structure.
7. Existing public API and schema definitions.
8. Relevant source-code context.

The analyzer should understand common API and compatibility changes, including:

- REST API changes.
- Protobuf and gRPC schema changes.
- OpenAPI changes.
- Database schema changes.
- Public interfaces.
- Configuration format changes.
- Removed or renamed public functionality.

Do not build separate language-specific analyzers for every language in the MVP.

Use a language-agnostic analysis pipeline, with optional repository context and future language-specific extensions.

### Large diff handling

- Set configurable limits for diff size and token usage.
- Split large changes into logical chunks.
- Preserve file and change relationships.
- Aggregate chunk-level findings.
- Avoid losing breaking-change evidence during aggregation.
- Report skipped or truncated content explicitly.

### Security

- Redact secrets and credentials before sending context to AI.
- Never include private keys, access tokens, passwords, or sensitive environment values.
- Treat source code, comments, commit messages, and MR descriptions as untrusted input.
- Protect prompts against prompt injection.
- Never allow repository content to override system-level analysis instructions.

---

## 7. Version Calculation

The version engine must implement deterministic SemVer calculations.

Example:

```text
Current version: 1.4.2

Changes:
- FIX
- FIX

Next version: 1.4.3
```

```text
Current version: 1.4.2

Changes:
- FEATURE
- FIX

Next version: 1.5.0
```

```text
Current version: 1.4.2

Changes:
- BREAKING
- FEATURE

Next version: 2.0.0
```

```text
Current version: 1.4.2

Changes:
- DOCS
- REFACTOR

Result: NO RELEASE
```

### Initial version

If no valid release tag exists:

- Use `initial_version` from configuration.
- Never ask AI to guess an initial version.
- Validate the configured initial version.

### Pre-1.0 behavior

Version behavior before 1.0.0 must be configurable.

For example, the project may choose whether breaking changes before 1.0.0 result in a MINOR or MAJOR increment.

The default behavior must be documented and covered by tests.

### Tag discovery

- Support configurable tag prefixes.
- Find the latest valid SemVer tag.
- Ignore unrelated tags.
- Define behavior for annotated and lightweight tags.
- Handle repositories with insufficient Git history.

---

## 8. Release Lifecycle

### Phase A — Merge Request Analysis

Command:

```bash
release-manager analyze-mr
```

Workflow:

1. Read MR metadata from GitLab CI variables.
2. Retrieve the MR diff and relevant commit information.
3. Build analysis context.
4. Classify changes using AI.
5. Calculate the predicted version using the deterministic engine.
6. Generate a concise analysis report.
7. Optionally post a comment to the Merge Request.

This phase is advisory.

The predicted version may change when other Merge Requests are merged.

### Phase B — Release Preparation

Command:

```bash
release-manager prepare-release
```

Workflow:

1. Verify that the current branch is the configured release branch.
2. Identify the latest valid release tag.
3. Determine the unreleased commit range.
4. Retrieve all relevant commits and changes.
5. Analyze the complete unreleased change set.
6. Aggregate semantic classifications.
7. Calculate the final next version.
8. Generate release notes.
9. Create a release candidate.
10. Save the candidate as a CI artifact.

If there are no release-worthy changes, return a successful NO_RELEASE result and do not create a release candidate.

### Phase C — Human Approval

The release candidate is presented through GitLab CI/CD.

A human must approve the release before publication.

Use GitLab protected environments and authorized approvers where available.

A job configured with `when: manual` alone is not sufficient authorization control.

### Phase D — Release Publication

Command:

```bash
release-manager publish-release
```

Workflow:

1. Load the release candidate artifact.
2. Validate its schema and integrity.
3. Verify that the candidate belongs to the current project.
4. Verify the base tag and target commit.
5. Re-check the current latest release tag.
6. Reject or require re-preparation if the candidate is stale.
7. Acquire the publication concurrency lock through GitLab CI.
8. Create the Git tag.
9. Create the GitLab Release.
10. Verify the final published state.

Do not hold the publication lock while waiting for human approval.

### Idempotency

Publication must be retry-safe.

Example:

- The Git tag was created.
- GitLab Release creation failed.
- The job is retried.

Semver must detect the existing tag and complete the missing GitLab Release operation instead of failing blindly or creating a duplicate release.

---

## 9. Release Candidate Format

The release candidate should be a versioned JSON document.

Suggested artifact:

```text
semver-release-candidate.json
```

Example structure:

```json
{
  "schema_version": "1",
  "project": "group/project",
  "base_tag": "v1.4.2",
  "target_commit": "abc123",
  "commit_range": "v1.4.2..abc123",
  "next_version": "1.5.0",
  "bump": "MINOR",
  "status": "PENDING_APPROVAL",
  "created_at": "2026-11-01T10:00:00Z",
  "analysis": {
    "provider": "openai-compatible",
    "model": "configured-model",
    "confidence": 0.94,
    "changes": []
  },
  "release_notes": "..."
}
```

The real implementation must define complete typed Go structures, validation rules, and schema compatibility.

The candidate should include enough information to detect stale or mismatched artifacts.

Do not use the JSON artifact as a substitute for a database.

---

## 10. Release Notes

Generate Markdown release notes with configurable sections.

Default sections:

- Breaking Changes
- Features
- Bug Fixes
- Performance Improvements
- Other Changes

Each entry should reference its source Merge Request when possible.

Avoid inventing features, issue references, or behavioral changes.

Release notes should be concise and understandable to developers and users.

---

## 11. Configuration

Configuration file:

```text
.release-manager.yml
```

Example:

```yaml
version: 1

release:
  branch: main
  tag_prefix: v
  initial_version: 0.1.0

  pre_1_0:
    breaking_change_bump: minor

  approval:
    required: true

  rules:
    performance: patch
    docs: none
    refactor: none
    chore: none
    tests: none

analysis:
  confidence_threshold: 0.80

  context:
    include_commit_messages: true
    include_mr_description: true
    include_repository_files: true

  limits:
    max_diff_bytes: 500000
    max_context_tokens: 30000

ai:
  provider: openai-compatible
  model: configured-model
  base_url: https://api.openai.com/v1
  api_key_env: SEMVER_AI_API_KEY
  timeout: 120s

gitlab:
  api_url: https://gitlab.example.com/api/v4
  token_env: GITLAB_TOKEN
  create_mr_comment: true
  create_release: true

release_notes:
  sections:
    - breaking_changes
    - features
    - bug_fixes
    - performance
    - other
```

This is an initial configuration contract. Validate all fields, reject invalid configuration, document defaults, and avoid silently ignoring unknown or invalid values.

Secrets must be supplied through environment variables or CI secret variables, never committed to the repository.

---

## 12. CLI Commands

The MVP should expose:

```bash
release-manager analyze-mr
release-manager predict
release-manager prepare-release
release-manager publish-release
release-manager version
release-manager validate-config
```

### Command behavior

- `analyze-mr`: Analyze the current Merge Request.
- `predict`: Print a predicted next version without publishing anything.
- `prepare-release`: Create a release candidate and release notes.
- `publish-release`: Publish a previously prepared and approved candidate.
- `version`: Print current and calculated version information.
- `validate-config`: Validate project configuration.

Support:

- `--config`
- `--output`
- `--verbose`
- `--dry-run`
- `--json`

Use meaningful exit codes for automation.

---

## 13. GitLab CI/CD Integration

Semver is executed from GitLab CI/CD using its Docker image.

Suggested pipeline:

```yaml
stages:
  - analysis
  - release-preparation
  - release-publication

semver-mr-analysis:
  stage: analysis
  image: ghcr.io/OWNER/semver:latest
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
  script:
    - release-manager analyze-mr

semver-prepare-release:
  stage: release-preparation
  image: ghcr.io/OWNER/semver:latest
  rules:
    - if: '$CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH && $CI_PIPELINE_SOURCE == "push"'
  script:
    - release-manager prepare-release
  artifacts:
    paths:
      - semver-release-candidate.json
      - RELEASE_NOTES.md
    expire_in: 7 days

semver-publish-release:
  stage: release-publication
  image: ghcr.io/OWNER/semver:latest
  needs:
    - job: semver-prepare-release
      artifacts: true
  resource_group: semver-release-publication
  environment:
    name: production-release
  rules:
    - if: '$CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH && $CI_PIPELINE_SOURCE == "push"'
      when: manual
  script:
    - release-manager publish-release
```

This is an illustrative template, not a production-ready universal pipeline.

The final implementation must account for:

- GitLab version differences.
- Protected environments and authorized approvers.
- Protected tags.
- Token permissions.
- Artifact expiry.
- Pipeline duplication.
- Manual job behavior.
- Default branch changes.
- Candidate freshness.
- Concurrent pipelines.

Use the least-privilege token required to create tags and releases.

---

## 14. Proposed Repository Structure

```text
semver/
├── cmd/
│   └── release-manager/
│       └── main.go
├── internal/
│   ├── application/
│   ├── analyzer/
│   ├── ai/
│   │   └── openai/
│   ├── config/
│   ├── git/
│   ├── gitlab/
│   ├── release/
│   ├── version/
│   └── output/
├── prompts/
│   ├── classify_changes.md
│   ├── breaking_changes.md
│   └── release_summary.md
├── templates/
│   └── release-notes.md
├── examples/
│   └── .release-manager.yml
├── tests/
│   ├── fixtures/
│   └── integration/
├── docs/
│   ├── ARCHITECTURE.md
│   ├── CONFIGURATION.md
│   ├── GITLAB_CI.md
│   ├── SECURITY.md
│   └── DEVELOPMENT.md
├── Dockerfile
├── Makefile
├── go.mod
├── README.md
└── LICENSE
```

The implementation may adjust package boundaries when justified, but must preserve the architectural responsibilities.

---

## 15. Testing Strategy

### Unit tests

- SemVer parsing and calculation.
- Version bump priorities.
- Pre-1.0 policies.
- Initial version behavior.
- Change aggregation.
- Confidence threshold.
- Configuration validation.
- Candidate validation.
- Release notes generation.

### Integration tests

- Git repository analysis.
- Git tag discovery.
- GitLab API interactions.
- AI provider response parsing.
- Release candidate generation.
- Git tag and GitLab Release publication.

Use mocks or test servers for external services.

Do not require real GitLab or AI credentials for the default test suite.

### Required acceptance scenarios

| Scenario | Expected result |
|---|---|
| Bug fix only | PATCH |
| Feature and bug fix | MINOR |
| Breaking change and feature | MAJOR |
| Documentation-only changes | NO RELEASE |
| Multiple MRs in one release | Aggregate all changes |
| No previous tag | Use configured initial version |
| AI provider unavailable | Fail safely |
| Invalid AI response | Reject response |
| Low confidence | Do not silently release |
| Existing tag, missing GitLab Release | Complete publication |
| Repeated publication | Idempotent result |
| Concurrent release pipelines | No duplicate release |
| Stale candidate | Reject or require preparation again |
| Unauthorized approval | Publication blocked |

---

## 16. Security and Reliability

- Never log secrets or authorization headers.
- Redact sensitive information before AI requests.
- Validate all AI-generated structured output.
- Treat repository content as untrusted.
- Use GitLab protected tags and environments.
- Use least-privilege credentials.
- Avoid writing secrets into artifacts.
- Use bounded HTTP timeouts.
- Implement retry policies for transient API errors.
- Do not retry non-idempotent operations blindly.
- Make release publication recoverable.
- Record enough metadata to audit a release.
- Never publish when analysis or validation fails.

---

## 17. MVP Definition of Done

The MVP is complete when:

1. A Go CLI builds and runs.
2. Docker image builds successfully.
3. Configuration validation works.
4. GitLab MR analysis works.
5. AI classification returns validated structured results.
6. Deterministic SemVer calculation works.
7. Release preparation analyzes all unreleased changes.
8. Release notes are generated.
9. A JSON release candidate is produced.
10. Human approval is enforced through GitLab configuration.
11. Stale candidates are rejected.
12. Release publication is idempotent.
13. Required unit and integration tests pass.
14. GitLab CI/CD examples are documented.
15. README explains installation, configuration, permissions, and usage.

**The MVP should prioritize correctness, security, predictability, and recoverability over feature count.**