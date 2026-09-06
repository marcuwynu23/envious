# AGENTS.md — Envious

> Operating manual for AI agents and human contributors working in this repo.
> Act as a **senior software engineer**: precise, minimal, evidence-backed, security-aware.
> Repo is a Go monorepo with **two independent Go modules**: `web/` (server) and `cli/` (client).

## 1. Repository Map

```
envious/
├── cli/                  # Go 1.21+, Cobra CLI — module `envious-cli`
│   ├── main.go           # entrypoint → cmd.Execute()
│   ├── cmd/              # Cobra commands: root, app, env, vars, login, version/, deps.go
│   ├── internal/
│   │   ├── client/       # HTTP API client (Base URL + X-API-Key)
│   │   ├── config/       # ~/.envious/config load/save
│   │   ├── service/      # business logic, VersionProvider
│   │   ├── view/         # table / version rendering
│   │   └── model/        # domain models
│   ├── test/             # external test packages mirroring cmd/ + internal/
│   ├── Makefile          # build | release-build | test | test-coverage | deps | run | clean
│   └── go.mod
├── web/                  # Go 1.25+, Echo + SQLite — module `envious-web`
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── api/          # Echo Server, REST + admin handlers, templates/, static/
│   │   ├── storage/      # SQLite repository (apps/envs/vars/api_key), sqlite.go
│   │   ├── auth/         # API key init/verify (bcrypt)
│   │   ├── middleware/   # Logging, Recovery, APIKeyAuth
│   │   ├── config/       # PORT, DATABASE_PATH, ENCRYPTION_KEY, LOG_LEVEL
│   │   └── env/          # env models
│   ├── makefile          # dev | dev-watch | start | build | test | deps | clean
│   └── go.mod            # modernc.org/sqlite (pure Go, slow first compile — be patient)
├── .github/              # pull_request_template.md, ISSUE_TEMPLATE/, FUNDING.yml
├── CHANGELOG.md | CONTRIBUTING.md | CODE_OF_CONDUCT.md | README.md
```

**Key facts:**

- No root `go.mod`. Always run Go commands **inside** `web/` or `cli/`.
- API auth: header `X-API-Key`. Server stores only bcrypt hash. First run prints key once to stdout.
- Admin dashboard is cookie-session (`envious_auth` HMAC-signed), not the API key directly.
- Server config via env: `PORT=8080`, `DATABASE_PATH=./envious.db`, `DB_DRIVER=sqlite|postgres`, `DATABASE_URL` (pg only), `DB_MAX_OPEN_CONNS/DB_MAX_IDLE_CONNS`, `RATE_LIMIT_RPS/BURST` (0 disables), `AUTH_CACHE_TTL_SECONDS` (0 disables), `ENCRYPTION_KEY` (optional), `LOG_LEVEL=info`, `LOG_FORMAT=json|text`.
- Audit trail: every mutation + login/logout is stored in `activity_logs` and streamed to stdout logs (`audit=true`); query via `GET /api/activity?action=&limit=`. Never log secret values or keys — metadata only.
- Traffic hardening: WAL + busy_timeout (sqlite), write-conflict retries in `SetVar`, cached bcrypt hash, HTTP timeouts, per-IP rate limit (429), 1MB body cap, `/healthz` (live) + `/readyz` (DB ping). Storage tests run on sqlite always and postgres when `TEST_POSTGRES_URL` is set; compose profiles and `web/k8s/` cover deployment.
- CLI config file: `~/.envious/config` (`api-base` + `api-key`). Set via `envious login --api-key=... --api-base=...`.

## 2. Loop Engineering Workflow (mandatory)

Every task — bugfix, feature, refactor — follows this closed loop. Do not skip steps.

```
1. OBSERVE → 2. PLAN → 3. IMPLEMENT → 4. VERIFY → 5. REFLECT
```

1. **OBSERVE** — Read before writing. Inspect `README.md`, relevant `cmd/` / `internal/` files, existing tests, `git log --oneline -10`, `git status`. Confirm which module(s) are affected (`web`, `cli`, or both).
2. **PLAN** — State: scope, files to touch, design pattern to apply, test cases to add. Keep scope focused; unrelated refactors are rejected. For >3 steps, track with a todo list.
3. **IMPLEMENT** — Smallest diff that solves the problem. Match surrounding style. Follow Sections 4–7.
4. **VERIFY** — Format + vet + build + **run full test suite for every touched module** (Section 5). Never claim "done" without executed test output.
5. **REFLECT** — Check: did I add tests? update docs? follow conventional commits? leave no secrets, dead code, or TODOs?

If verification fails, loop back to step 3 — do not paper over failures.

## 3. Commands (run in module dir)

### web/ (`envious-web`)

