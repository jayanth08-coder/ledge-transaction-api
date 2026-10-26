#!/usr/bin/env bash
# setup-and-run.sh — install prerequisites (if missing), reset demo DB, seed data, run the ledger API.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

APP_HOST="${APP_HOST:-127.0.0.1}"
APP_PORT="${APP_PORT:-8080}"
DATABASE_PATH="${DATABASE_PATH:-data/ledger.db}"
SEED_DEMO="${SEED_DEMO:-true}"
RESET_DB="${RESET_DB:-true}"

log()  { printf '\n==> %s\n' "$*"; }
ok()   { printf '    [OK] %s\n' "$*"; }
warn() { printf '    ! %s\n' "$*" >&2; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

ensure_path_go_bin() {
  local gobin
  gobin="$(go env GOPATH 2>/dev/null)/bin"
  if [[ -n "${gobin}" && -d "${gobin}" ]]; then
    case ":$PATH:" in
      *":${gobin}:"*) ;;
      *) export PATH="${gobin}:$PATH" ;;
    esac
  fi
}

install_go_linux() {
  if need_cmd apt-get; then
    sudo apt-get update -y
    sudo apt-get install -y golang-go
  elif need_cmd dnf; then
    sudo dnf install -y golang
  elif need_cmd yum; then
    sudo yum install -y golang
  elif need_cmd pacman; then
    sudo pacman -Sy --noconfirm go
  else
    warn "No supported package manager found. Install Go from https://go.dev/dl/"
    exit 1
  fi
}

install_go_macos() {
  if need_cmd brew; then
    brew install go
  else
    warn "Homebrew not found. Install Go from https://go.dev/dl/ or install Homebrew first."
    exit 1
  fi
}

ensure_go() {
  if need_cmd go; then
    ok "Go $(go version | awk '{print $3}')"
    return
  fi
  log "Go not found — attempting install..."
  case "$(uname -s)" in
    Linux*)  install_go_linux ;;
    Darwin*) install_go_macos ;;
    MINGW*|MSYS*|CYGWIN*)
      warn "On Windows/Git Bash, install Go with: winget install GoLang.Go"
      warn "Then reopen the terminal and re-run this script."
      exit 1
      ;;
    *)
      warn "Unsupported OS for automatic Go install. See https://go.dev/dl/"
      exit 1
      ;;
  esac
  if ! need_cmd go; then
    warn "Go still not on PATH after install."
    exit 1
  fi
  ok "Go installed: $(go version)"
}

ensure_templ() {
  ensure_path_go_bin
  if need_cmd templ; then
    ok "templ CLI available"
    return
  fi
  log "Installing templ CLI..."
  go install github.com/a-h/templ/cmd/templ@v0.3.906
  ensure_path_go_bin
  if ! need_cmd templ; then
    warn "templ installed but not on PATH. Add \"$(go env GOPATH)/bin\" to PATH."
    exit 1
  fi
  ok "templ installed"
}

ensure_curl() {
  if need_cmd curl; then
    ok "curl available"
    return
  fi
  log "Installing curl..."
  case "$(uname -s)" in
    Linux*)
      if need_cmd apt-get; then sudo apt-get install -y curl
      elif need_cmd dnf; then sudo dnf install -y curl
      elif need_cmd yum; then sudo yum install -y curl
      fi
      ;;
    Darwin*)
      need_cmd brew && brew install curl || true
      ;;
  esac
  need_cmd curl || { warn "curl is required"; exit 1; }
  ok "curl ready"
}

reset_database() {
  mkdir -p data
  if [[ "${RESET_DB}" == "true" ]]; then
    log "Clearing SQLite database files..."
    rm -f data/*.db data/*.db-wal data/*.db-shm 2>/dev/null || true
    ok "Database cleared (${DATABASE_PATH})"
  else
    ok "Skipping DB reset (RESET_DB=${RESET_DB})"
  fi
}

prepare_modules() {
  log "Downloading Go modules..."
  go mod download
  go mod tidy
  ok "Modules ready"

  log "Generating Templ code..."
  templ generate
  ok "Templ generated"
}

print_banner() {
  cat <<EOF

============================================================
  Transaction Ledger API
  UI:   http://${APP_HOST}:${APP_PORT}/
  API:  http://${APP_HOST}:${APP_PORT}/api
  Docs: API.md  |  ASSUMPTIONS.md
  DB:   ${DATABASE_PATH}   SEED_DEMO=${SEED_DEMO}
============================================================
  Demo accounts (after seed):
    demo-account-alice    Alice Sharma     ₹120
    demo-account-bob      Bob Kumar        ₹80
    demo-account-cara     Cara Patel       ₹25
    demo-account-dev      Dev Ops Ledger   ₹0
    demo-account-eva      Eva Nair         ₹200
    demo-account-finance  Finance Pool     ₹150
============================================================

EOF
}

main() {
  log "Project root: ${ROOT}"
  ensure_go
  ensure_templ
  ensure_curl
  reset_database
  prepare_modules
  print_banner

  export APP_HOST APP_PORT DATABASE_PATH SEED_DEMO
  export APP_ENV="${APP_ENV:-development}"
  export LOG_LEVEL="${LOG_LEVEL:-info}"

  log "Starting server (Ctrl+C to stop)..."
  exec go run ./cmd/server
}

main "$@"
