#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="install"
FORCE=0
MISSING=0

usage() {
  cat <<'EOF'
Usage: ./scripts/setup.sh [--check | --dry-run] [--force]

  --check    Verify the development environment without changing it.
  --dry-run  Show what setup would change without changing it.
  --force    Reinstall project dependencies and rebuild the Docker image.
EOF
}

for arg in "$@"; do
  case "$arg" in
    --check) MODE="check" ;;
    --dry-run) MODE="dry-run" ;;
    --force) FORCE=1 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $arg" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ "$MODE" == "check" && "$FORCE" == "1" ]]; then
  echo "--force cannot be combined with --check." >&2
  exit 2
fi

info() { printf '\033[1;34m→\033[0m %s\n' "$*"; }
ok() { printf '\033[1;32m✓\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m✗\033[0m %s\n' "$*"; MISSING=$((MISSING + 1)); }

version_at_least() {
  awk -v current="$1" -v minimum="$2" 'BEGIN {
    split(current, a, "."); split(minimum, b, ".");
    for (i = 1; i <= 3; i++) {
      av = (a[i] == "" ? 0 : a[i]) + 0;
      bv = (b[i] == "" ? 0 : b[i]) + 0;
      if (av > bv) exit 0;
      if (av < bv) exit 1;
    }
    exit 0;
  }'
}

BREW_BIN=""
find_brew() {
  if command -v brew >/dev/null 2>&1; then
    BREW_BIN="$(command -v brew)"
  elif [[ -x /opt/homebrew/bin/brew ]]; then
    BREW_BIN="/opt/homebrew/bin/brew"
  elif [[ -x /usr/local/bin/brew ]]; then
    BREW_BIN="/usr/local/bin/brew"
  fi
}

ensure_homebrew() {
  find_brew
  if [[ -n "$BREW_BIN" ]]; then
    export PATH="$(dirname "$BREW_BIN"):$PATH"
    return
  fi

  if [[ "$MODE" == "dry-run" ]]; then
    info "Would install Homebrew from brew.sh"
    return
  fi

  info "Installing Homebrew (macOS may request your password)"
  installer="$(mktemp)"
  curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh -o "$installer"
  /bin/bash "$installer"
  rm -f "$installer"
  find_brew
  if [[ -z "$BREW_BIN" ]]; then
    echo "Homebrew installed but could not be found. Open a new terminal and rerun setup." >&2
    exit 1
  fi
  export PATH="$(dirname "$BREW_BIN"):$PATH"
}

brew_install_or_upgrade() {
  package="$1"
  if [[ "$MODE" == "dry-run" ]]; then
    info "Would install or upgrade $package with Homebrew"
    return
  fi
  ensure_homebrew
  if "$BREW_BIN" list --formula "$package" >/dev/null 2>&1; then
    "$BREW_BIN" upgrade "$package"
  else
    "$BREW_BIN" install "$package"
  fi
}

ensure_versioned_command() {
  command_name="$1"
  package="$2"
  minimum="$3"
  version="$4"

  if command -v "$command_name" >/dev/null 2>&1 && version_at_least "$version" "$minimum"; then
    ok "$command_name $version satisfies $minimum+"
    return
  fi

  if [[ "$MODE" == "check" ]]; then
    if command -v "$command_name" >/dev/null 2>&1; then
      fail "$command_name $version is older than $minimum"
    else
      fail "$command_name is not installed"
    fi
    return
  fi

  brew_install_or_upgrade "$package"
}

ensure_simple_command() {
  command_name="$1"
  package="$2"
  if command -v "$command_name" >/dev/null 2>&1; then
    ok "$command_name is installed"
    return
  fi
  if [[ "$MODE" == "check" ]]; then
    fail "$command_name is not installed"
    return
  fi
  brew_install_or_upgrade "$package"
}

configure_docker_path() {
  for docker_bin_dir in "$HOME/.docker/bin" "/Applications/Docker.app/Contents/Resources/bin"; do
    if [[ -d "$docker_bin_dir" ]]; then
      case ":$PATH:" in
        *":$docker_bin_dir:"*) ;;
        *) export PATH="$docker_bin_dir:$PATH" ;;
      esac
    fi
  done
}

ensure_docker() {
  if command -v docker >/dev/null 2>&1; then
    ok "Docker is installed"
  else
    if [[ "$MODE" == "dry-run" ]]; then
      warn "Docker Desktop is not installed and must be installed manually"
    else
      fail "Docker Desktop is not installed; install and open it, then rerun setup"
    fi
    return
  fi

  docker_config_file="${DOCKER_CONFIG:-$HOME/.docker}/config.json"
  if [[ -f "$docker_config_file" ]] && grep -Eq '"credsStore"[[:space:]]*:[[:space:]]*"desktop"' "$docker_config_file"; then
    if command -v docker-credential-desktop >/dev/null 2>&1; then
      ok "Docker Desktop credential helper is available"
    elif [[ "$MODE" == "dry-run" ]]; then
      warn "Docker is configured to use docker-credential-desktop, but the helper was not found"
    else
      fail "Docker Desktop credential helper was not found; reopen Docker Desktop, then rerun setup"
      return
    fi
  fi

  if [[ "$MODE" == "check" ]]; then
    if docker info >/dev/null 2>&1; then
      ok "Docker Desktop is running"
    else
      fail "Docker Desktop is installed but not running"
    fi
  fi
}

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "This setup script currently supports macOS only." >&2
  exit 1
