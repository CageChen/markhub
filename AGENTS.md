# AGENTS.md

## Dev Environment

- **Language**: Go 1.24+, frontend vanilla JS/CSS (no bundler)
- **Build**: `make build` → `bin/markhub`; `make run` → build + serve `--path . --open`
- **Dependencies**: `make deps` (runs `go mod tidy && go mod download`)
- **Hot reload dev**: `make dev` (requires [air](https://github.com/air-verse/air))
- **Cross-compile**: `make build-all` (linux/darwin/windows, amd64/arm64)
- **Docker**: `make docker-build` → `docker run -p 8080:8080 -v $(pwd)/docs:/docs markhub`
- **Frontend assets**: embedded via `//go:embed` in `cmd/server/main.go`, no separate build step

## Testing

- **Run all tests**: `make test` or `go test -v ./...`
- **Single test**: `go test -v -run TestName ./internal/package/`
- **Coverage**: `make test-coverage` → generates `coverage.html`
- **CI**: GitHub Actions (`.github/workflows/ci.yml`) — runs `gofmt` check, `go test`, `golangci-lint`

## Code Style

- **Formatter**: `gofmt -s -w .` + `goimports -w .` (via `make fmt`)
- **Linter**: `golangci-lint run` (via `make lint`), config in `.golangci.yml`
- **Line length**: 120 characters max (enforced by `lll` linter)
- **Commit message**: `type: description` format (e.g. `feat:`, `fix:`, `chore:`)

## Architecture

```
cmd/server/
  main.go              # Entry point: config, router, watcher, embedded assets
  web/                 # Frontend (HTML/CSS/JS), embedded into binary
internal/
  config/              # YAML + CLI flag config, multi-folder management, save/load
  fs/                  # FileSystem interface: LocalFS (os) + GitFS (git CLI)
  handler/             # Gin HTTP handlers: file serving, tree API, folder CRUD, WebSocket
  markdown/            # Goldmark parser with GFM, Chroma highlighting, TOC extraction
  watcher/             # fsnotify recursive watcher, triggers WebSocket broadcasts
```

### API Routes

| Method | Endpoint | Handler |
|--------|----------|---------|
| GET | `/api/tree` | `TreeHandler.GetTree` |
| GET | `/api/files/{alias}/{path}` | `FileHandler.GetFile` |
| GET | `/api/raw/{alias}/{path}` | `FileHandler.GetRaw` |
| GET | `/api/ws` | `WSHandler.HandleWS` |
| GET/POST/PUT/DELETE | `/api/folders` | `TreeHandler.*Folder` |
| PUT | `/api/exclude` | `TreeHandler.UpdateGlobalExclude` |
| PUT | `/api/repo-exclude` | `TreeHandler.UpdateRepoExclude` |

## Code Review

- Run `make fmt && make lint && make test` before committing
- All exported types and functions must have doc comments
