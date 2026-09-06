# Envious User Guide

> The full operator manual. The [README](README.md) is the overview;
> this guide is the day-to-day reference for normal and enterprise use.
> The same content in web form lives on the [official site](docs/index.html).

- [Part 1 — Normal way](#part-1--normal-way-single-node-sqlite)
- [Part 2 — Enterprise way](#part-2--enterprise-way)

---

# Part 1 — Normal way (single node, SQLite)

SQLite is the default backend: zero setup, one file, correct under
concurrency. Skip every `DB_*` variable unless Part 2 applies to you.

## 1. Install

No Go toolchain needed — download a ready binary or run Docker:

```bash
# Linux (x86_64)
curl -L -o envious \
  https://github.com/marcuwynu23/envious/releases/latest/download/envious-linux-amd64
chmod +x envious
./envious version
```

```powershell
# Windows (PowerShell)
Invoke-WebRequest `
  -Uri https://github.com/marcuwynu23/envious/releases/latest/download/envious-windows-amd64.exe `
  -OutFile envious.exe
.\envious.exe version
```

```bash
# macOS (Apple Silicon; use envious-darwin-amd64 on Intel)
curl -L -o envious \
  https://github.com/marcuwynu23/envious/releases/latest/download/envious-darwin-arm64
chmod +x envious
```

```bash
# Server via Docker
docker run -p 8080:8080 \
  -e ENCRYPTION_KEY="pick-a-long-random-value" \
  -v envious-data:/data \
  -e DATABASE_PATH=/data/envious.db \
  ghcr.io/marcuwynu23/envious-web:latest
```

## 2. First run and login

Start the server. On first run it prints the admin API key **once** and
writes it to `envious_api_key.txt` next to the database (mode `0600`).
Only a bcrypt hash is stored.

```bash
cd web
go run ./cmd/server
# Envious initial API key (store it securely): <KEY>

cd ../cli
./envious login --api-key=<KEY> --api-base=http://127.0.0.1:8080
# login saved
```

Credentials live in `~/.envious/config`. If you lose the key and the
`.txt` file, see [rotating the admin key](#rotating-the-admin-api-key).

## 3. Core workflow

```bash
./envious app create billing
./envious app list
./envious env create prod --app-id=2
./envious env list --app-id=2
./envious var set 10 DATABASE_URL "postgres://..."
./envious var list 10              # values hidden by default
./envious var list 10 --show-values
./envious var export 10 > .env
./envious var import 10 .env
```

Prefer a UI? Open `http://localhost:8080/`, log in with the API key, and
follow Applications → Environments → Variables. The build version is
always visible in the header badge and on the `/about` page
(`-version` flag and `GET /api/version` show it too).

## 4. `.env` round trip and name resolution

`export` prints what `import` (and your app) consumes: blank lines and
`#` comments are skipped, keys split on the first `=`, keys trimmed.

Environments resolve as `--env-id` → positional `[env_id]` →
`--env-name` with `--app-id`/`--app-name`. Ambiguous names error out
instead of guessing:

```bash
./envious var export --app-id 2 --env-name development > staging.env
./envious var import --app-id 2 --env-name staging staging.env
```

## 5. CLI reference

Global flags `--config` / `-v/--verbose` are reserved. Config lives at
`~/.envious/config`.

### `login` — store credentials

```bash
./envious login --api-key=<KEY> --api-base=http://127.0.0.1:8080
```

| Flag         | Default | Description                                  |
| ------------ | ------- | -------------------------------------------- |
| `--api-key`  | `""`    | Admin API key printed on first run           |
| `--api-base` | `""`    | Server base URL (keeps old value if omitted) |

### `app` (aliases: `application`, `apps`)

```bash
./envious app list
./envious app create myapp
./envious app delete 2
```

| Command  | Args     | Description             |
| -------- | -------- | ----------------------- |
| `list`   | —        | List all applications   |
| `create` | `<name>` | Create an application   |
| `delete` | `<id>`   | Delete by id (strict `> 0`) |

### `env` (aliases: `environment`, `envs`, `environments`)

```bash
./envious env list --app-id 2
./envious env create dev --app-id 2
./envious env delete 10
```

| Flag       | Default | Description                                        |
| ---------- | ------- | -------------------------------------------------- |
| `--app-id` | `0`     | `list`: `0` = all apps. `create`: `0` = default app |

### `var` (aliases: `variable`, `vars`, `variables`)

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

| Flag            | Default | Description                                           |
| --------------- | ------- | ----------------------------------------------------- |
| `--env-id`      | `0`     | Environment id (alternative to positional `[env_id]`) |
| `--env-name`    | `""`    | Resolve env by name (with `--app-id`/`--app-name`)    |
| `--app-id`      | `0`     | Application id for name resolution (`0` = all)        |
| `--app-name`    | `""`    | Application name for name resolution                  |
| `--show-values` | `false` | Show secret values in `list` (hidden by default)      |

## 6. Configuration reference

The server needs no config file. All options are environment variables.

| Variable        | Default        | Description                                                        |
| --------------- | -------------- | ------------------------------------------------------------------ |
| `PORT`          | `8080`         | TCP port the server binds                                          |
| `DATABASE_PATH` | `./envious.db` | SQLite file (back this up)                                         |
| `DB_DRIVER`     | `sqlite`       | `sqlite` (single file) or `postgres` (server, multi-instance)      |
| `DATABASE_URL`  | `""`           | Postgres URL, e.g. `postgres://user:pass@host:5432/envious?sslmode=require` |
| `DB_MAX_OPEN_CONNS` | `10` (sqlite) / `25` (pg) | SQL pool size                                           |
| `DB_MAX_IDLE_CONNS` | `10` (sqlite) / `5` (pg)  | SQL idle pool                                           |
| `RATE_LIMIT_RPS`/`RATE_LIMIT_BURST` | `20` / `40` | Per-IP API throttle (`0` disables)                       |
| `AUTH_CACHE_TTL_SECONDS` | `60`      | In-memory API-key hash cache (`0` = verify in DB every time)       |
| `ENCRYPTION_KEY`| `""`           | Optional key enabling value encryption at rest; also signs sessions |
| `LOG_LEVEL`     | `info`         | `debug` \| `info` \| `warn` \| `error`                              |
| `LOG_FORMAT`    | `json`         | `json` (collectors) or `text` (local dev)                          |

| Source                           | Precedence |
| -------------------------------- | ---------- |
| CLI flags (`--env-id`, …)        | Highest    |
| CLI config (`~/.envious/config`) | Middle     |
| Server env / built-in defaults   | Lowest     |

> `ENCRYPTION_KEY` also signs dashboard sessions — always set it in
> production; the built-in fallback secret is dev-only.

## 7. REST API summary

All endpoints live under `/api` and require `X-API-Key`, except
`GET /api/version` (public build version).

| Endpoint | Description |
| -------- | ----------- |
| `GET /api/apps` · `POST /api/apps` | List / create applications |
| `GET /api/apps/:id` · `DELETE /api/apps/:id` | Fetch / delete an application |
| `GET /api/envs?app_id=` · `POST /api/envs` | List (optionally filtered) / create environments |
| `GET /api/envs/:id` · `DELETE /api/envs/:id` | Fetch / delete an environment |
| `GET /api/envs/:id/vars` · `POST /api/envs/:id/vars` | List / set variables |
| `PUT /api/vars/:id` · `DELETE /api/vars/:id` | Update / delete a variable |
| `GET /api/activity?action=&limit=` | Query the audit trail |
| `GET /healthz` · `GET /readyz` | Liveness / readiness (DB ping) probes |

Errors are `{"error": "..."}`: `404` missing, `409` duplicate, `400`
bad input, `401` bad key, `429` past the rate limit.

## 8. Backup and upgrade (SQLite)

```bash
cp envious.db envious.db.bak   # the whole state is this one file
# upgrade: stop server, replace binary, start again (migrations run on boot)
```

## 9. Troubleshooting

| Symptom | Check |
|---|---|
| `401` from CLI | `login --api-base` correct? Key rotated? See audit for `auth.login_failed` |
| `version` shows `dev` | Expected for unstamped builds; use `make release-build` or `VERSION=v1.0.0` |
| `429 rate limit exceeded` | Raise `RATE_LIMIT_RPS`/`BURST`, or back off the client |
| Dashboard shows login loop | `ENCRYPTION_KEY` changed between restarts (sessions invalidated) — log in again |
| Port in use | Change `PORT` |

---

# Part 2 — Enterprise way

## 10. Choosing a backend

|  | SQLite (default) | Postgres (`DB_DRIVER=postgres`) |
|---|---|---|
| Setup | Zero — one file | Needs a server (compose profile or managed) |
| Instances | 1 (`Recreate` on K8s) | Many (`RollingUpdate`) |
| Throughput | Tens of RPS, bursty writes retried | Hundreds of RPS, concurrent writers |
| Backup | Copy the file | `pg_dump` / managed PITR |
| When | Single node, team tool | Multi-instance, audited production |

The API, CLI, dashboard, and tests are identical on both backends
(storage tests run against Postgres when `TEST_POSTGRES_URL` is set).

## 11. Postgres via Compose

```bash
DB_DRIVER=postgres DATABASE_URL=postgres://envious:envious@db:5432/envious?sslmode=disable \
  docker compose --profile postgres up --build -d
# from web/: plain sqlite stays the default
docker compose up --build -d
```

`web/docker-compose.yml` ships a healthy `db` service (Postgres 16,
persistent volume) behind the `postgres` profile; the app reads
`DATABASE_URL` and reports its dialect on `/readyz`. For managed
Postgres, point `DATABASE_URL` at it with `sslmode=require`.

## 12. Kubernetes

Manifests live in `web/k8s/` (validated with kubeconform):

```bash
kubectl apply -f web/k8s/namespace.yaml
kubectl apply -f web/k8s/configmap.yaml
kubectl create secret generic envious-secrets --namespace envious \
  --from-literal=ENCRYPTION_KEY="$(openssl rand -hex 32)" --from-literal=DATABASE_URL=""
kubectl apply -f web/k8s/pvc.yaml web/k8s/deployment.yaml web/k8s/service.yaml
# first-run key: kubectl -n envious logs deploy/envious-web | grep 'initial API key'
```

Rules: SQLite → `replicas: 1` + `Recreate` + PVC. Postgres → set
`DATABASE_URL`, drop the volume/PVC, raise replicas, switch to
`RollingUpdate`. Probes are wired (`/healthz` liveness, `/readyz`
readiness). Prefer sealed-secrets / external-secrets over the literal
template in production.

## 13. TLS and networking

The server speaks plain HTTP by design — terminate TLS at your load
balancer or ingress (NGINX, Traefik, cloud LB, `ingress-nginx` with
cert-manager). Never expose it directly to the internet without auth
at the edge either; the single admin key is the only gate.

## 14. Hardening reference

| Control | Knobs | Notes |
|---|---|---|
| Throttle | `RATE_LIMIT_RPS=20`, `RATE_LIMIT_BURST=40` | Per-IP on `/api` + `/login`; `429` past it; probes and `/api/version` exempt |
| Auth cost | `AUTH_CACHE_TTL_SECONDS=60` | bcrypt hash cached in memory; rotation applies on reload; `0` = DB every time |
| Body size | Fixed 1MB | JSON + `.env` uploads |
| Timeouts | Fixed: header 10s, read/write 30s, idle 120s | Slow-client protection |
| Pool | `DB_MAX_OPEN_CONNS/DB_MAX_IDLE_CONNS` | Defaults 10/10 sqlite, 25/5 pg |
| Sessions | `ENCRYPTION_KEY` | Required in prod; rotation logs everyone out |

## 15. Logs, audit, and Fluent Bit

Stdout JSON with request IDs (`X-Request-ID` echoed back). Every
mutation and login is stored in SQLite/Postgres **and** streamed with
`"audit":true`. Details carry metadata only — never values or keys.

```bash
curl -H "X-API-Key: $KEY" 'http://127.0.0.1:8080/api/activity?action=var.set&limit=50'
```

```ini
[INPUT]
    Name              tail
    Path              /var/log/envious/*.log
    Parser            json
    Tag               envious.*
    Refresh_Interval  5

[OUTPUT]
    Name  stdout
    Match envious.*
```

Forward to Loki/Elasticsearch/OpenSearch with their output plugins.
Alert on `"action":"auth.login_failed"` (brute force) and mirror
`"audit":true` into long-term storage.

## 16. Probes and alerting

- Liveness: `GET /healthz` → 200 when the process runs.
- Readiness: `GET /readyz` → 200 only when the DB answers (503 otherwise).
- Suggested alerts: readiness flapping, `429` rate spikes, `auth.login_failed` bursts, `audit_write_failed` warnings.

## 17. Backups and disaster recovery

- SQLite: copy `DATABASE_PATH` on a schedule (stop-the-world not needed for reads; copy the WAL too, or `sqlite3 .backup`).
- Postgres: `pg_dump` nightly + managed PITR where available.
- The audit trail lives in the same database — it is backed up with it.

## 18. Rotating the admin API key

The server stores only a bcrypt hash, so a lost key cannot be recovered:

```bash
# sqlite
sqlite3 envious.db "DELETE FROM api_key;"
# postgres
psql "$DATABASE_URL" -c "DELETE FROM api_key;"
# then restart: a fresh key is generated, printed once, and written to envious_api_key.txt
```

> Rotating `ENCRYPTION_KEY` re-keys sessions (everyone logs out) and —
> critically — makes previously encrypted values unreadable. Rotate it
> only with a decrypt-export / change-key / encrypt-import maintenance
> window.

## 19. Load testing and capacity

```bash
API_BASE=http://127.0.0.1:8080 API_KEY=<key> k6 run web/test/load/k6.js
```

Mixed 50-VU workload asserting `p(95)<500ms` and `<1%` errors. Raise
`RATE_LIMIT_*` above the target rate first or throttling fails the run
by design. Contention correctness (no lost updates/duplicates) is
covered continuously by `TestConcurrentSetVar` on both dialects
(`-race` runs in CI).

## 20. Enterprise security checklist

- [ ] `ENCRYPTION_KEY` set (32+ random bytes), stored in a secret manager
- [ ] Postgres with `sslmode=require` for multi-instance
- [ ] TLS terminated at LB/ingress; no direct internet exposure
- [ ] `LOG_LEVEL=info`, JSON logs shipped to a collector with retention
- [ ] Audit mirrored long-term; `auth.login_failed` alerted
- [ ] Backups scheduled and restore-tested
- [ ] Binaries/images pinned to release tags with checksums verified