fi

info "Checking system tools"
find_brew
if [[ -n "$BREW_BIN" ]]; then
  export PATH="$(dirname "$BREW_BIN"):$PATH"
fi
configure_docker_path

go_version="0"
if command -v go >/dev/null 2>&1; then
  go_version="$(go version | awk '{print $3}' | sed 's/^go//')"
fi
ensure_versioned_command "go" "go" "1.22" "$go_version"

node_version="0"
if command -v node >/dev/null 2>&1; then
  node_version="$(node --version | sed 's/^v//')"
fi
ensure_versioned_command "node" "node" "18" "$node_version"
ensure_simple_command "ffmpeg" "ffmpeg"
ensure_docker

if [[ "$MODE" == "install" && "$MISSING" -gt 0 ]]; then
  echo
  echo "Setup cannot continue until Docker Desktop is installed and opened." >&2
  echo "Download it from https://www.docker.com/products/docker-desktop/" >&2
  exit 1
fi

if [[ "$MODE" == "check" ]]; then
  if command -v npm >/dev/null 2>&1; then
    ok "npm is installed"
  else
    fail "npm is not installed"
  fi

  if (cd "$ROOT_DIR/client" && npm ls --depth=0 >/dev/null 2>&1); then
    ok "Client dependencies are installed"
  else
    fail "Client dependencies are missing or do not match package-lock.json"
  fi

  vector_hash="$(shasum -a 256 "$ROOT_DIR/vectordb/Dockerfile" "$ROOT_DIR/vectordb/requirements.txt" "$ROOT_DIR/vectordb/main.py" | shasum -a 256 | awk '{print $1}')"
  image_hash="$(docker image inspect --format '{{ index .Config.Labels "com.semantic-video.source-hash" }}' semantic-video:dev 2>/dev/null || true)"
  if [[ "$image_hash" == "$vector_hash" ]]; then
    ok "Vector service image matches the source"
  elif [[ -n "$image_hash" ]]; then
    fail "Vector service image is out of date"
  else
    fail "Vector service image semantic-video:dev is missing"
  fi

  if [[ "$MISSING" -gt 0 ]]; then
    echo
    echo "$MISSING setup check(s) failed. Run ./scripts/setup.sh to fix them." >&2
    exit 1
  fi
  echo
  ok "Development environment is ready"
  exit 0
fi

if [[ "$MODE" == "dry-run" ]]; then
  info "Would download Go modules"
  if (cd "$ROOT_DIR/client" && npm ls --depth=0 >/dev/null 2>&1); then
    ok "Client dependencies already match package-lock.json"
  else
    info "Would install locked client dependencies"
  fi
  if docker info >/dev/null 2>&1; then
    vector_hash="$(shasum -a 256 "$ROOT_DIR/vectordb/Dockerfile" "$ROOT_DIR/vectordb/requirements.txt" "$ROOT_DIR/vectordb/main.py" | shasum -a 256 | awk '{print $1}')"
    image_hash="$(docker image inspect --format '{{ index .Config.Labels "com.semantic-video.source-hash" }}' semantic-video:dev 2>/dev/null || true)"
    if [[ "$image_hash" == "$vector_hash" ]]; then
      ok "Vector service image already matches the source"
    else
      info "Would build semantic-video:dev from vectordb/Dockerfile"
    fi
  else
    info "Would start the already-installed Docker Desktop and build the vector service image if needed"
  fi
  echo
  ok "Dry run complete; no changes were made"
  exit 0
fi

info "Installing project dependencies"
if [[ "$FORCE" == "1" ]] || ! (cd "$ROOT_DIR/client" && npm ls --depth=0 >/dev/null 2>&1); then
  (cd "$ROOT_DIR/client" && npm ci)
else
  ok "Client dependencies are already installed"
fi
(cd "$ROOT_DIR" && go mod download)

if ! docker info >/dev/null 2>&1; then
  info "Starting Docker Desktop"
  open -a Docker
  for _ in $(seq 1 60); do
    if docker info >/dev/null 2>&1; then
      break
    fi
    sleep 2
  done
fi
if ! docker info >/dev/null 2>&1; then
  echo "Docker Desktop did not become ready. Finish its first-run prompts, then rerun setup." >&2
  exit 1
fi
ok "Docker Desktop is running"

vector_hash="$(shasum -a 256 "$ROOT_DIR/vectordb/Dockerfile" "$ROOT_DIR/vectordb/requirements.txt" "$ROOT_DIR/vectordb/main.py" | shasum -a 256 | awk '{print $1}')"
image_hash="$(docker image inspect --format '{{ index .Config.Labels "com.semantic-video.source-hash" }}' semantic-video:dev 2>/dev/null || true)"
if [[ "$FORCE" == "1" || "$image_hash" != "$vector_hash" ]]; then
  info "Building the vector service image (the first build can take several minutes)"
  docker build \
    --label "com.semantic-video.source-hash=$vector_hash" \
    -t semantic-video:dev \
    "$ROOT_DIR/vectordb"
else
  ok "Vector service image already matches the source"
fi

echo
ok "Setup complete"
echo "Run ./scripts/dev.sh to start Semantic Video."
