<div align="center">

<img src="./web/internal/api/static/logo/logo.svg" width="208">

![GitHub release](https://img.shields.io/github/v/release/marcuwynu23/envious)
![Go version](https://img.shields.io/badge/Go-1.21%2B%20%7C%201.23%2B-00ADD8?logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-003B57?logo=sqlite&logoColor=white)
![Echo](https://img.shields.io/badge/Echo-4B32C3)
![License](https://img.shields.io/badge/license-MIT-green)
![Release CLI](https://github.com/marcuwynu23/envious/actions/workflows/release-cli.yml/badge.svg)
![Release web Docker](https://github.com/marcuwynu23/envious/actions/workflows/release-web-docker.yml/badge.svg)
![Downloads](https://img.shields.io/github/downloads/marcuwynu23/envious/total)

<strong>Multi-application environment variable manager.</strong>
Stop scattering `.env` files across every service — run one SQLite-backed server with an admin dashboard and a CLI, and manage every app, environment, and variable (with versioning and optional encryption at rest) from a single place.

[Read the CLI guide →](cli/README.md) · [Read the server guide →](web/README.md)

</div>

## Table of Contents

- [What Is Envious?](#what-is-envious)
- [Use Cases](#use-cases)
- [Benefits for Developers](#benefits-for-developers)
- [Advantages Over Other Tools](#advantages-over-other-tools)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [CLI Reference](#cli-reference)
- [Configuration](#configuration)
- [Example Output](#example-output)
- [CI/CD Integration](#cicd-integration)
- [Development](#development)
- [Architecture](#architecture)

## What Is Envious?

**Envious** is a self-hosted environment variable manager written in Go. One server (`envious-web`: REST API + server-rendered admin dashboard, SQLite storage) holds every application, environment, and variable; one CLI (`envious-cli`: Cobra, talks to the API) creates, lists, sets, exports, and imports them.

### What It Does

- **Organizes** — Groups variables under applications → environments (`myapp / dev / DATABASE_URL`), instead of loose `.env` files
- **Versions** — Every `var set` on an existing key bumps a version counter, so updates are traceable
- **Encrypts** — Optional `ENCRYPTION_KEY` enables encryption at rest for stored values
- **Serves** — REST API under `/api` guarded by a single admin API key (bcrypt hash only, `X-API-Key` header)
- **Renders** — Cookie-session admin dashboard: Applications → Environments → Variables, with paginated variable lists and `.env` file import
- **Exports/Imports** — `var export` prints `.env` format; `var import` (CLI) and dashboard upload read it back (skips blanks/`#`, splits on first `=`)
- **Ships** — Version-stamped CLI binaries for Linux, macOS, Windows plus a multi-arch Docker image via GitHub Actions releases

### What It Does NOT Do

- **Does not sync secrets to third parties** — Storage is your own SQLite file. No cloud account required
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
| Hand-rolled secret sharing for side projects         | Single binary + single SQLite file; runs on localhost or a small VPS              |

### The Philosophy

1. **Minimal setup, maximum value.** `go run ./cmd/server`, copy the printed key, `login` — managing variables in under a minute. No config file required on the server.
2. **Your process stays yours.** Works as local binaries, Docker containers, or release assets — no SaaS dependency.
3. **Boring technology.** SQLite, single admin key, plain `.env` format. Easy to back up (copy the `.db` file), easy to reason about.

## Use Cases

| Scenario                  | How Envious Helps                                                                                  |
| ------------------------- | -------------------------------------------------------------------------------------------------- |
| **Side-project secrets**  | One server tracks every project's dev/prod variables instead of a folder of `.env.backup` files    |
| **Team onboarding**       | Teammates `login` once, then `var export` the environment they need — no secret sprawl in chat     |
| **Multi-env deployments** | `dev`/`staging`/`prod` environments per app keep promotion explicit (`export` from one, `import`) |
| **Homelab services**      | Single SQLite file backs up with everything else; dashboard edits beat SSH + `vim .env`            |
| **CI/CD config**          | Pipelines fetch the environment via the CLI or API and write a `.env` at deploy time               |

## Benefits for Developers

- **Single binary per piece** — `envious-server` and `envious` CLI, both pure Go with minimal dependencies
- **Single-file storage** — SQLite means backups are `cp envious.db`; no separate database to operate
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
| **Infra to operate**    | 1 binary + 1 SQLite file         | None                   | Cluster know-how| None (SaaS)                     |
| **Multi-app / multi-env** | Yes, first-class               | Manual folders         | Yes             | Yes                             |
| **Versioning**          | Yes (per-variable counter)       | No (git-hack it)       | Yes             | Yes                             |
| **Encryption at rest**  | Yes (optional key)               | No                     | Yes             | Yes                             |
| **Admin dashboard**     | Yes (built in)                   | No                     | Yes (complex)   | Yes                             |
| **Offline capable**     | Yes                              | Yes                    | Yes             | No                              |
| **Per-user ACLs**       | No (single admin key)            | N/A                    | Yes             | Yes                             |
| **Secret rotation**     | Manual (`set`)                   | Manual                 | Dynamic engines | Varies                          |
| **License**             | MIT                              | N/A                    | BUSL            | Proprietary                     |

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

## CLI Reference

Global flags: `--config` (reserved), `-v/--verbose` (reserved). Config lives at `~/.envious/config`.

### `login` — store credentials

```bash
./envious login --api-key=<KEY> --api-base=http://127.0.0.1:8080
```

| Flag        | Default | Description                        |
| ----------- | ------- | ---------------------------------- |
| `--api-key` | `""`    | Admin API key printed on first run |
| `--api-base`| `""`    | Server base URL (keeps old value if omitted) |

### `app` (aliases: `application`, `apps`)

```bash
./envious app list
./envious app create myapp
./envious app delete 2
```

| Command  | Args     | Description          |
| -------- | -------- | -------------------- |
| `list`   | —        | List all applications |
| `create` | `<name>` | Create an application |
| `delete` | `<id>`   | Delete by id (strict `> 0`) |

### `env` (aliases: `environment`, `envs`, `environments`)

```bash
./envious env list --app-id 2
./envious env create dev --app-id 2
./envious env delete 10
```

| Flag      | Default | Description                              |
| --------- | ------- | ---------------------------------------- |
| `--app-id`| `0`     | `list`: `0` = all apps. `create`: `0` = default app |

### `var` (aliases: `variable`, `vars`, `variables`) — values hidden by default

```bash
./envious var list 10
./envious var list 10 --show-values
./envious var list --app-id 2 --env-name development
./envious var set 10 DATABASE_URL "postgres://..."
./envious var set --env-id 10 DATABASE_URL "postgres://..."
./envious var delete 55
./envious var export 10 > .env
./envious var import 10 .env
```

| Flag            | Default | Description                                              |
| --------------- | ------- | -------------------------------------------------------- |
| `--env-id`      | `0`     | Environment id (alternative to positional `[env_id]`)    |
| `--env-name`    | `""`    | Resolve env by name (with `--app-id`/`--app-name`)       |
| `--app-id`      | `0`     | Application id for name resolution (`0` = all)           |
| `--app-name`    | `""`    | Application name for name resolution                     |
| `--show-values` | `false` | Show secret values in `list` (hidden by default)         |

Resolution order: `--env-id` → positional `[env_id]` → `--env-name` (+ `--app-id`/`--app-name`). Ambiguous names error out instead of guessing.

### Examples

**Create a staging environment and seed it from dev:**

```bash
./envious env create staging --app-id 2
./envious var export --app-id 2 --env-name development > staging.env
./envious var import --app-id 2 --env-name staging staging.env
```

**Rotate a secret:**

```bash
./envious var set 10 API_KEY "new-secret"
./envious var list 10 --show-values
```

## Configuration

The server needs no config file. All options are environment variables.

| Variable        | Default        | Description                                                        |
| --------------- | -------------- | ------------------------------------------------------------------ |
| `PORT`          | `8080`         | TCP port the server binds                                          |
| `DATABASE_PATH` | `./envious.db` | SQLite file (back this up)                                         |
| `ENCRYPTION_KEY`| `""`           | Optional key enabling value encryption at rest; also signs sessions |
| `LOG_LEVEL`     | `info`         | Log level                                                          |

| Source                        | Precedence |
| ----------------------------- | ---------- |
| CLI flags (`--env-id`, …)     | Highest    |
| CLI config (`~/.envious/config`) | Middle  |
| Server env / built-in defaults| Lowest     |

> First run prints the admin key once to stdout **and** writes it to `envious_api_key.txt` next to the database (mode `0600`). The server stores only its bcrypt hash. `ENCRYPTION_KEY` also signs dashboard sessions — always set it in production; the built-in fallback secret is dev-only.

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

## CI/CD Integration

### GitHub Actions (publish `.env` at deploy time)

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Download CLI
        run: |
          gh release download v1.0.0 \
            --repo marcuwynu23/envious \
            --pattern 'envious-linux-amd64' \
            --output ./envious
          chmod +x ./envious
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      - name: Export environment
        run: ./envious var export --env-id 10 > .env
        env:
          # login once in a setup step, or point at a pre-baked config
          HOME: ${{ github.workspace }}
```

### Server via GHCR

```yaml
jobs:
  deploy-server:
    runs-on: ubuntu-latest
    steps:
      - run: docker pull ghcr.io/marcuwynu23/envious-web:latest
```

## Development

### Prerequisites

| Tool | Version   | Purpose              |
| ---- | --------- | -------------------- |
| Go   | 1.21+ (cli), 1.23+ (web) | Compiler |
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
make test             # go test ./... (first run compiles SQLite, allow minutes)
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
├── web/                    # envious-web (Go 1.23+, Echo + SQLite) — its own Go module
│   ├── cmd/server/main.go
│   └── internal/
│       ├── api/            # Echo server, REST + admin handlers, templates/, static/
│       ├── storage/        # SQLite repository (apps/envs/vars/api_key)
│       ├── auth/           # API key init/verify (bcrypt)
│       ├── middleware/     # Logging, Recovery, APIKeyAuth
│       ├── config/         # PORT, DATABASE_PATH, ENCRYPTION_KEY, LOG_LEVEL
│       └── env/            # env models
├── .github/workflows/      # release-cli.yml, release-web-docker.yml
├── AGENTS.md               # contributor/agent operating manual
├── CHANGELOG.md | CONTRIBUTING.md | CODE_OF_CONDUCT.md | LICENSE
```

## Architecture

- **Entrypoint** (`web/cmd/server/main.go`) — Loads env config, opens SQLite, ensures the admin key, serves Echo; graceful shutdown on `SIGINT`/`SIGTERM`
- **Layered API** (`web/internal/api/server.go` → `storage/`) — Handlers do HTTP only (validation at the boundary, `ErrNotFound→404` / `ErrDuplicateKey→409`); all SQL lives in the repository
- **Repository** (`web/internal/storage/sqlite.go`) — Apps/envs/vars/API-key queries with sentinel errors; AES value encryption when `ENCRYPTION_KEY` is set
- **Auth** (`web/internal/auth/`) — 32-byte random key from `crypto/rand`, bcrypt hash storage, constant-time compare; middleware (`APIKeyAuth`) guards `/api`, HMAC-signed cookie sessions guard the dashboard
- **CLI** (`cli/cmd/` + `internal/`) — Cobra commands wire thinly to `client/`; `service/` holds pure logic, `view/` renders tables; shared `loadClient()`/`parseID()` helpers keep errors strict and safe
- **Releases** (`.github/workflows/`) — Tags (`v*`) run vet+tests, then publish the GHCR image and the five CLI binaries with checksums

---

## License

[MIT](LICENSE) — Copyright (c) 2026 Mark Wayne Menorca

Happy Coding! 🚀