```bash
cd web
go mod tidy && go mod download
go run ./cmd/server              # dev (PORT=8080 DATABASE_PATH=./envious.db)
make dev                         # same as above
make dev-watch                   # air hot-reload (needs .air.toml)
make build                       # → dist/envious-server (+ templates copy)
make test                        # go test ./...  — first run compiles modernc sqlite, can take 2-5 min
go test -list '.*' ./...         # cheap test discovery without full run
go test -run TestAPIEnvCRUD -v ./internal/api/
go test -run TestInitAndVerify -v ./internal/auth/
go test -run TestEnvAndVarCRUD -v ./internal/storage/
gofmt -l . && go vet ./...
```

### cli/ (`envious-cli`)

```bash
cd cli
go mod tidy
go build -o envious .            # or: make build  → dist/envious[.exe]
go build -ldflags "-X envious-cli/cmd.Version=$(git describe --tags --always --dirty) -X envious-cli/cmd.Commit=$(git rev-parse --short HEAD)" -o envious .
make release-build               # version-stamped build
make test                        # go test -v ./...
make test-coverage                # → coverage.out + coverage.html
./envious login --api-key=<KEY> --api-base=http://127.0.0.1:8080
./envious app create myapp && ./envious app list
./envious env create dev --app-id=2 && ./envious env list --app-id=2
./envious var set 10 DATABASE_URL "postgres://..." && ./envious var list 10
gofmt -l . && go vet ./...
```

Windows note: use `pwsh`; quote paths with spaces; `Makefile`/`makefile` `rm -rf` targets assume sh — prefer direct `go` commands on Windows if `make` is unavailable.

## 4. Engineering Standards

### 4.1 Go best practices (both modules)

- `gofmt` clean, `go vet` clean. No warnings committed.
- Standard layout: `cmd/` = wiring only, `internal/` = logic. No business logic in `main.go`.
- Errors: wrap with context (`fmt.Errorf("...: %w", err)`), never swallow. Handlers map sentinel errors to HTTP codes:
  - `storage.ErrNotFound` → 404, `storage.ErrDuplicateKey` → 409, validation → 400.
- `context.Context`: always thread `c.Request().Context()` into storage calls. Never `context.Background()` in request path (tests/seeding only).
- Input validation at the boundary (handler/command), not deep in storage.
- IDs: `int64`, parsed via `parseIDParam` / `parseInt64` in web. Validate `> 0` where applicable.
- Timeouts: CLI HTTP client is `15s`. Keep it; never use a client without timeout.
- No `panic` in request path — middleware `Recovery()` is a net, not a strategy.
- No global mutable state except Cobra `rootCmd` and build-time `Version/Commit/BuildDate/Author`.
- Dependencies: minimal. New deps need justification + `go mod tidy`.

### 4.2 Design patterns in use — follow them

| Pattern | Where | Rule |
|---|---|---|
| Layered (Handler → Store → SQLite) | `web/internal/api/server.go` → `storage/` | Handlers do HTTP only; SQL lives only in `storage/` |
| Repository | `web/internal/storage/sqlite.go` | New queries go here with `ErrNotFound` / `ErrDuplicateKey` semantics |
| Middleware chain | `web/internal/middleware/` + `api.Use(APIKeyAuth)` | Auth/logging/recovery stay in middleware, not handlers |
| Template Registry | `api.TemplateRegistry` | Server-rendered admin pages via `templates/*.html` |
| Dependency injection | `cli/cmd/deps.go`, `version.NewCommand(provider, renderer)` | Wire via constructors/factories so tests can inject fakes |
| DTO / View separation | `cli/internal/model` vs `view/` vs `service/` | No printing in `service/`; no HTTP in `view/` |
| Command (Cobra) | `cli/cmd/*.go` | One file per resource (`app.go`, `env.go`, `vars.go`, `login.go`); shared flags on `root.go` only |

Do not introduce singletons, god-objects, or cross-layer imports (e.g. `view` importing `client`).

### 4.3 API conventions (web)

- REST: `GET /api/apps`, `POST /api/apps` (201), `GET/DELETE /api/apps/:id` (200/204), same shape for `/api/envs`, `/api/envs/:id/vars`, `/api/vars/:id`.
- JSON error shape: `{"error": "<msg>"}`. Success shapes stay stable — changing them is breaking.
- Admin routes (`/`, `/apps/:id`, …) are form/redirect based, guarded by `requireSession`. Keep API and admin handlers separate.
- `.env` import parsing: skip blank/`#` lines, split on first `=`, trim key. Preserve this behavior.

### 4.4 CLI conventions

- Cobra structure: `Use`, `Short`, `Long` with `Examples:` block. `SilenceErrors=true, SilenceUsage=true`; `Execute()` prints to stderr + `os.Exit(1)`.
- `RootCmd()` exists for `test/cmd` — do not remove or rename.
- Output goes through `view/` renderers (tables), never ad-hoc `fmt.Printf` in commands except errors/version.
- Flags: `--app-id` (0 = default/all per README), `--env-id`, `--config`, `-v/--verbose`. Keep names stable.

### 4.5 Security — non-negotiable

