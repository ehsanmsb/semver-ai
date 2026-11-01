# Codex Implementation Prompt — Semver

You are acting as a Senior Go Engineer, Software Architect, and DevOps Engineer.

Your task is to design, implement, test, and document **Semver**, an AI-powered GitLab Release Management CLI.

The project must be implemented in this GitHub repository.

## 1. First: Understand the Project

Before writing code:

1. Inspect the entire repository.
2. Read `docs/ARCHITECTURE.md` carefully.
3. Read all existing project documentation.
4. Inspect the current Go version, dependencies, conventions, and existing implementation.
5. Identify missing requirements and architectural inconsistencies.
6. Check whether an existing implementation can be reused instead of rebuilding it.
7. Identify suitable mature Go libraries for GitLab API access, Git operations, configuration, logging, and SemVer.

Do not assume that the repository is empty.

Do not overwrite existing work without understanding its purpose.

If the repository contains existing code, preserve useful components and refactor only when justified.

## 2. Architecture Requirements

Implement Semver as a modular monolithic Go application.

Follow the architecture described in:

`docs/ARCHITECTURE.md`

Important requirements:

- Go CLI application.
- Docker-based execution.
- Modular and testable architecture.
- Clear separation of application logic, business logic, and external integrations.
- Deterministic SemVer engine.
- AI provider abstraction.
- GitLab API integration.
- Git repository analysis.
- Release candidate generation.
- Human-approved release publication.
- No database or microservices in the MVP.

Avoid unnecessary abstractions and overengineering.

Prefer mature, maintained Go libraries over custom implementations when they satisfy the requirements.

Do not introduce a framework just for the sake of using one.

## 3. Implementation Strategy

Implement the project incrementally.

### Phase 1 — Project Foundation

Implement:

- Go module and CLI entry point.
- Command structure.
- Configuration loader.
- Configuration validation.
- Structured logging.
- Error handling conventions.
- Dockerfile.
- Makefile.
- Basic documentation.
- Unit test structure.

The application must start and validate its configuration before integrating external services.

### Phase 2 — Deterministic Version Engine

Implement the SemVer engine independently from AI.

Requirements:

- Parse valid semantic versions.
- Discover and validate version tags.
- Support configurable tag prefixes.
- Calculate MAJOR, MINOR, PATCH, and NONE.
- Support initial versions.
- Support configurable pre-1.0 behavior.
- Aggregate multiple classifications.
- Define explicit handling of invalid versions and unrelated Git tags.

The engine must be fully unit-tested.

AI must never calculate or directly choose the final version number.

### Phase 3 — Git Repository Integration

Implement:

- Repository inspection.
- Commit history retrieval.
- Git diff retrieval.
- Tag discovery.
- Commit range calculation.
- Unreleased change detection.
- Target commit identification.

Handle shallow clones and insufficient Git history explicitly.

Do not silently analyze an incomplete commit range.

### Phase 4 — GitLab Integration

Implement GitLab API integration.

Required capabilities:

- Authentication.
- MR metadata retrieval.
- MR diff retrieval.
- Commit retrieval.
- MR comment creation.
- Tag creation.
- GitLab Release creation.
- Existing tag and release inspection.

Requirements:

- Use a maintained Go GitLab client if suitable.
- Support self-managed GitLab instances.
- Support configurable API URLs.
- Handle pagination.
- Handle rate limits.
- Apply timeouts.
- Handle transient failures safely.
- Do not expose tokens in logs.

Keep GitLab-specific implementation out of the business logic.

### Phase 5 — AI Provider Integration

Implement an AI provider abstraction.

The first provider must support OpenAI-compatible APIs.

The architecture must allow future implementations of:

- Anthropic.
- Gemini.
- Ollama.
- Self-hosted OpenAI-compatible models.

The analyzer must receive repository and change context and return structured semantic classifications.

Do not tightly couple application logic to a specific AI SDK.

Requirements:

1. Define provider interfaces.
2. Define typed request and response models.
3. Implement prompt loading and versioning.
4. Validate AI responses.
5. Reject malformed JSON.
6. Handle timeouts and provider failures.
7. Support configurable model and endpoint.
8. Redact secrets before requests.
9. Protect against prompt injection.
10. Include evidence for each classification.
11. Support confidence scores.
12. Support large-diff chunking and aggregation.

Never trust AI-generated output without validation.

If the AI provider fails, do not guess a version.

