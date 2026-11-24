# Roadmap — envious
> Updated: 2026-11-25
> Sources: status audit, gh issues, comparable-project research

## Current status

Version maturity: latest release is `v2.0.0` (2026-09-06); `v1.0.0` also tagged. On main, 1 commit ahead of `origin/main` (unpushed).
GitHub Projects board: `26 "envious Roadmap"` (marcuwynu23), created 2026-11-25, linked to repository `marcuwynu23/envious` (`gh project link 26 --owner marcuwynu23 --repo marcuwynu23/envious`), with a short `Description` (Multi-application environment variable manager. One self-hosted server…) and the repo `README.md` content as the board `Readme` (verified via `gh project view`), plus all 7 open issues as items; `Queue`/`Priority` fields synced from the queue below (verified via `gh project item-list`).

What works today (evidence-linked):

- Web server (`envious-web`): apps CRUD (`web/internal/api/server.go:124-127,304-359`), envs CRUD incl. nested `GET/POST /api/apps/:id/envs` (`server.go:128-134,361-462`), vars CRUD (`GET/POST /api/envs/:id/vars`, `PUT/DELETE /api/vars/:id`) (`server.go:135-138,464-537`).
- Variable versioning: `variables.version` + `variable_versions` history table (`web/internal/storage/sqlite.go:259,265`); `SetVar` bumps version and writes history (`sqlite.go:832-866`); concurrent-version test `TestConcurrentSetVar` (`web/internal/storage/dialect_test.go:137,176`).
- Audit trail: `LogActivity` (`sqlite.go:1015`), `GET /api/activity` (`server.go:139,284-302`), `s.audit()` writes to DB + structured log (`server.go:184-201`).
- Admin dashboard: templates `apps.html`, `app_envs.html`, `vars.html`, `about.html`, `login.html`, `error.html` (`web/internal/api/templates/`).
- Auth: bcrypt API key, HMAC-SHA256 session cookie `envious_auth` (`server.go:240-271,161-178`); probes `/healthz`, `/readyz` (`server.go:104-105,208-221`).
- Hardening observed: WAL + busy_timeout, write-conflict retries, cached bcrypt hash, 1MB body cap, per-IP rate limit — `web/README.md` key facts; `dialect_test.go`.
- Encryption at rest (AES, `ENCRYPTION_KEY`) — `storage/sqlite.go`; dialect parity (SQLite + PostgreSQL) — `dialect_test.go`, go.mod uses `modernc.org/sqlite`.

Quality gates, observed 2026-11-25:

- `web`: `go test ./...` → `ok envious-web/internal/api`, `internal/auth`, `internal/storage`, `internal/version`; `go vet ./...` clean; `gofmt -l .` lists 11 files (see Risks).
- `cli`: `go test ./...` → `ok envious-cli/test/cmd`, `test/internal/client`, `test/internal/model`, `test/internal/service`, `test/internal/view`; `go vet ./...` clean; `gofmt -l .` lists 23 files.

## Implemented features

| Feature | Status | Evidence |
| --- | --- | --- |
| Apps CRUD (REST + dashboard) | implemented | `web/internal/api/server.go:124-127,304-359,545-564` |
| Envs CRUD (nested + global) | implemented | `web/internal/api/server.go:128-134,361-462,643-685` |
| Vars CRUD (REST + dashboard) | implemented | `web/internal/api/server.go:135-138,464-537,687-728` |
| `.env` export/import (CLI + dashboard) | implemented | `cli/cmd/vars.go:231-342`, `server.go:730-771` |
| Variable version counter | implemented | `web/internal/storage/sqlite.go:259,832-866` |
| Audit trail + activity query | implemented | `web/internal/storage/sqlite.go:1015,1025`; `server.go:139,184-201` |
| Admin dashboard (apps/envs/vars pages) | implemented | `web/internal/api/templates/*.html` (7 templates) |
| bcrypt API key auth + session cookie | implemented | `web/internal/api/server.go:240-271,161-178`; `internal/auth/auth.go` |
| Encryption at rest | implemented | `web/internal/storage/sqlite.go` (`ENCRYPTION_KEY`) |
| SQLite + PostgreSQL dialects | implemented | `web/internal/storage/dialect_test.go`; go.mod deps |
| Health/readiness probes | implemented | `server.go:104-105,208-221` |

