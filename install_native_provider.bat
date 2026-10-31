@echo off
setlocal enabledelayedexpansion
title Installing Mantra MFS100 Native Credential Provider

:: Check for Administrator elevation
net session >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo [!] Requesting Administrator privileges...
    powershell -Command "Start-Process cmd -ArgumentList '/c \"\"%~f0\"\"' -Verb RunAs"
    exit /b
)

echo ============================================================
echo   Mantra MFS100 Native Fingerprint Credential Provider Setup
echo ============================================================
echo.

set "SCRIPT_DIR=%~dp0"
set "SRC_DLL=%SCRIPT_DIR%native_cp\Mfs100CredentialProviderNative.dll"
set "DEST_DLL=C:\Windows\System32\Mfs100CredentialProviderNative.dll"
set "CLSID={7C6A8723-53BC-4545-98D1-9B5F9542D4C1}"

if not exist "%SRC_DLL%" (
    echo [-] Error: %SRC_DLL% not found!
    echo     Please run native_cp\build_native.bat first.
    pause
    exit /b 1
)

echo [1/4] Copying native 64-bit DLL to System32...
copy /Y "%SRC_DLL%" "%DEST_DLL%" >nul
if %ERRORLEVEL% NEQ 0 (
    echo [-] Failed to copy DLL to System32.
    pause
    exit /b %ERRORLEVEL%
)
echo [+] Copied to %DEST_DLL%

echo.
echo [2/4] Registering native COM InprocServer32...
:: Clean up old .NET keys first
reg delete "HKLM\SOFTWARE\Classes\CLSID\%CLSID%" /f >nul 2>&1

:: Register new native COM keys
reg add "HKLM\SOFTWARE\Classes\CLSID\%CLSID%" /ve /t REG_SZ /d "Mantra MFS100 Fingerprint Credential Provider" /f >nul
reg add "HKLM\SOFTWARE\Classes\CLSID\%CLSID%\InprocServer32" /ve /t REG_SZ /d "%DEST_DLL%" /f >nul
reg add "HKLM\SOFTWARE\Classes\CLSID\%CLSID%\InprocServer32" /v "ThreadingModel" /t REG_SZ /d "Apartment" /f >nul
echo [+] COM In-Process Server registered in HKLM.

echo.
echo [3/4] Registering in Windows Authentication Credential Providers...
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Authentication\Credential Providers\%CLSID%" /ve /t REG_SZ /d "Mfs100CredentialProviderNative" /f >nul
echo [+] Windows Credential Provider active in LogonUI.

echo.
echo [4/4] Verifying background auth daemon...
tasklist /FI "IMAGENAME eq mfs100-auth-service.exe" 2>nul | findstr /I "mfs100-auth-service.exe" >nul
if %ERRORLEVEL% NEQ 0 (
    echo [!] Auth service is not running. Starting silent background daemon...
    start "" /B "%SCRIPT_DIR%mfs100-auth-service.exe" -db "%SCRIPT_DIR%mfs100.db"
    echo [+] Auth service started in background.
) else (
    echo [+] Auth service is already running silently in the background.
)

echo.
echo ============================================================
echo  SUCCESS! Mantra MFS100 Native Credential Provider is INSTALLED!
echo ============================================================
echo.
echo - Your lock screen (Win + L) will now display the Mantra Fingerprint tile!
echo - Built-in PIN and Password sign-in options remain 100%% unblocked.
echo - To test now: Press Win + L to lock your screen and touch the scanner.
echo - To remove at any time: run uninstall_native_provider.bat
echo.
pause
