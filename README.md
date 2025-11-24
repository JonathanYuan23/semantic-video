# Semantic Video

API daemon for frame extraction plus an Electron-wrapped React client.

## Daemon (API)
- **Prereqs**: Go 1.22+, FFmpeg installed
- **Install deps**: `go mod tidy`
- **Run**: `go run cmd/daemon/main.go` (listens on `:8080`)
- **Swagger docs**:
  - Generate: `swag init -g cmd/daemon/main.go -o internal/docs`
  - View UI: http://localhost:8080/swagger
- Env: `FRAMES_ROOT` optional (defaults to `frames/`)

## Client (Electron + Vite/React)
- **Prereqs**: Node.js 18+, npm, Electron-capable environment (WSLg or native desktop)
- **Install deps**: `cd client && npm install`
- **Dev (Electron + Vite)**: `npm run electron:dev`
  - Vite dev server fixed at http://localhost:5173; Electron launches with preload exposing `window.electronAPI`.
  - Use DevTools in the Electron window; `!!window.electronAPI` should be true.
- **Prod build (client-only)**: `npm run build` (outputs `dist/`; Electron packaging not wired yet)
- **File/folder selection**: Use the built-in pickers in the Video Library tab (no manual path entry); absolute paths are sent to the daemon. Recursive scan toggle is available.