## Production readiness checklist

- [x] Tests run green + coverage baseline — `go test ./...` green in web (api/auth/storage/version) and cli (test/cmd, client, model, service, view); coverage not measured as baseline (not required by repo standards)
- [ ] CI on every push/PR — observed only release workflows (`.github/workflows/release-cli.yml`, `release-web-docker.yml`, tag/`workflow_dispatch`) — no CI on push/PR
- [~] Error handling and logging/observability — audit trail + health/readyz present; no request-level error-rate metrics
- [x] Docs match actual behavior — `README.md`, `USER_GUIDE.md`; known gap: `PUT /api/envs/:id` is a 501 stub (open issue #6)
- [x] Packaging, versioning, changelog, release process — releases `v2.0.0`/`v1.0.0`; `CHANGELOG.md` exists (generated — do not hand-edit)
- [~] Dependency audit / secret scanning — not audited this run (unverified)

## Prioritized queue (Now / Next / Later)

| Item | Issue | Priority | Effort | Status |
| --- | --- | --- | --- | --- |
| Fix `PUT /api/envs/:id` 501 stub (route exists, handler unimplemented) | #6 | P1 | S/M | Queued — see Now |
| Variable version history read + rollback (API + dashboard + CLI) | #4 | P2 | L | Queued |
| API key rotation and auth/session management (API + dashboard + CLI) | #3 | P2 | XL | Queued |
| Activity Logs dashboard page | #1 | P3 | M | Queued |
| CLI activity/audit log command (API `GET /api/activity` exists, no command) | #5 | P3 | S/M | Queued |
| Dashboard `.env` export button | #7 | P3 | M | Queued |
| Multiple users and per-user access control (current model: single admin key by design) | #2 | P3 | XL | Queued |

### Triage rubric (recorded, 2026-11-25)

- **Impact** — breaks users / blocks release / friction / cosmetic
- **Effort** — S (<1d) / M (1–3d) / L (1–2w) / XL (split it)
- **Priority** — P0 security/data-loss · P1 production readiness · P2 core gaps vs comparable projects · P3 enhancements
- Duplicate check: `gh issue list --search "..."` run before proposing new items.
- Every item links to an issue (`#N`); items without an issue are `needs-issue` (none today).

## Research: comparable projects

| Practice | Source | Decision | Why it fits |
| --- | --- | --- | --- |
| Apps→envs (project/environment) model | Infisical — https://infisical.com/blog/best-secret-management-tools | Adopt | Already matches (`apps` → `envs`) |
| CLI-first `.env` round-trip (`var export`/`var import`) | `cli/cmd/vars.go:231-342`, README | Adopt | Already implemented |
| Encryption at rest via operator key | README + `storage/sqlite.go` | Adopt | Already implemented |
| MIT self-host model, projects + environments | Infisical — https://infisical.com/blog/best-secret-management-tools | Adapt | Optional direction; envious stays Apache-2.0 |
| Vault dynamic secrets (short-lived credentials) | https://github.com/hashicorp/vault | Skip | Out of scope for single-server tool |
| Doppler-style multi-target sync | https://encore.dev/articles/doppler-alternatives | Skip | Enterprise scope; out of core roadmap |
| 1Password Secrets Automation model | https://encore.dev/articles/doppler-alternatives | Consider (later) | Depends on issue #3 |
| Multi-user / RBAC | Infisical — https://infisical.com/platform/secrets-management | Defer | Issue #2, queued at P3 |

## Risks / open questions

1. **gofmt dirty in both modules** — web: 11 files listed (e.g. `internal/api/server.go`, `internal/storage/sqlite.go`); cli: 23 files. Fix or document as accepted.
2. **No CI on push/PR** — only release workflows; release notes must be honest about this.
3. **Dashboard gaps vs open issues** — no activity page (#1), no `.env` export button (#7) templates found.
4. ~~GitHub Projects board~~ — addressed: `envious Roadmap` created 2026-11-25 (project 26, owner `marcuwynu23`); previously no board existed for `marcuwynu23/envious` (`gh project list --owner marcuwynu23` listed only unrelated boards: git-policy, podfire, surisc, geosentinel).
5. Dependency audit / secret scanning — unverified this run.
6. Coexistence: server-ahead-by-1 (`git status` shows `main` ahead of `origin/main`) — publish or rebase decision.
