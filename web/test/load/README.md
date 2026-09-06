# Load testing (manual)

k6 is not installed in CI; run it locally against a staging server:

```bash
API_BASE=http://127.0.0.1:8080 API_KEY=<key> k6 run web/test/load/k6.js
```

The script ramps to 50 VUs of mixed app/env/var writes and reads with
`p(95)<500ms` and `<1%` errors. Raise `RATE_LIMIT_RPS`/`RATE_LIMIT_BURST`
server-side above the target rate first, or throttling will fail the run
by design. Concurrency correctness (no lost updates, no duplicates) is
covered automatically by `TestConcurrentSetVar` in `go test` (both
dialects; add `-race` in CI where cgo is available).