### Phase 6 — Change Analysis

Implement language-agnostic semantic analysis.

Use:

- Git diffs.
- Commit messages.
- MR titles and descriptions.
- Changed file paths.
- Repository context.
- Public API and schema changes.

Required classifications:

- BREAKING
- FEATURE
- FIX
- PERFORMANCE
- REFACTOR
- DOCS
- CHORE
- TEST
- UNKNOWN

Each finding should contain:

- Classification.
- Explanation.
- Evidence.
- Affected files.
- Confidence.
- Potential compatibility impact.

Support configurable classification policies.

The system must not interpret every code change as a feature or bug fix.

### Phase 7 — Release Preparation

Implement:

`release-manager prepare-release`

Requirements:

1. Find the latest valid release tag.
2. Determine all unreleased changes.
3. Analyze all changes in the commit range.
4. Aggregate classifications.
5. Calculate the next version.
6. Generate release notes.
7. Create a release candidate JSON artifact.
8. Return NO_RELEASE if no release-worthy changes exist.

The candidate must contain:

- Schema version.
- Project identity.
- Base tag.
- Target commit.
- Commit range.
- Calculated version.
- Bump classification.
- Analysis results.
- Confidence.
- AI provider and model.
- Release notes.
- Creation timestamp.

Validate the candidate structure before saving it.

### Phase 8 — Release Publication

Implement:

`release-manager publish-release`

Requirements:

- Load the candidate artifact.
- Validate its schema.
- Verify project identity.
- Verify target commit.
- Verify base tag.
- Check whether the candidate is stale.
- Recalculate or reject stale candidates according to documented behavior.
- Create the Git tag.
- Create the GitLab Release.
- Support retries safely.

Important:

The publication process must be idempotent.

If a Git tag exists but the corresponding GitLab Release does not, the next execution must complete the missing operation.

Do not create duplicate tags or releases.

Use GitLab CI `resource_group` for publication concurrency control.

Do not hold a concurrency lock while waiting for manual approval.

Do not assume that a manual CI job automatically provides proper authorization. Document protected environments, protected tags, and approver configuration.

### Phase 9 — GitLab CI/CD Examples

Provide a production-oriented GitLab CI/CD example.

It must demonstrate:

- MR analysis.
- Default branch release preparation.
- Release candidate artifacts.
- Manual publication.
- Protected environment configuration.
- Publication concurrency control.
- Required credentials.
- Appropriate job dependencies.

Clearly document GitLab version and permission assumptions.

Avoid duplicate pipeline execution and unintended release triggers.

### Phase 10 — Documentation and Release

Complete:

- README.md
- docs/ARCHITECTURE.md
- docs/CONFIGURATION.md
- docs/GITLAB_CI.md
- docs/SECURITY.md
- docs/DEVELOPMENT.md
- Example configuration.
- Docker usage.
- CLI usage examples.

Documentation must match the actual implementation.

Do not document features that do not exist.

---

## 4. CLI Interface

Implement these commands:

```bash
release-manager analyze-mr
release-manager predict
release-manager prepare-release
release-manager publish-release
release-manager version
release-manager validate-config
```

Support:

```text
--config
--output
--verbose
--dry-run
--json
```

Use consistent exit codes and helpful error messages.

Commands must be suitable for CI/CD execution.

---

## 5. Configuration

Use `.release-manager.yml`.

Follow the configuration contract in `docs/ARCHITECTURE.md`.

Requirements:

- Validate all required fields.
- Document defaults.
- Validate URLs, version values, timeouts, and policies.
- Reject invalid configuration.
- Never silently ignore invalid settings.
- Load credentials from environment variables.
- Never commit secrets or put them into generated artifacts.

Provide a complete example configuration.

---

## 6. Testing Requirements

Testing is mandatory.

Do not consider a feature complete just because the code compiles.

### Unit tests

Cover:

- SemVer parsing.
- Version calculation.
- Version bump priority.
- Pre-1.0 behavior.
- Initial version.
- Classification aggregation.
- Configuration validation.
- AI response validation.
- Confidence threshold.
- Release candidate validation.
- Release notes generation.
- Idempotency logic.

### Integration tests

Cover:

- Git repository operations.
- Tag discovery.
- Commit range calculation.
- GitLab API client behavior.
- AI provider behavior using mocks or test servers.
- Release preparation.
- Release publication.
- Partial publication recovery.

