@echo off
setlocal enabledelayedexpansion
title Uninstalling Mantra MFS100 Credential Provider

net session >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo [!] Requesting Administrator privileges...
    powershell -Command "Start-Process cmd -ArgumentList '/c \"\"%~f0\"\"' -Verb RunAs"
    exit /b
)

echo ============================================================
echo   Uninstalling Mantra MFS100 Fingerprint Credential Provider
echo ============================================================
echo.

set "CLSID={7C6A8723-53BC-4545-98D1-9B5F9542D4C1}"
set "DEST_DLL=C:\Windows\System32\Mfs100CredentialProviderNative.dll"

echo [1/3] Removing from Credential Providers registry...
reg delete "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Authentication\Credential Providers\%CLSID%" /f >nul 2>&1
echo [+] Credential Provider key removed.

echo.
echo [2/3] Removing COM CLSID registration...
reg delete "HKLM\SOFTWARE\Classes\CLSID\%CLSID%" /f >nul 2>&1
echo [+] COM registration removed.

echo.
echo [3/3] Removing DLL from System32...
if exist "%DEST_DLL%" (
    del /f /q "%DEST_DLL%" >nul 2>&1
    echo [+] DLL deleted from System32.
) else (
    echo [*] DLL already removed.
)

echo.
echo ============================================================
echo  SUCCESS: Mantra Credential Provider uninstalled completely!
echo ============================================================
echo.
pause
