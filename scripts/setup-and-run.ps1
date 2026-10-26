# setup-and-run.ps1 — Windows helper: install deps if missing, reset DB, seed, run.
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$env:APP_HOST = if ($env:APP_HOST) { $env:APP_HOST } else { "127.0.0.1" }
$env:APP_PORT = if ($env:APP_PORT) { $env:APP_PORT } else { "8080" }
$env:DATABASE_PATH = if ($env:DATABASE_PATH) { $env:DATABASE_PATH } else { "data/ledger.db" }
$env:SEED_DEMO = if ($env:SEED_DEMO) { $env:SEED_DEMO } else { "true" }
$ResetDb = if ($env:RESET_DB) { $env:RESET_DB } else { "true" }

function Write-Step($msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }
function Write-Ok($msg) { Write-Host "    ✓ $msg" -ForegroundColor Green }

# Refresh PATH (Machine + User + Go bin)
$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
            [System.Environment]::GetEnvironmentVariable("Path", "User") + ";" +
            "$env:USERPROFILE\go\bin"

Write-Step "Checking Go"
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  Write-Host "Go not found. Installing via winget..." -ForegroundColor Yellow
  winget install GoLang.Go --accept-package-agreements --accept-source-agreements
  $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
              [System.Environment]::GetEnvironmentVariable("Path", "User") + ";" +
              "$env:USERPROFILE\go\bin"
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw "Go is still not available. Install from https://go.dev/dl/ and reopen PowerShell."
}
Write-Ok (go version)

Write-Step "Checking templ CLI"
if (-not (Get-Command templ -ErrorAction SilentlyContinue)) {
  go install github.com/a-h/templ/cmd/templ@v0.3.906
  $env:Path = "$env:USERPROFILE\go\bin;" + $env:Path
}
if (-not (Get-Command templ -ErrorAction SilentlyContinue)) {
  throw "templ not on PATH. Add $env:USERPROFILE\go\bin to PATH."
}
Write-Ok "templ available"

Write-Step "Database reset"
New-Item -ItemType Directory -Force -Path data | Out-Null
if ($ResetDb -eq "true") {
  Get-ChildItem data -Filter "*.db*" -ErrorAction SilentlyContinue | Remove-Item -Force
  Write-Ok "Cleared data/*.db*"
} else {
  Write-Ok "Skipped reset (RESET_DB=$ResetDb)"
}

Write-Step "Modules + Templ"
go mod download
go mod tidy
templ generate
Write-Ok "Ready"

Write-Host @"

============================================================
  Transaction Ledger API
  UI:   http://$($env:APP_HOST):$($env:APP_PORT)/
  API:  http://$($env:APP_HOST):$($env:APP_PORT)/api
  Docs: docs/API.md  |  docs/ASSUMPTIONS.md
  SEED_DEMO=$($env:SEED_DEMO)
============================================================
  Demo accounts:
    demo-account-alice    Alice Sharma     Rs 120
    demo-account-bob      Bob Kumar        Rs 80
    demo-account-cara     Cara Patel       Rs 25
    demo-account-dev      Dev Ops Ledger   Rs 0
    demo-account-eva      Eva Nair         Rs 200
    demo-account-finance  Finance Pool     Rs 150
============================================================

"@

Write-Step "Starting server (Ctrl+C to stop)"
go run ./cmd/server
