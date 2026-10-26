# setup-and-run.ps1 - Windows helper: install deps if missing, reset DB, seed, run.
# ASCII-only so Windows PowerShell 5.1 parses this after ZIP download on any locale.
$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $Root

if (-not $env:APP_HOST) { $env:APP_HOST = "127.0.0.1" }
if (-not $env:APP_PORT) { $env:APP_PORT = "8080" }
if (-not $env:DATABASE_PATH) { $env:DATABASE_PATH = "data/ledger.db" }
if (-not $env:SEED_DEMO) { $env:SEED_DEMO = "true" }
if (-not $env:RESET_DB) { $env:RESET_DB = "true" }

function Write-Step {
  param([string]$Message)
  Write-Host ""
  Write-Host ("==> " + $Message) -ForegroundColor Cyan
}

function Write-Ok {
  param([string]$Message)
  Write-Host ("    [OK] " + $Message) -ForegroundColor Green
}

function Refresh-Path {
  $machine = [System.Environment]::GetEnvironmentVariable("Path", "Machine")
  $user = [System.Environment]::GetEnvironmentVariable("Path", "User")
  $goBin = Join-Path $env:USERPROFILE "go\bin"
  $parts = @()
  if ($machine) { $parts += $machine }
  if ($user) { $parts += $user }
  $parts += $goBin
  $env:Path = ($parts -join ";")
}

Refresh-Path

Write-Step "Checking Go"
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  Write-Host "Go not found. Installing via winget..." -ForegroundColor Yellow
  winget install GoLang.Go --accept-package-agreements --accept-source-agreements
  Refresh-Path
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw "Go is still not available. Install from https://go.dev/dl/ and reopen PowerShell."
}
Write-Ok (go version)

Write-Step "Checking templ CLI"
if (-not (Get-Command templ -ErrorAction SilentlyContinue)) {
  go install github.com/a-h/templ/cmd/templ@v0.3.906
  Refresh-Path
}
if (-not (Get-Command templ -ErrorAction SilentlyContinue)) {
  $hint = Join-Path $env:USERPROFILE "go\bin"
  throw ("templ not on PATH. Add " + $hint + " to PATH.")
}
Write-Ok "templ available"

Write-Step "Database reset"
New-Item -ItemType Directory -Force -Path data | Out-Null
if ($env:RESET_DB -eq "true") {
  Get-ChildItem -Path data -Filter "*.db*" -ErrorAction SilentlyContinue | Remove-Item -Force
  Write-Ok "Cleared data/*.db*"
} else {
  Write-Ok ("Skipped reset (RESET_DB=" + $env:RESET_DB + ")")
}

Write-Step "Modules + Templ"
go mod download
go mod tidy
templ generate
Write-Ok "Ready"

Write-Host ""
Write-Host "============================================================"
Write-Host "  Transaction Ledger API"
Write-Host ("  UI:   http://" + $env:APP_HOST + ":" + $env:APP_PORT + "/")
Write-Host ("  API:  http://" + $env:APP_HOST + ":" + $env:APP_PORT + "/api")
Write-Host "  Docs: API.md  |  ASSUMPTIONS.md"
Write-Host ("  SEED_DEMO=" + $env:SEED_DEMO)
Write-Host "============================================================"
Write-Host "  Demo accounts:"
Write-Host "    demo-account-alice    Alice Sharma     Rs 120"
Write-Host "    demo-account-bob      Bob Kumar        Rs 80"
Write-Host "    demo-account-cara     Cara Patel       Rs 25"
Write-Host "    demo-account-dev      Dev Ops Ledger   Rs 0"
Write-Host "    demo-account-eva      Eva Nair         Rs 200"
Write-Host "    demo-account-finance  Finance Pool     Rs 150"
Write-Host "============================================================"
Write-Host ""

Write-Step "Starting server (Ctrl+C to stop)"
go run ./cmd/server
