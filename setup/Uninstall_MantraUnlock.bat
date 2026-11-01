@echo off
setlocal enabledelayedexpansion

:: Check for Administrator elevation
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [!] Administrator privileges required.
    echo [*] Attempting to elevate...
    powershell -Command "Start-Process cmd -ArgumentList '/c \"\"%~f0\"\"' -Verb RunAs"
    exit /b
)

echo ========================================================
echo   Mantra MFS100 Biometric Windows Logon Uninstaller
echo ========================================================
echo.

:: 1. Terminate running processes
echo [*] Terminating active processes...
taskkill /f /im mfs100-auth-service.exe >nul 2>&1
taskkill /f /im MantraFingerprintManager.exe >nul 2>&1
taskkill /f /im fingerprint-manager.exe >nul 2>&1

:: 2. Stop and remove scheduled task
echo [*] Removing background authentication service...
schtasks /end /tn "MFS100AuthService" >nul 2>&1
schtasks /delete /f /tn "MFS100AuthService" >nul 2>&1

:: 3. Unregister Credential Provider from Windows Registry
echo [*] Unregistering Windows Credential Provider...
reg delete "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Authentication\Credential Providers\{7B1E4398-75F9-4670-9831-C58DB1F25832}" /f >nul 2>&1
reg delete "HKLM\SOFTWARE\Classes\CLSID\{7B1E4398-75F9-4670-9831-C58DB1F25832}" /f >nul 2>&1

:: 4. Remove Credential Provider DLL from System32
echo [*] Removing Mfs100CredentialProviderNative.dll from System32...
if exist "C:\Windows\System32\Mfs100CredentialProviderNative.dll" (
    del /f /q "C:\Windows\System32\Mfs100CredentialProviderNative.dll" >nul 2>&1
)

:: 5. Remove Program Files directory
echo [*] Removing files from Program Files...
if exist "C:\Program Files\MantraUnlock" (
    rmdir /s /q "C:\Program Files\MantraUnlock" >nul 2>&1
)

:: 6. Remove Shortcuts
echo [*] Removing shortcuts...
del /f /q "%ProgramData%\Microsoft\Windows\Start Menu\Programs\Mantra Fingerprint Security\*.*" >nul 2>&1
rmdir /s /q "%ProgramData%\Microsoft\Windows\Start Menu\Programs\Mantra Fingerprint Security" >nul 2>&1
del /f /q "%Public%\Desktop\Mantra Fingerprint Manager.lnk" >nul 2>&1

:: 7. Optional Database removal
echo.
set /p DEL_DB="Do you also want to permanently delete saved fingerprints & credentials in C:\ProgramData\MantraUnlock? (y/N): "
if /i "%DEL_DB%"=="y" (
    echo [*] Deleting biometric vault...
    rmdir /s /q "C:\ProgramData\MantraUnlock" >nul 2>&1
    echo [+] Biometric data removed.
) else (
    echo [i] Biometric vault preserved in C:\ProgramData\MantraUnlock.
)

echo.
echo ========================================================
echo [+] Mantra Biometric Logon has been completely uninstalled.
echo     Windows has reverted to standard PIN and Password login.
echo ========================================================
pause
