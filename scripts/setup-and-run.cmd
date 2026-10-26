@echo off
setlocal
REM Windows launcher - bypasses ExecutionPolicy for this script only.
cd /d "%~dp0.."
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0setup-and-run.ps1" %*
exit /b %ERRORLEVEL%
