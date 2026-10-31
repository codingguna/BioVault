# Build script for Mantra MFS100 Fingerprint Manager
$ErrorActionPreference = "Stop"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " Building MFS100 Fingerprint Manager (32-bit)" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

$env:GOARCH = "386"
$env:GOOS = "windows"

Write-Host "`n[1/3] Running automated unit tests..." -ForegroundColor Yellow
go test ./internal/...

Write-Host "`n[2/3] Compiling fingerprint-manager.exe..." -ForegroundColor Yellow
go build -ldflags="-s -w" -o "fingerprint-manager.exe" ./cmd/fingerprint-manager

Write-Host "`n[3/3] Compiling probe.exe (hardware test CLI)..." -ForegroundColor Yellow
go build -ldflags="-s -w" -o "probe.exe" ./cmd/probe

Write-Host "`n[4/4] Compiling mfs100-auth-service.exe (Windows Background Auth Service)..." -ForegroundColor Yellow
go build -ldflags="-s -w" -o "mfs100-auth-service.exe" ./cmd/mfs100-auth-service

Write-Host "`nBUILD SUCCESSFUL!" -ForegroundColor Green
Get-Item "fingerprint-manager.exe", "probe.exe", "mfs100-auth-service.exe" | Format-Table Name, Length, LastWriteTime