- Never log, print, or commit API keys, `ENCRYPTION_KEY`, or `~/.envious/config` contents.
- Server stores bcrypt hash only. Comparison via `bcrypt.CompareHashAndPassword`.
- Session cookie: `HttpOnly`, `SameSite=Lax`, HMAC-SHA256 signed. Constant-time compare in `verifySig` — keep it.
- `Secure: false` is dev-only; flag it in any production discussion.
- Validate file uploads (admin `.env` import) by size/type before scanning; existing scanner behavior is the baseline.

## 5. Testing Policy — test all features

> No feature, fix, or refactor is complete without covering tests. This is enforced in review.

- **Where tests live:**
  - `web`: colocated `*_test.go` — `internal/api/api_integration_test.go` (`TestAPIEnvCRUD`), `internal/auth/auth_test.go` (`TestInitAndVerify`), `internal/storage/sqlite_test.go` (`TestEnvAndVarCRUD`).
  - `cli`: external suite under `test/` — `test/cmd/*_test.go` (root, login, app/env/var help), `test/internal/{model,service,view}/*_test.go`.
- **What to add per change:**
  - New endpoint/command → happy path + 400 + 401/404/409 case.
  - New storage query → CRUD round-trip + duplicate + not-found cases (follow `TestEnvAndVarCRUD` table style).
  - Bugfix → regression test reproducing the bug first (red), then fix (green).
  - Use **table-driven tests** with `t.Run` subtests as the default Go style.
- **How to run:** `make test` (or `go test ./...`) in **each touched module**. First `web` compile is slow — allow 300s timeout, use `-list` or `-run` for iteration, full suite before PR.
- **Coverage:** `make test-coverage` (cli) for new logic; don't chase % — cover branches that matter (validation, error mapping, auth).
- **Do not:** commit `coverage.out/html`, `dist/`, `*.db`, `envious_api_key.txt` (all gitignored).

## 6. Git — Conventional Commits (enforced)

History already follows this — keep it. Format: `<type>(<scope>): <subject>` (lowercase subject, imperative, ≤72 chars).

- **Types:** `feat` (new), `fix`, `docs`, `style` (formatting), `refactor`, `perf`, `test`, `chore`, `ci`, `revert`.
- **Scopes in use:** `cli`, `web`, `api`, `auth`, `storage`, `ui`, `assets`, `deps`. Reuse them; new scope only if none fits.
- Examples: `feat(api): add variable pagination`, `fix(cli): handle missing api-base`, `test(web): cover duplicate env name`.
- One logical change per commit. Docs-only and code changes stay separate.
- Before commit: `git status`, `git diff`, stage only intended files. Never commit secrets, `.db`, `dist/`, coverage artifacts.
- PRs: fill `.github/pull_request_template.md`; link issue; describe problem, solution, trade-offs, test output. Keep scope focused.

## 7. Documentation & Cleanliness

- Update `README.md` / `cli/README.md` / `web/README.md` when commands, flags, env vars, or flows change.
- `CHANGELOG.md` is generated by `auto-changelog` — do not hand-edit entries; rely on conventional commits.
- Code comments: explain **why**, not what. Exported symbols get doc comments. No commented-out code, no stray TODOs without an issue link.
- Keep diffs minimal: no drive-by reformatting, no unrelated renames. Match naming/formatting of nearby code.

## 8. Agent Guardrails

- **Read-only first:** `Read`/`Glob`/`Grep` before `Edit`. Verify parent dir with `Test-Path` before creating files.
- **Prefer `Edit` over `Write`**; never create docs (`*.md`) unprompted — except this file when asked.
- **No destructive commands:** no `git push --force`, `git reset --hard`, `rm -rf /`, mass deletes, or `Set-Content` via shell. Use dedicated file tools.
- **No credential exfiltration:** never `cat ~/.envious/config`, never echo keys/tokens into logs.
- **Build artifacts:** never commit `dist/`, `coverage.*`, `*.db`.
- **If blocked:** stop, state what evidence you have, what you tried, and ask — do not guess at APIs, URLs, or file paths.

## 9. Quick Triage

| Symptom | Check |
|---|---|
| `401` from CLI | `login --api-base` correct? `X-API-Key` set? hash in DB intact? |
| `web` test hang on first run | normal — `modernc.org/sqlite` compile; wait / raise timeout to 300s |
| `var` 404/400 | wrong env id? `env list --app-id=<id>` to confirm |
| Template change not reflected in `make build` | `dist/templates` copy step — rebuild |
| `Version=dev` in binary | expected for `make build`; use `make release-build` for stamped version |

## 10. Definition of Done

- [ ] Loop followed (observe → plan → implement → verify → reflect)
- [ ] `gofmt` + `go vet` clean in touched modules
- [ ] Tests added/updated for **every** behavior change; `make test` green in `web` and/or `cli` as touched
- [ ] Docs updated if user-facing behavior changed
- [ ] Conventional commit message(s); no secrets or artifacts committed
- [ ] PR body describes what/why/tests per template
