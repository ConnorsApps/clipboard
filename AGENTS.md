# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## What This Project Is

A shared clipboard service with real-time WebSocket sync for sharing text and files across devices. Password-based auth, optional MongoDB token persistence, resumable file uploads via the tus protocol. Includes a `cb` CLI for interacting with the service from the terminal.

## Commands

### Backend (Go)
```bash
go build -o server .          # Build server binary
go run .                      # Run without live reload
wgo run .                     # Live reload (install: go install github.com/bokwoon95/wgo@latest)
```

### CLI (`cb`)
```bash
go build -o cb ./cmd/cb       # Build CLI binary
go run ./cmd/cb               # Run CLI without building

cb login                      # Authenticate (prompts for server URL + password)
cb get                        # Print clipboard content
cb set <text>                 # Set clipboard content (reads stdin if no arg)
cb watch                      # Stream clipboard updates in real time via WebSocket
cb files list                 # List uploaded files
cb files upload <path>        # Upload a file
cb files download [id]        # Download a file (interactive picker if no id)
cb files delete <id>          # Delete a file
cb logout                     # Remove saved credentials
```

CLI config is stored at `$XDG_CONFIG_HOME/clipboard-cli/config.yaml` (defaults to `~/.config/clipboard-cli/config.yaml`).

### Frontend
```bash
cd frontend
npm install
npm run dev      # Dev server, proxies /api and /ws to localhost:8080
npm run build    # TypeScript check + Vite build
npm run preview  # Preview production build
```

### Dev (full stack)
```bash
./scripts/dev.sh   # Starts backend (wgo) + frontend (npm run dev) together
```

### Docker
```bash
docker build .                              # Multi-stage build (Node → Go → Alpine)
./scripts/docker-build-and-push.sh          # Build and push image
```

### Helm
```bash
./scripts/package-chart.sh                                          # Package chart
helm install my-clipboard clipboard/clipboard -f my-values.yaml    # Deploy
```

## Architecture

**Stack**: React 19 + TypeScript + Vite (frontend), Go 1.25 + Gorilla WebSocket + tusd v2 (backend), optional MongoDB.

### Backend Packages (`internal/`)

- **`server/`** — HTTP server setup, routing, middleware (CORS, token auth, user context injection), SPA fallback to `index.html`
- **`auth/`** — Password login, token generation (64-char hex, crypto random), user ID derivation (SHA-256 of password, first 16 chars)
- **`tokenstore/`** — Interface with two implementations: in-memory (volatile) and MongoDB (persistent, TTL configurable via `TOKEN_EXPIRY`)
- **`clipboard/`** — Per-user in-memory WebSocket connection map; broadcasts `update`, `clear`, and `files_list` messages to all connected clients
- **`files/`** — Per-user file storage, tusd integration for resumable uploads, broadcasts file list changes over WebSocket
- **`config/`** — Loads all config from environment variables

### CLI Package (`cmd/cb/`, `pkg/cbclient/`)

- **`cmd/cb/`** — CLI entry point built with `urfave/cli/v3` and `lipgloss` for styled output; commands: `login`, `get`, `set`, `watch`, `files`, `logout`
- **`pkg/cbclient/`** — Reusable Go client library wrapping the REST + tus upload API

### Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `CLIPBOARD_PASSWORDS` | `1234` | Comma-separated valid passwords |
| `MONGODB_URI` | *(unset)* | If set, uses MongoDB for token persistence; otherwise in-memory |
| `FILES_DIR` | `./tmp-files` | Directory for uploaded files |
| `PORT` | `8080` | HTTP server port |
| `IS_LOCAL` | `false` | Enable console log output |
| `TOKEN_EXPIRY` | `30d` | Token lifetime — number of days (e.g. `10d`) or `never`; used as MongoDB TTL index |

### API Surface

- `POST /api/login` — Returns a token
- `GET /api/clipboard` — Get current clipboard content
- `POST /api/clipboard` — Set clipboard content
- `GET /api/files` — List uploaded files
- `GET /api/files/<id>` — Download a file
- `DELETE /api/files/<id>` — Delete a file
- `POST/PATCH /api/uploads` — tusd resumable upload endpoints
- `GET /ws` — WebSocket for real-time clipboard + file-list sync

### Key Integration Points

- The Helm chart uses a `Recreate` (not `RollingUpdate`) deployment strategy to avoid token loss when using the in-memory token store.
- Vite dev server proxies `/api` and `/ws` to `localhost:8080` — run the Go backend separately when doing frontend dev.
- The Docker build is 3-stage: Node 24 builds the frontend, Go builds the binary, Alpine is the final runtime image.
- The `cb watch` command connects to `/ws` and filters for `content`-type messages, converting `http://` → `ws://` and `https://` → `wss://` automatically.
