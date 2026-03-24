﻿﻿﻿<div align="center">
  <h1>Envious</h1>
  <p>Multi-application environment variable manager (Go + SQLite)</p>
  <p>
    <img alt="Go" src="https://img.shields.io/badge/Go-1.21%2B%20%7C%201.23%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white" />
    <img alt="SQLite" src="https://img.shields.io/badge/SQLite-003B57?style=for-the-badge&logo=sqlite&logoColor=white" />
    <img alt="Echo" src="https://img.shields.io/badge/Echo-4B32C3?style=for-the-badge" />
  </p>
</div>

- **Web server** (`envious-web`): API + server-rendered admin dashboard, backed by SQLite
- **CLI** (`envious-cli`): manages applications, environments, and variables via the web API

## Prerequisites

- Go (web uses Go 1.23+, cli uses Go 1.21+)

## Run the server (web)

```bash
cd web
```

```bash
go mod tidy
go run ./cmd/server
```

### Server configuration

Environment variables:

- `PORT` (default: `8080`)
- `DATABASE_PATH` (default: `./envious.db`)
- `ENCRYPTION_KEY` (optional; enables encryption at rest for stored values)

### First-run API key

On first run the server generates a single admin API key and prints it once to stdout:

```
Envious initial API key (store it securely): <KEY>
```

Keep this key safe. The server stores only a bcrypt hash.

### Admin dashboard

- Open: http://localhost:8080/
- Login using the API key above
- Flow: Applications → Environments → Variables

## Build and use the CLI

From:

```bash
cd cli
```

```bash
go mod tidy
go build -o envious .
```

### Configure CLI (login)

```bash
./envious login --api-key=<KEY> --api-base=http://127.0.0.1:8080
```

The CLI stores configuration in:

- `~/.envious/config`

### Application workflow

Create an application:

```bash
./envious app create myapp
./envious app list
```

Create environments under an app:

```bash
./envious env create dev --app-id=2
./envious env list --app-id=2
```

Set and list variables:

```bash
./envious var set 10 DATABASE_URL "postgres://..."
./envious var list 10
./envious var export 10 > .env
./envious var import 10 .env
```

Notes:

- `--app-id=0` means “default application” for create, and “all applications” for list.
- `var` commands operate on an environment id.