### Mandatory scenarios

1. Bug fix only → PATCH.
2. Feature and bug fix → MINOR.
3. Breaking change and feature → MAJOR.
4. Documentation-only changes → NO RELEASE.
5. Multiple MRs → aggregate all relevant changes.
6. No previous tag → configured initial version.
7. AI unavailable → fail safely.
8. Invalid AI response → reject.
9. Low confidence → do not silently release.
10. Existing tag without GitLab Release → recover correctly.
11. Repeated publication → idempotent.
12. Concurrent pipelines → no duplicate release.
13. Stale candidate → reject or require re-preparation.
14. Unauthorized release → blocked by the documented GitLab authorization setup.

Do not require real GitLab or AI credentials for the default unit and integration test suite.

Use mocks and test servers where appropriate.

---

## 7. Engineering Standards

Follow these rules throughout the implementation:

- Use idiomatic Go.
- Keep functions focused and small.
- Avoid unnecessary global state.
- Use context.Context for operations involving I/O.
- Handle errors explicitly.
- Wrap errors with useful context.
- Avoid panic for normal error handling.
- Avoid unnecessary dependencies.
- Use structured logging.
- Use deterministic business logic.
- Avoid hidden side effects.
- Make external operations testable.
- Document non-obvious decisions.
- Keep public APIs minimal.
- Validate untrusted input.
- Avoid logging sensitive information.

Do not create abstractions without a clear reason.

Do not implement a custom GitLab API client if an appropriate maintained library exists.

Do not implement a custom SemVer parser if a suitable maintained Go library exists.

---

## 8. Definition of Done

A phase is complete only when:

- Implementation exists.
- Tests are written.
- Tests pass.
- Errors are handled.
- Documentation is updated.
- No unrelated functionality is broken.

Before declaring the project complete:

1. Run formatting.
2. Run static analysis.
3. Run all tests.
4. Build the binary.
5. Build the Docker image.
6. Validate example configuration.
7. Review security-sensitive code.
8. Review release idempotency.
9. Verify documentation against actual CLI behavior.

At minimum, run appropriate equivalents of:

```bash
gofmt -w .
go test ./...
go vet ./...
go build ./...
docker build .
```

If additional tools are available, use appropriate linters and security checks.

Do not claim that a check passed unless it was actually executed successfully.

---

## 9. Working Method

Follow this execution process:

### Step 1 — Repository Assessment

Inspect the repository and architecture document.

Provide a concise report containing:

- Existing repository structure.
- Existing implementation.
- Missing components.
- Architecture concerns.
- Dependency recommendations.
- Proposed implementation phases.

### Step 2 — Implementation Plan

Create a practical implementation checklist.

Identify dependencies between phases.

Do not spend time producing an unnecessarily large theoretical plan.

### Step 3 — Implement

Start implementing the project.

Work in small, logically separated changes.

Do not stop after creating directories, interfaces, or placeholder files.

Each implemented component must have meaningful functionality and tests.

### Step 4 — Verify

Run tests and builds after each meaningful phase.

Fix issues before moving forward.

### Step 5 — Final Report

At the end, report:

- Implemented functionality.
- Repository structure.
- Important architectural decisions.
- Dependencies introduced and why.
- Tests executed and their actual results.
- Docker build result.
- Remaining limitations.
- Features not implemented.
- Required GitLab configuration and permissions.
- Required CI/CD variables.
- How to run the application.

Do not claim full completion if any mandatory requirement remains unfinished.

---

## 10. Important Product Decisions

These decisions are mandatory unless a genuine technical conflict is discovered:

1. Semver is a Go CLI application.
2. GitLab is the initial SCM and release platform.
3. The product supports multiple programming languages.
4. AI classifies semantic changes.
5. Go calculates the final version deterministically.
6. Every release requires human approval in the MVP.
7. A release candidate is a JSON artifact.
8. No database is required.
9. Publication must be idempotent.
10. Stale candidates must not be published.
11. AI failures must fail safely.
12. The architecture must support future AI providers.

If a requirement cannot be implemented exactly as written, explain the limitation and propose the smallest safe alternative.

Do not silently change product behavior.

## Final Instruction

Start by inspecting the repository and reading the architecture document.

Then provide the repository assessment and implementation plan.

After that, proceed with implementation phase by phase.

**Your objective is to deliver a working, tested, documented MVP — not just an architecture proposal or an empty project skeleton.**