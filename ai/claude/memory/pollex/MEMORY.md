# Pollex - Project Memory

## Project
Text polishing tool: Go API + Browser Extension + llama.cpp GPU on Jetson Nano 4GB.

## Key Paths
- Entry point: `cmd/pollex/main.go`
- Adapter interface + impls: `internal/adapter/`
- Config: `internal/config/`
- Handlers: `internal/handler/`
- Middleware: `internal/middleware/`
- Server wiring: `internal/server/`
- System prompt: `prompts/polish.txt`
- Tasks: `tasks/todo.md`, `tasks/lessons.md`
- Vault docs: `~/Projects/knowledge/10_projects/pollex/`

## Architecture
- Module: `github.com/mlorentedev/pollex` (Go 1.26, stdlib net/http + yaml.v3 + prometheus/client_golang)
- `LLMAdapter` interface: `Name()`, `Polish(ctx, text, systemPrompt)`, `Available()`
- Adapters: MockAdapter, OllamaAdapter, ClaudeAdapter, LlamaCppAdapter
- Handlers: handler.Health(), handler.Models(), handler.Polish()
- Metrics: `internal/metrics/` package (promauto, default registry)
- Middleware: middleware.Chain() = CORS → RequestID → Logging → Metrics → APIKey → RateLimit → MaxBytes → Timeout → mux
- Logging: `log/slog` with JSON output (structured: request_id, method, path, status, duration_ms)
- Config: YAML file + env var overrides (POLLEX_*)
- `server.SetupMux(adapters, models, systemPrompt, apiKey, version)` — extracted for httptest testability
- Version: `var version = "dev"` in main.go, injected via `-ldflags -X main.version` at build time
- `cmd/pollex/main.go:buildAdapters()` — thin composition root, only place knowing concrete types
- Graceful shutdown: http.Server + signal.Notify(SIGINT, SIGTERM) + 10s drain

## Hardening
- API key: X-API-Key header, crypto/subtle.ConstantTimeCompare, /api/health + /metrics exempt
- Request ID: crypto/rand 32 hex, X-Request-ID header, context propagation
- Rate limit: 10 req/min/IP sliding window (in-memory)
- Body limit: 64KB maxBytes middleware
- Text limit: maxTextLength = 10000 chars (handler), extension enforces 1500 chars (timeout budget)
- Rich health: per-adapter availability with unavailableReason()

## Secrets
- Managed by dotfiles project (`~/Projects/dotfiles/sensitive/pollex.api-key.secret.age`)
- age-encrypted, env-mapping.conf maps POLLEX_API_KEY=pollex.api-key
- `make deploy-secrets` reads $POLLEX_API_KEY from shell → writes /etc/pollex/secrets.env on Jetson
- Rotation: `secrets_rotate POLLEX_API_KEY` → `make deploy-secrets` → update extension Settings

## Dev Workflow
- `make dev` = `go run ./cmd/pollex --mock --port 8090`
- `make test` = `go test -v -race ./...`
- All `go run` / `go test` commands need `source ~/.zshrc` in Bash tool for Go 1.26

## Extension Architecture
- Service worker (background.js): persistent fetch via importScripts("api.js"), message protocol (POLISH_START/CANCEL/TICK)
- Storage keys: apiUrl, apiKey, draftText, polishJob, history (7 entries max)
- popup.js: UI layer, storage.onChanged listener, recoverJobState on open
- Input validation in background.js: type, empty, max 1500 chars, error truncation 200 chars
- Color scheme: cyan-700 (#0e7490)

## Status
- Phases 1-16: COMPLETE
- 80+ tests (with subtests), -race clean, go vet clean

## Docker
- Image: alpine:3.21, multi-stage, 24.7MB, user pollex:pollex (UID 1001)
- `make docker-dev` = mock mode, `make docker-build` = with git version/SHA labels
- `docker-compose.monitoring.yml` = Prometheus + Alertmanager + Grafana (needs `make dev` or `make docker-dev`)
- Alerting: 6 rules in `deploy/prometheus/alerts.yml` (PollexDown, LlamaCppDown, latency, errors, burn rate)
- Dashboard: `deploy/grafana/pollex-dashboard.json` (11 panels, SLO status + traffic + latency + infra)

## Notes
- `source ~/.zshrc` needed in Bash tool for Go 1.26
- Rate limiter: reads Cf-Connecting-Ip header; authenticated requests bypass rate limit
- Tunnel ID: fb8f4bb2-14a4-48ce-9b94-b206042753a6 (pollex.mlorente.dev)
- Binary sizes: local ~10MB, ARM64 ~9.5MB
- Extension needs manual load: chrome://extensions → Load unpacked → extension/
- Config default PromptPath: `prompts/polish.txt` (relative to repo root)
