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
set "EXE_FILE=%SCRIPT_DIR%mfs100-auth-service.exe"
set "DB_FILE=%SCRIPT_DIR%mfs100.db"
set "CLSID={7C6A8723-53BC-4545-98D1-9B5F9542D4C1}"

if not exist "%SRC_DLL%" (
    echo [-] Error: %SRC_DLL% not found!
    pause
    exit /b 1
)

echo [1/5] Stopping existing background auth processes...
taskkill /F /IM mfs100-auth-service.exe >nul 2>&1
taskkill /F /IM fingerprint-manager.exe >nul 2>&1
timeout /t 1 >nul 2>&1

:: If updated service binary exists, replace it
if exist "%SCRIPT_DIR%mfs100-auth-service-new.exe" (
    move /Y "%SCRIPT_DIR%mfs100-auth-service-new.exe" "%EXE_FILE%" >nul 2>&1
    echo [+] Replaced mfs100-auth-service.exe with latest build.
)

echo [2/5] Installing Boot-Time Background Service (Task Scheduler)...
:: Register Scheduled Task to run under SYSTEM at startup before any user logs in
schtasks /create /tn "MFS100AuthService" /tr "\"%EXE_FILE%\" -db \"%DB_FILE%\"" /sc ONSTART /ru "NT AUTHORITY\SYSTEM" /rl HIGHEST /f >nul
if %ERRORLEVEL% EQU 0 (
    echo [+] Scheduled task 'MFS100AuthService' registered to start at Windows Boot.
    powershell -Command "$s = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit (New-TimeSpan -Days 365); Set-ScheduledTask -TaskName 'MFS100AuthService' -Settings $s" >nul 2>&1
    schtasks /run /tn "MFS100AuthService" >nul 2>&1
) else (
    echo [!] Warning: Could not register startup task, starting process directly...
    start "" /B "%EXE_FILE%" -db "%DB_FILE%"
)

:: Remove old HKCU Run key to avoid duplicate processes
reg delete "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "MFS100AuthService" /f >nul 2>&1

echo.
echo [3/5] Copying native 64-bit DLL to System32...
copy /Y "%SRC_DLL%" "%DEST_DLL%" >nul
if %ERRORLEVEL% NEQ 0 (
    echo [-] Failed to copy DLL to System32. Make sure LogonUI is closed.
    pause
    exit /b %ERRORLEVEL%
)
echo [+] Copied to %DEST_DLL%

echo.
echo [4/5] Registering native COM InprocServer32...
reg delete "HKLM\SOFTWARE\Classes\CLSID\%CLSID%" /f >nul 2>&1
reg add "HKLM\SOFTWARE\Classes\CLSID\%CLSID%" /ve /t REG_SZ /d "Mantra MFS100 Fingerprint Credential Provider" /f >nul
reg add "HKLM\SOFTWARE\Classes\CLSID\%CLSID%\InprocServer32" /ve /t REG_SZ /d "%DEST_DLL%" /f >nul
reg add "HKLM\SOFTWARE\Classes\CLSID\%CLSID%\InprocServer32" /v "ThreadingModel" /t REG_SZ /d "Apartment" /f >nul
echo [+] COM In-Process Server registered in HKLM.

echo.
echo [5/5] Registering in Windows Authentication Credential Providers...
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Authentication\Credential Providers\%CLSID%" /ve /t REG_SZ /d "Mfs100CredentialProviderNative" /f >nul
echo [+] Windows Credential Provider active in LogonUI.

echo.
echo ============================================================
echo  SUCCESS! Mantra MFS100 Native Credential Provider is READY!
echo ============================================================
echo.
echo  Changes Applied:
echo  1. "The parameter is incorrect" bug FIXED (packed relative string offsets).
echo  2. Loop scanning removed - click the arrow button ("Scan Finger") to scan!
echo  3. Automatic Boot Service configured (starts silently on Windows startup).
echo.
echo  To Test: Press Win + L to lock screen, then click 'Scan Finger' and touch sensor!
echo  To Remove: Run uninstall_native_provider.bat as Administrator anytime.
echo.
pause
