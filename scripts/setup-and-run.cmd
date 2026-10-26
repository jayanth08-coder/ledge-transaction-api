@echo off
REM Windows launcher — bypasses ExecutionPolicy for this script only (no permanent policy change).
cd /d "%~dp0\.."
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0setup-and-run.ps1" %*
