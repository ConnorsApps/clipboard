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
go test -race ./...           # Run tests
```

### CLI (`cb`)
```bash
go build -o cb ./cmd/cb       # Build CLI binary
go run ./cmd/cb               # Run CLI without building

cb login                      # Authenticate (prompts for server URL + password)
cb get                        # Print clipboard content (no trailing newline when piped)
cb set [text]                 # Set clipboard content (reads stdin if no arg; --trim flag, default on)
cb clear                      # Clear clipboard content
cb live                       # Live TUI viewer/editor (bubbletea; e/i=edit, c=copy, ctrl+d=clear, q=quit)
cb files list [--json]        # List uploaded files
cb files upload <path>        # Upload a file (with progress; tab-completes paths)
cb files download [id] [-o]   # Download a file (interactive picker if no id; pipes to stdout if not a tty)
cb files clear                # Delete all uploaded files
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

**Stack**: React 19 + TypeScript + Vite + Tailwind v4 + tus-js-client (frontend), Go 1.25 + Gorilla WebSocket + tusd v2 (backend), optional MongoDB.

### Backend Packages (`internal/`)

- **`server/`** — HTTP server setup, routing, middleware (CORS, token auth, user context injection), SPA fallback to `index.html`
- **`auth/`** — Password login, token generation (64-char hex, crypto random), user ID derivation (SHA-256 of password, first 16 chars)
- **`tokenstore/`** — Interface with two implementations: in-memory (volatile) and MongoDB (persistent, TTL configurable via `TOKEN_EXPIRY`)
- **`clipboard/`** — Per-user in-memory WebSocket connection map; broadcasts `content` and `files_list` messages to all connected clients. Each connection has its own write lock (gorilla allows only one concurrent writer)
- **`files/`** — Per-user file storage, tusd integration for resumable uploads, broadcasts file list changes over WebSocket. The list hides tusd `.info`/`.lock` files and uploads still in progress, reports each file's MIME `type` (tus `filetype` metadata → extension → content sniffing), and is sorted newest first
- **`config/`** — Loads all config from environment variables

### Frontend (`frontend/src/`)

A single screen (no router): shared text on top, a thumbnail grid of images/files below (side by side on wide screens), and an add bar (Paste / Photos / Camera / Files) that is a fixed bottom dock on phones.

- **`App.tsx`** — Switches between `Login` and `Home` based on the token in `localStorage`
- **`useSync.ts`** — The one WebSocket: shared text, file list, connection state; reconnects with backoff and immediately when the tab becomes visible or goes online; text edited while offline is pushed on reconnect
- **`useUploads.ts`** — Upload queue (3 parallel) over tus with local previews; a finished upload hands its preview over once the server's `files_list` includes it
- **`Home.tsx`** — Layout plus global paste (images upload; text outside the text box replaces the shared text; `text/plain` wins when both are present) and drag-and-drop
- **`FileViewer.tsx`** — Full-screen preview with Copy image / Share / Download / Delete; pushes a history entry so Back closes it
- **`clipboard.ts`** — Copy text (iOS-safe `execCommand` path), copy image (as PNG via `ClipboardItem`), read the system clipboard for the Paste button

### CLI Package (`cmd/cb/`, `pkg/cbclient/`)

- **`cmd/cb/`** — CLI entry point built with `urfave/cli/v3` and `lipgloss` for styled output; commands: `login`, `get`, `set`, `clear`, `live`, `files`, `logout`; shell completion enabled via `EnableShellCompletion: true`
- **`pkg/cbclient/`** — Reusable Go client library wrapping the REST + tus upload API; `DownloadFileAt` supports byte-range resumable downloads

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
- `GET /api/files` — List uploaded files (`id`, `name`, `type`, `size`, `uploadedAt`; newest first)
- `GET /api/files/<id>` — Download a file (`Content-Type` from stored metadata)
- `DELETE /api/files/<id>` — Delete a file
- `POST/PATCH/HEAD /api/uploads` — tusd resumable upload endpoints (HEAD is authenticated so interrupted uploads can resume)
- `GET /ws` — WebSocket for real-time clipboard + file-list sync; sends `content` then `files_list` right after connecting

### Key Integration Points

- The Helm chart uses a `Recreate` (not `RollingUpdate`) deployment strategy to avoid token loss when using the in-memory token store.
- Vite dev server proxies `/api` and `/ws` to `localhost:8080` — run the Go backend separately when doing frontend dev.
- `<img>`/`<video>` sources and download links pass the token as a `?token=` query parameter, since they can't send an `Authorization` header.
- The Docker build is 3-stage: Node 24 builds the frontend, Go builds the binary, Alpine is the final runtime image.
- The `cb live` command is a bubbletea TUI that connects to `/ws`, filters for `content`-type messages, and supports in-place editing, local clipboard copy (via `github.com/atotto/clipboard`), and auto-reconnect with exponential backoff.
- `cb files download` retries with exponential backoff (up to 3 attempts) using HTTP Range requests to resume interrupted downloads.
