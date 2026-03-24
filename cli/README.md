﻿﻿﻿# Envious CLI (envious-cli)

This folder contains the Envious CLI. It talks to the Envious web server API and manages:

- Applications
- Environments
- Variables

## Prerequisites

- Go 1.21+
- Running Envious server (`d:\Projects\marcuwynu23\envious\web`)

## Build

From:

`d:\Projects\marcuwynu23\envious\cli`

```bash
go mod tidy
go build -o envious .
```

## Configure (login)

You need the admin API key printed by the server on first run.

```bash
./envious login --api-key=<KEY> --api-base=http://127.0.0.1:8080
```

Config is stored at:

- `~/.envious/config`

## Commands

### Applications

```bash
./envious app list
./envious app create myapp
./envious app delete 2
```

### Environments

Create/list environments for a specific application:

```bash
./envious env list --app-id=2
./envious env create dev --app-id=2
./envious env delete 10
```

Notes:

- `--app-id=0` means “all apps” for `env list`
- `--app-id=0` means “default app” for `env create`

### Variables

Variables operate on an environment id:

```bash
./envious var set 10 DATABASE_URL "postgres://..."
./envious var list 10
./envious var delete 55
```

Export/import `.env`:

```bash
./envious var export 10 > .env
./envious var import 10 .env
```

## API base

The CLI talks to the server API under `/api` (example: `http://127.0.0.1:8080/api/...`).

## Tests

```bash
go test ./...
```

## Docker (optional)

This folder includes `Dockerfile` and `docker-compose.yml` to package the CLI.
