# Registration script for MFS100 Windows Credential Provider
# MUST BE RUN AS ADMINISTRATOR
$ErrorActionPreference = "Stop"

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Warning "This script requires Administrator privileges. Please right-click PowerShell and select 'Run as Administrator'."
    exit 1
}

$dllPath = "$PSScriptRoot\Mfs100CredentialProvider.dll"
$regasm = "C:\Windows\Microsoft.NET\Framework64\v4.0.30319\regasm.exe"
$clsid = "{7C6A8723-53BC-4545-98D1-9B5F9542D4C1}"

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "   Registering MFS100 Windows Fingerprint Credential Provider" -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# 1. Register with 64-bit regasm for COM
Write-Host "`n[1/2] Registering COM In-Process Server via 64-bit regasm..." -ForegroundColor Yellow
& $regasm $dllPath /codebase /silent
Write-Host "[+] COM Server Registered." -ForegroundColor Green

# 2. Register with Windows LogonUI Credential Providers
Write-Host "`n[2/2] Registering in Windows Authentication Credential Providers..." -ForegroundColor Yellow
$regKey = "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Authentication\Credential Providers\$clsid"
if (-not (Test-Path $regKey)) {
    New-Item -Path $regKey -Force | Out-Null
}
Set-ItemProperty -Path $regKey -Name "(Default)" -Value "Mfs100Provider"
Write-Host "[+] Credential Provider Registered in HKLM." -ForegroundColor Green

Write-Host "`n==========================================================" -ForegroundColor Green
Write-Host " SUCCESS: MFS100 Credential Provider is now active!" -ForegroundColor Green
Write-Host " Fallback PIN and Password sign-in remain 100% available." -ForegroundColor Yellow
Write-Host " To unregister at any time, run: .\unregister_unlock.ps1" -ForegroundColor Yellow
Write-Host "==========================================================" -ForegroundColor Green
