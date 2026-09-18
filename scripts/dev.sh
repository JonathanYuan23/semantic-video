#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONTAINER_NAME="semantic-video-vectordb"
LOG_DIR="$(mktemp -d "${TMPDIR:-/tmp}/semantic-video-dev.XXXXXX")"
VECTOR_STARTED=0
DAEMON_PID=""

cleanup() {
  exit_code=$?
  trap - EXIT INT TERM

  if [[ -n "$DAEMON_PID" ]] && kill -0 "$DAEMON_PID" >/dev/null 2>&1; then
    kill "$DAEMON_PID" >/dev/null 2>&1 || true
    wait "$DAEMON_PID" >/dev/null 2>&1 || true
  fi
  if [[ "$VECTOR_STARTED" == "1" ]]; then
    docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
  fi
  rm -rf "$LOG_DIR"
  exit "$exit_code"
}
trap cleanup EXIT INT TERM

wait_for_url() {
  name="$1"
  url="$2"
  log_file="$3"
  attempts="$4"

  for _ in $(seq 1 "$attempts"); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done

  echo "$name did not become ready at $url." >&2
  if [[ -f "$log_file" ]]; then
    echo "Last $name log lines:" >&2
    tail -30 "$log_file" >&2
  fi
  return 1
}

listener_pid() {
  lsof -tiTCP:"$1" -sTCP:LISTEN 2>/dev/null | head -n 1 || true
}

process_working_directory() {
  lsof -a -p "$1" -d cwd -Fn 2>/dev/null | sed -n 's/^n//p' | head -n 1 || true
}

"$ROOT_DIR/scripts/setup.sh"

if curl -fsS http://localhost:8000/docs >/dev/null 2>&1; then
  echo "✓ Reusing the vector service already running on port 8000"
else
  running_id="$(docker ps -q --filter "name=^/${CONTAINER_NAME}$")"
  existing_id="$(docker ps -aq --filter "name=^/${CONTAINER_NAME}$")"
  if [[ -n "$running_id" ]]; then
    echo "→ Waiting for the existing vector service"
    wait_for_url "Vector service" "http://localhost:8000/docs" "$LOG_DIR/vectordb.log" 150 || {
      docker logs "$CONTAINER_NAME" >&2 || true
      exit 1
    }
  elif [[ -n "$existing_id" ]]; then
    echo "Removing the stopped development container"
    docker rm "$CONTAINER_NAME" >/dev/null
  fi

  if [[ -z "$running_id" ]]; then
    echo "→ Starting the vector service"
    docker run --rm -d \
      --name "$CONTAINER_NAME" \
      -e STATELESS_MODE=1 \
      -p 8000:8000 \
      -v semantic-video-model-cache:/root/.cache/huggingface \
      semantic-video:dev >/dev/null
    VECTOR_STARTED=1

    # The embedding model is downloaded on first launch, so allow extra time.
    wait_for_url "Vector service" "http://localhost:8000/docs" "$LOG_DIR/vectordb.log" 150 || {
      docker logs "$CONTAINER_NAME" >&2 || true
      exit 1
    }
  fi
  echo "✓ Vector service is ready"
fi

if curl -fsS http://localhost:8080/health >/dev/null 2>&1; then
  echo "✓ Reusing the Go API already running on port 8080"
else
  echo "→ Starting the Go API"
  (
    cd "$ROOT_DIR"
    go build -o "$LOG_DIR/semantic-video-daemon" ./cmd/daemon
  )
  STATELESS_MODE=1 VECTORDB_URL=http://localhost:8000 \
    "$LOG_DIR/semantic-video-daemon" >"$LOG_DIR/daemon.log" 2>&1 &
  DAEMON_PID=$!
  wait_for_url "Go API" "http://localhost:8080/health" "$LOG_DIR/daemon.log" 30
  echo "✓ Go API is ready"
fi

echo "Press Ctrl-C to stop services started by this script."

vite_pid="$(listener_pid 5173)"
if [[ -n "$vite_pid" ]]; then
  vite_cwd="$(process_working_directory "$vite_pid")"
  if [[ "$vite_cwd" != "$ROOT_DIR/client" ]]; then
    echo "Port 5173 is already used by a process outside this project's client directory." >&2
    echo "Stop that process, then rerun ./scripts/dev.sh." >&2
    exit 1
  fi

  echo "✓ Reusing the Vite server already running on port 5173"
  echo "→ Starting the Electron client"
  (
    cd "$ROOT_DIR/client"
    ELECTRON_START_URL=http://localhost:5173 ./node_modules/.bin/electron .
  )
else
  echo "→ Starting the Vite server and Electron client"
  (cd "$ROOT_DIR/client" && npm run electron:dev)
fi
