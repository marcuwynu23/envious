# Envious Web Server (envious-web)

This folder contains the Envious web server: **API + server-rendered admin dashboard**, backed by **SQLite**.

## Quick start

From:

`d:\Projects\marcuwynu23\envious\web`

```bash
go mod tidy
go run ./cmd/server
```

Server default address: `http://127.0.0.1:8080`

## Configuration

Environment variables:

- `PORT` (default: `8080`)
- `DATABASE_PATH` (default: `./envious.db`)
- `ENCRYPTION_KEY` (optional; enables encryption at rest for stored values)

Example (PowerShell):

```powershell
$env:PORT="8080"
$env:DATABASE_PATH=".\envious.db"
go run ./cmd/server
```

## Authentication

- Single admin API key
- Generated on first run and printed once to stdout
- Stored as a bcrypt hash in SQLite

All API requests must include:

- Header: `X-API-Key: <YOUR_KEY>`

## Admin dashboard (server-rendered)

- Open: `http://localhost:8080/`
- Login with the API key
- Flow: **Applications → Environments → Variables**

Dashboard routes are cookie-session based and do not require `X-API-Key` headers.

## API routes

All API endpoints are under `/api` and require `X-API-Key`.

### Applications

- `GET    /api/apps`
- `POST   /api/apps` (body: `{ "name": "myapp" }`)
- `GET    /api/apps/:id`
- `DELETE /api/apps/:id`

### Environments

- `GET    /api/envs?app_id=<id>` (optional filter)
- `POST   /api/envs` (body: `{ "app_id": 2, "name": "dev" }` — `app_id` optional → default app)
- `GET    /api/envs/:id`
- `DELETE /api/envs/:id`

### Variables

- `GET    /api/envs/:id/vars`
- `POST   /api/envs/:id/vars` (body: `{ "key": "FOO", "value": "bar" }`)
- `PUT    /api/vars/:id` (body: `{ "value": "new" }`)
- `DELETE /api/vars/:id`

## Tests

```bash
go test ./...
```

## Docker (optional)

This folder includes `Dockerfile` and `docker-compose.yml`. If you use them:

```bash
docker compose up --build
```

- **Development (simple run)**

  ```bash
  make dev
  ```

  This runs:

  ```bash
  go run app/main.go
  ```

- **Development watch (requires a file watcher such as Air)**

  ```bash
  make dev-watch
  ```

  After installing `air`, you can also run:

  ```bash
  air -c .air.toml
  ```

- **Production-style start**

  ```bash
  make start
  ```

  This target is functionally similar to `make dev` but is kept separate so you can introduce production-specific flags or behavior later.

---

## Building Binaries

The `makefile` is configured to support multi-OS/arch builds using Go cross-compilation.

- **Build for current platform**

  ```bash
  make build
  ```

  Output: `build/webapp-<GOOS>-<GOARCH>[.exe]`

- **Cross-compile**

  ```bash
  make build GOOS=linux GOARCH=amd64
  make build GOOS=windows GOARCH=arm64
  ```

You can adjust the output directory and app name via the variables in `makefile`.

---

## API Overview

Base URL (default): `http://0.0.0.0:8080`

- **View routes**
  - `GET /page/home` – Render `views/home.html`

- **User API routes (JSON)**
  - `GET  /api/v1/users/all` – List all users
  - `POST /api/v1/users` – Create a user
    - Example body:

      ```json
      {
        "name": "John Doe",
        "email": "john@example.com",
        "age": 30
      }
      ```

  - `GET    /api/v1/users/:id` – Get user by MongoDB ObjectID
  - `PUT    /api/v1/users/:id` – Replace user fields
  - `PATCH  /api/v1/users/:id` – Partially update user
  - `DELETE /api/v1/users/:id` – Delete user

---

## Testing

The tests live under `tests/app/**` and are split by concern.

- **Run all tests**

  ```bash
  make test
  ```

  This runs:

  ```bash
  go test ./tests/...
  ```

Some tests that touch MongoDB are defensive: they either skip or handle failure gracefully if MongoDB is not available.

---

## Environment & Configuration

The app uses `github.com/joho/godotenv` to load environment variables from `.env` at startup:

- `APP_LISTEN_ADDR` – address Echo will bind to (default: `0.0.0.0:8080`)
- `MONGO_URI` – MongoDB connection string (default: `mongodb://localhost:27017`)
- `MONGO_DB_NAME` – MongoDB database name (default: `ginApp`)

You can also configure these via real environment variables in production instead of `.env`.

---

## Service Configuration (Linux & Windows)

### Linux: systemd service

On a Linux host using `systemd`, you can run the compiled binary as a service.

1. **Build the binary** (example for Linux amd64):

   ```bash
   make build GOOS=linux GOARCH=amd64
   ```

2. **Copy artifacts to a target directory** (for example `/opt/webapp`):

   ```bash
   sudo mkdir -p /opt/webapp
   sudo cp build/webapp-linux-amd64 /opt/webapp/webapp
   sudo cp .env /opt/webapp/.env
   sudo cp -r views /opt/webapp/views
   ```

3. **Create a `systemd` unit file**, for example `/etc/systemd/system/webapp.service`:

   ```ini
   [Unit]
   Description=Golang Web MVC Project Template
   After=network.target

   [Service]
   Type=simple
   WorkingDirectory=/opt/webapp
   ExecStart=/opt/webapp/webapp
   Restart=on-failure
   EnvironmentFile=/opt/webapp/.env

   [Install]
   WantedBy=multi-user.target
   ```

4. **Reload and start the service**:

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable webapp
   sudo systemctl start webapp
   sudo systemctl status webapp
   ```

### Windows: service via `sc.exe`

On Windows you can install the compiled binary as a service using `sc.exe`.

1. **Build a Windows binary** (from a dev machine or CI):

   ```bash
   make build GOOS=windows GOARCH=amd64
   ```

   This produces something like:

   ```text
   build\webapp-windows-amd64.exe
   ```

2. **Copy artifacts to a directory**, for example:

   ```text
   C:\webapp\
     webapp.exe
     .env
     views\...
   ```

3. **Create the service** (run in an elevated PowerShell or Command Prompt):

   ```powershell
   sc.exe create WebApp binPath= "C:\webapp\webapp.exe" start= auto
   ```

4. **Start and manage the service**:

   ```powershell
   sc.exe start WebApp
   sc.exe stop WebApp
   sc.exe delete WebApp   # to remove the service
   ```

Make sure the service account has read access to the application directory and `.env` file. In more advanced setups you may want to externalize configuration to real environment variables rather than relying solely on `.env`.

---

## License

This template is provided under the **MIT License**. See the `LICENSE` file for the full license text. Adapt it freely to your own projects.
