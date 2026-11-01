# Development

Go 1.26+, Git, and Docker (for image validation) are required. Use `make fmt`,
`make test`, `make lint`, `make build`, and `make validate`. Default tests use local
fixtures and HTTP test servers and never need GitLab or AI credentials.

Use Conventional Commits: `feat:`, `fix:`, `perf:`, `docs:`, `test:`, `ci:`.
Update `docs/PLAN.md` and affected operational docs with every milestone.
Commit identity: `ehsanmsb <ehsan.mosaieby@gmail.com>`.
