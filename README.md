<div align="center">

<img src="./web/internal/api/static/logo/logo.svg" width="208">

![GitHub release](https://img.shields.io/github/v/release/marcuwynu23/envious)
![Go version](https://img.shields.io/badge/Go-1.21%2B%20%7C%201.25%2B-00ADD8?logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-003B57?logo=sqlite&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?logo=postgresql&logoColor=white)
![Echo](https://img.shields.io/badge/Echo-4B32C3)
![License](https://img.shields.io/badge/license-Apache%202.0-blue?logo=apache)
![Release CLI](https://github.com/marcuwynu23/envious/actions/workflows/release-cli.yml/badge.svg)
![Release web Docker](https://github.com/marcuwynu23/envious/actions/workflows/release-web-docker.yml/badge.svg)
![Downloads](https://img.shields.io/github/downloads/marcuwynu23/envious/total)

<strong>Multi-application environment variable manager.</strong>
Stop scattering `.env` files across every service — run one self-hosted server with an admin dashboard and a CLI, and manage every app, environment, and variable (with versioning and optional encryption at rest) from a single place. Starts on a zero-setup local database and grows into PostgreSQL when you need multiple instances.

[Read the CLI guide →](cli/README.md) · [Read the server guide →](web/README.md)

</div>

## Table of Contents

- [What Is Envious?](#what-is-envious)
- [Use Cases](#use-cases)
- [Benefits for Developers](#benefits-for-developers)
- [Advantages Over Other Tools](#advantages-over-other-tools)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Example Output](#example-output)
- [User Guide](USER_GUIDE.md)
- [Development](#development)
- [Architecture](#architecture)

## What Is Envious?

**Envious** is a self-hosted environment variable manager written in Go. One server (`envious-web`: REST API + server-rendered admin dashboard, your choice of database) holds every application, environment, and variable; one CLI (`envious-cli`: Cobra, talks to the API) creates, lists, sets, exports, and imports them.

### What It Does

- **Organizes** — Groups variables under applications → environments (`myapp / dev / DATABASE_URL`), instead of loose `.env` files
- **Versions** — Every `var set` on an existing key bumps a version counter, so updates are traceable
- **Encrypts** — Optional `ENCRYPTION_KEY` enables encryption at rest for stored values
- **Serves** — REST API under `/api` guarded by a single admin API key (bcrypt hash only, `X-API-Key` header)
- **Renders** — Cookie-session admin dashboard: Applications → Environments → Variables, with paginated variable lists and `.env` file import
- **Exports/Imports** — `var export` prints `.env` format; `var import` (CLI) and dashboard upload read it back (skips blanks/`#`, splits on first `=`)
- **Ships** — Version-stamped CLI binaries for Linux, macOS, Windows plus a multi-arch Docker image via GitHub Actions releases

### What It Does NOT Do

- **Does not sync secrets to third parties** — Storage is yours: a local database file or your own PostgreSQL. No cloud account required
- **Does not do per-user access control** — A single admin API key guards the whole API; every holder has full access
- **Does not rotate secrets for you** — It stores and serves values; rotation happens by setting new values
- **Does not execute your app** — It manages config; your application still reads its own environment

### Why Use It?

| Problem                                              | How Envious Solves It                                                              |
| ---------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `.env` files scattered across servers and laptops    | One server holds every app/env/var; CLI and dashboard read/write from anywhere     |
| No history of who changed what                       | Version counter on every variable update                                           |
| Plaintext secrets on disk                            | Optional `ENCRYPTION_KEY` encrypts values at rest                                  |
| Onboarding means copy-pasting secrets over chat      | New dev runs `login` + `var export <env>` and gets the exact working `.env`        |
| Staging vs production drift                          | Environments are first-class (`dev`, `staging`, `prod` per app)                    |
| Hand-rolled secret sharing for side projects         | Single binary + embedded database; runs on localhost or a small VPS               |

### The Philosophy

1. **Minimal setup, maximum value.** `go run ./cmd/server`, copy the printed key, `login` — managing variables in under a minute. No config file required on the server.
2. **Your process stays yours.** Works as local binaries, Docker containers, or release assets — no SaaS dependency.
3. **Boring technology.** A familiar relational store, single admin key, plain `.env` format. Easy to back up, easy to reason about.

## Use Cases

| Scenario                  | How Envious Helps                                                                                  |
| ------------------------- | -------------------------------------------------------------------------------------------------- |
| **Side-project secrets**  | One server tracks every project's dev/prod variables instead of a folder of `.env.backup` files    |
| **Team onboarding**       | Teammates `login` once, then `var export` the environment they need — no secret sprawl in chat     |
| **Multi-env deployments** | `dev`/`staging`/`prod` environments per app keep promotion explicit (`export` from one, `import`) |
| **Homelab services**      | One database to back up with everything else; dashboard edits beat SSH + `vim .env`            |
| **CI/CD config**          | Pipelines fetch the environment via the CLI or API and write a `.env` at deploy time               |

## Benefits for Developers

- **Single binary per piece** — `envious-server` and `envious` CLI, both pure Go with minimal dependencies
- **Single-file option** — Start on the embedded database: backups are a file copy, no separate database to operate; switch to PostgreSQL when you scale
- **Zero-config server start** — Sensible defaults (`:8080`, `./envious.db`); only `ENCRYPTION_KEY` is worth setting
- **Familiar `.env` round-trip** — `export` produces what `import` (and your app) consumes
- **Strict, helpful errors** — Invalid ids rejected client- and server-side; missing deletes return `404`, not false success
- **Cross-platform** — Ships as Linux (amd64+arm64), macOS (Intel + Apple Silicon), and Windows binaries, plus a Docker image
- **Tested** — API, auth, storage, and CLI suites run in CI before every release build

## Advantages Over Other Tools

| Aspect                  | Envious                          | Scattered `.env` files | HashiCorp Vault | Hosted managers (Doppler, etc.) |
| ----------------------- | -------------------------------- | ---------------------- | --------------- | ------------------------------- |
| **Setup time**          | ~1 minute                        | Zero (then chaos)      | Minutes–hours   | Minutes + account               |
| **Self-hosted**         | Yes                              | Yes                    | Yes             | No                              |
| **Infra to operate**    | 1 binary (+ optional Postgres)   | None                   | Cluster know-how| None (SaaS)                     |
| **Multi-app / multi-env** | Yes, first-class               | Manual folders         | Yes             | Yes                             |
| **Versioning**          | Yes (per-variable counter)       | No (git-hack it)       | Yes             | Yes                             |
| **Encryption at rest**  | Yes (optional key)               | No                     | Yes             | Yes                             |
| **Admin dashboard**     | Yes (built in)                   | No                     | Yes (complex)   | Yes                             |
| **Offline capable**     | Yes                              | Yes                    | Yes             | No                              |
| **Per-user ACLs**       | No (single admin key)            | N/A                    | Yes             | Yes                             |
| **Secret rotation**     | Manual (`set`)                   | Manual                 | Dynamic engines | Varies                          |
| **License**             | Apache 2.0                       | N/A                    | BUSL            | Proprietary                     |

## Installation

### From Binary (Windows, macOS, Linux)

Download the latest release from [GitHub Releases](https://github.com/marcuwynu23/envious/releases). CLI assets are named `envious-<os>-<arch>` (`envious-windows-amd64.exe` on Windows).

### From Source (Go)

```bash
# CLI
cd cli
go mod tidy
go build -o envious .

# Server
cd ../web
go mod tidy
go run ./cmd/server
```

### Build from Repository (version-stamped CLI)

```bash
cd cli
make release-build
```

The binary lands in `dist/envious` (`dist/envious.exe` on Windows) with `Version`/`Commit`/`BuildDate` baked in. Plain `make build` gives an unstamped dev binary. Set an explicit version with `make build VERSION=v1.0.0` (same in `web/`), then `./envious version` shows it; the server reports it via `-version`, `GET /api/version`, the dashboard header/footer and the `/about` page, and the startup log. Unstamped server builds read the live `git describe` tag instead of `dev`, and `/about` re-reads it on every visit.

### Docker / Podman

```bash
docker pull ghcr.io/marcuwynu23/envious-web:latest
```

Or build the server locally:

```bash
cd web
docker compose up --build
```

### Verify

```bash
./envious --help
./envious version
```

## Quick Start

```bash
# 1. Start the server (first run prints the admin key once — save it)
cd web
go run ./cmd/server
# Envious initial API key (store it securely): <KEY>

# 2. Log the CLI in (new terminal)
cd ../cli
./envious login --api-key=<KEY> --api-base=http://127.0.0.1:8080
# login saved

# 3. Manage an app, an environment, and its variables
./envious app create myapp
./envious app list
./envious env create dev --app-id=2
./envious env list --app-id=2
./envious var set 10 DATABASE_URL "postgres://..."
./envious var list 10
./envious var export 10 > .env
./envious var import 10 .env
```

Prefer the dashboard? Open `http://localhost:8080/`, log in with the API key, and follow Applications → Environments → Variables.

## CLI & Configuration

Daily commands — the full reference lives in the **[User Guide](USER_GUIDE.md)**:

```bash
./envious login --api-key=<KEY> --api-base=http://127.0.0.1:8080
./envious app create billing && ./envious env create prod --app-id=2
./envious var set 10 DATABASE_URL "postgres://..."
./envious var list 10 --show-values
./envious var export 10 > .env
```

Server defaults need no config file: `PORT=8080`, an embedded database at
`./envious.db` (or your own PostgreSQL via one variable), JSON logs to stdout. Everything else — scaling,
rate limits, audit queries, Fluent Bit, Kubernetes — is in the
**[User Guide](USER_GUIDE.md)** (normal way + enterprise way).

## Example Output

```text
$ ./envious var list 10
ID  KEY           VERSION
55  API_KEY       v3
56  DATABASE_URL  v1

$ ./envious var list 10 --show-values
ID  KEY           VERSION  VALUE
55  API_KEY       v3       secret
56  DATABASE_URL  v1       postgres://...

$ ./envious var export 10
API_KEY=secret
DATABASE_URL=postgres://...
```

## Operations & Enterprise

Logs (JSON to stdout, Fluent Bit-ready), the audit trail, Postgres mode,
Docker Compose profiles, Kubernetes manifests, hardening, backups, and
load testing are covered in depth in the **[User Guide](USER_GUIDE.md)**
— start with the normal way, graduate to the enterprise way when you
need multi-instance scale.

## Development

### Prerequisites

| Tool | Version   | Purpose              |
| ---- | --------- | -------------------- |
| Go   | 1.21+ (cli), 1.25+ (web) | Compiler |
| Make | Any       | Build automation     |
| Docker | Any    | Server image (optional) |

### Commands

```bash
# CLI (from cli/)
make build            # dev binary → dist/
make release-build    # version-stamped binary
make test             # go test -v ./...
make test-coverage    # coverage.out + coverage.html

# Server (from web/)
make dev              # go run ./cmd/server
make dev-watch        # air hot-reload (needs .air.toml)
make build            # → dist/envious-server
make test             # go test ./... (first run takes a few minutes — dependency compile)
```

Single tests: `go test -run TestAPIEnvCRUD -v ./internal/api/` (web), full details in [AGENTS.md](AGENTS.md).

### Project Structure

```
envious/
├── cli/                    # envious-cli (Go 1.21+, Cobra) — its own Go module
│   ├── main.go             # entrypoint → cmd.Execute()
│   ├── cmd/                # Cobra commands: root, app, env, vars, login, version/, deps.go
│   ├── internal/
│   │   ├── client/         # HTTP API client (Base URL + X-API-Key, 15s timeout)
│   │   ├── config/         # ~/.envious/config load/save
│   │   ├── service/        # business logic, VersionProvider
│   │   ├── view/           # table / version rendering
│   │   └── model/          # domain models
│   └── test/               # external test packages mirroring cmd/ + internal/
├── web/                    # envious-web (Go 1.25+, Echo + SQL storage) — its own Go module
│   ├── cmd/server/main.go
│   └── internal/
│       ├── api/            # Echo server, REST + admin handlers, templates/, static/
│       ├── storage/        # SQL repository, same API on SQLite and PostgreSQL
│       ├── auth/           # API key init/verify (bcrypt)
│       ├── middleware/     # Logging, Recovery, APIKeyAuth
│       ├── config/         # PORT, DATABASE_PATH, ENCRYPTION_KEY, LOG_LEVEL
│       └── env/            # env models
├── .github/workflows/      # release-cli.yml, release-web-docker.yml
├── AGENTS.md               # contributor/agent operating manual
├── USER_GUIDE.md           # full operator manual (normal + enterprise)
├── CHANGELOG.md | CONTRIBUTING.md | CODE_OF_CONDUCT.md | LICENSE
```

## Architecture

- **Entrypoint** (`web/cmd/server/main.go`) — Loads env config, opens the database, ensures the admin key, serves Echo; graceful shutdown on `SIGINT`/`SIGTERM`
- **Layered API** (`web/internal/api/server.go` → `storage/`) — Handlers do HTTP only (validation at the boundary, `ErrNotFound→404` / `ErrDuplicateKey→409`); all database work lives in the repository
- **Repository** (`web/internal/storage/`) — Apps, environments, variables, and API-key queries with sentinel errors; AES value encryption when `ENCRYPTION_KEY` is set
- **Auth** (`web/internal/auth/`) — 32-byte random key from `crypto/rand`, bcrypt hash storage, constant-time compare; middleware (`APIKeyAuth`) guards `/api`, HMAC-signed cookie sessions guard the dashboard
- **CLI** (`cli/cmd/` + `internal/`) — Cobra commands wire thinly to `client/`; `service/` holds pure logic, `view/` renders tables; shared `loadClient()`/`parseID()` helpers keep errors strict and safe
- **Releases** (`.github/workflows/`) — Tags (`v*`) run vet+tests, then publish the GHCR image and the five CLI binaries with checksums

---

## License

[Apache 2.0](LICENSE) — Copyright (c) 2026 Mark Wayne Menorca

A permissive license that grants you the freedom to use, modify, distribute, and sell the software, provided you include the original copyright notice. It also includes an express grant of patent rights from contributors.

Happy Coding! 🚀
