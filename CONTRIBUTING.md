# Contributing to Envious

Thank you for taking the time to contribute. This document explains how we work on **Envious** (the `web` server and `cli` client).

## Code of conduct

Participation is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By contributing, you agree to uphold it.

## Project layout

| Directory | Role |
|-----------|------|
| `web/` | API and server-rendered admin UI (Go 1.23+, SQLite) |
| `cli/` | Command-line client that talks to the web API (Go 1.21+) |

Each subdirectory is its own Go module with its own `go.mod`, `README`, and tests.

## Before you open a pull request

1. **Describe the change** in the PR (what problem it solves, any trade-offs).
2. **Keep scope focused** — unrelated refactors make review harder.
3. **Run tests** for every module you touch:
   - From `cli/`: `go test ./...` or `make test`
   - From `web/`: `go test ./...` or `make test`
4. **Match existing style** — naming, formatting, and patterns used in nearby code.

## Development quick start

See the root [README.md](README.md) for prerequisites, how to run the server, and how to build and log in with the CLI.

## Reporting issues

Use the [issue templates](.github/ISSUE_TEMPLATE) when they fit your report. Include:

- What you expected vs. what happened
- Steps to reproduce (for bugs)
- Versions: Go, OS, and whether the problem is in `web`, `cli`, or both

## Questions

If something is unclear in the docs or the code, opening a short issue (blank form is fine) is welcome.
