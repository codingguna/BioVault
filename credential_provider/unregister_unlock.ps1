# Rollback script to remove MFS100 Windows Credential Provider
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
Write-Host "   Unregistering MFS100 Windows Credential Provider       " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# 1. Remove registry key from Windows Authentication
$regKey = "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Authentication\Credential Providers\$clsid"
if (Test-Path $regKey) {
    Remove-Item -Path $regKey -Recurse -Force
    Write-Host "[+] Removed from Windows Authentication Credential Providers." -ForegroundColor Green
}

# 2. Unregister COM server via regasm /unregister
if (Test-Path $dllPath) {
    & $regasm $dllPath /unregister /silent
    Write-Host "[+] Unregistered COM Server." -ForegroundColor Green
}

Write-Host "`n[+] Rollback complete. Normal Windows sign-in fully restored." -ForegroundColor Green
