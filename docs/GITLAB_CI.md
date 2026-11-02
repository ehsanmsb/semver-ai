# GitLab CI integration

The example in `examples/gitlab-ci.yml` is for a trusted project on GitLab 18.4+
Premium/Ultimate with protected environments. Older releases may need changes to
`artifacts:access: maintainer`. Validate the merged pipeline in your instance's
CI Lint before enabling it. No live GitLab instance was configured by this project.

## Required setup

1. Commit `.release-manager.yml` to the consuming project (the tool's own checkout
   ignores its local config). Set the release branch to its protected default
   branch, select an AI model, and set the self-managed GitLab API URL if applicable.
2. Protect the release branch and tag pattern (`v*` by default). Restrict tag creation
   to the publication token's dedicated bot/role and release maintainers.
3. Protect the `production-release` environment. Select explicit **Allowed to deploy**
   users/groups and approval rules. Disable self-approval where your instance allows
   it. The manual button alone is not an authorization boundary.
4. Add masked CI variables below. Scope the publication token to
   `production-release`, mark it protected, and never expose it to analysis jobs.
5. Restrict candidate artifact access to maintainers; keep the seven-day expiry.
   Download and review the candidate's evidence, calculated version and notes before
   approving. A checksum detects corruption, not malicious replacement by a writer.
6. Keep `resource_group: semver-release-publication` on every publication path in
   the project. The job acquires the lock only when it runs after approval.
7. Pin the tool image to a reviewed release/digest in production. `:1` in the example
   follows compatible tool updates; it is published by this repository's workflow.

| Variable | Use / permission |
|---|---|
| `SEMVER_AI_API_KEY` | Credential for the approved AI endpoint; analysis/preparation only |
| `SEMVER_GITLAB_READ_TOKEN` | Project-scoped `read_api` token for metadata/tags; mapped to `GITLAB_TOKEN` |
| `SEMVER_GITLAB_PUBLISH_TOKEN` | Dedicated project-scoped `api` token with the minimum role allowed to create protected tags/releases; environment-scoped and protected |
| `CI_PROJECT_ID`, `CI_COMMIT_SHA`, `CI_COMMIT_BRANCH` | Built-in GitLab project/checkout identity |
| `CI_MERGE_REQUEST_IID` | Built-in MR identifier for advisory analysis |

`create_mr_comment` defaults to false. Enabling it requires an `api`-scoped analysis
credential capable of posting notes; use a separate bot, never the publication
credential. Comments are advisory and may be repeated when a job is retried.

Fork MRs are excluded from the example analysis job. Same-project MR code is also
untrusted: do not grant secrets to contributors who can alter CI configuration.
Protected variables are usually unavailable to unprotected MR refs. Use a trusted,
reviewed analysis pipeline or an approved runner/secrets policy; do not unprotect a
publication token to make an MR job pass. Local tests require no credentials.

## Lifecycle and failure behavior

- Full Git history and tags are required (`GIT_DEPTH: 0`, fetch tags). Missing MR
  head/base objects fail explicitly; fetch the MR ref if using a custom checkout.
- `analyze-mr` is advisory and compares the MR's complete local Git range. It also
  retrieves GitLab diff/commit pages and rejects truncated API diffs.
- `prepare-release` analyzes all changes since the highest reachable stable tag.
  Local tags must agree with GitLab. No release-worthy changes exit successfully
  with `NO_RELEASE`; no candidate is left behind.
- The manual publication job is optional (`allow_failure: true`), so ordinary
  pipelines are not left blocked when no release is needed. Monitor the release job
  itself: a failed optional publication does not fail the whole pipeline.
- Publication checks the current project, configured branch head, base tag commit,
  latest stable tag, configuration, expiry and checkout. A branch advance while
  waiting for approval invalidates the candidate; prepare and approve again.
- A retry after tag creation completes a missing GitLab Release. Existing tag or
  release content must exactly match. Conflicting state fails instead of overwriting.
- Mutating requests are not automatically retried by the HTTP client. The publisher
  reads state after ambiguous failures to discover operations that actually succeeded.
- Prerelease channels and independently versioned packages are outside this MVP.

The CLI checks `GITLAB_CI=true`, `CI_JOB_MANUAL=true`,
`CI_COMMIT_REF_PROTECTED=true`, and `CI_ENVIRONMENT_NAME=production-release`.
These are defense-in-depth checks; GitLab permissions enforce authorization.
GitLab Free lacks the protected-environment gate required by this example. Do not
use a plain manual job as a substitute; use a separately secured approval system
before adapting publication for that tier.

References: [protected environments](https://docs.gitlab.com/ci/environments/protected_environments/),
[resource groups](https://docs.gitlab.com/ci/resource_groups/),
[deployment approvals](https://docs.gitlab.com/ci/environments/deployment_approvals/).
