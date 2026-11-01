@echo off
net session >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    powershell -Command "Start-Process cmd -ArgumentList '/c \"\"%~f0\"\"' -Verb RunAs"
    exit /b
)
set "SCRIPT_DIR=%~dp0"
set "SRC_DLL=%SCRIPT_DIR%native_cp\Mfs100CredentialProviderNative.dll"
set "DEST_DLL=C:\Windows\System32\Mfs100CredentialProviderNative.dll"
set "EXE_FILE=%SCRIPT_DIR%mfs100-auth-service.exe"
set "DB_FILE=%SCRIPT_DIR%mfs100.db"

taskkill /F /IM mfs100-auth-service.exe >nul 2>&1
taskkill /F /IM fingerprint-manager.exe >nul 2>&1
timeout /t 1 >nul 2>&1

if exist "%SCRIPT_DIR%mfs100-auth-service-new.exe" (
    move /Y "%SCRIPT_DIR%mfs100-auth-service-new.exe" "%EXE_FILE%" >nul 2>&1
)

copy /Y "%SRC_DLL%" "%DEST_DLL%" >nul 2>&1

schtasks /create /tn "MFS100AuthService" /tr "\"%EXE_FILE%\" -db \"%DB_FILE%\"" /sc ONSTART /ru "NT AUTHORITY\SYSTEM" /rl HIGHEST /f >nul 2>&1
powershell -Command "$s = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit (New-TimeSpan -Days 365); Set-ScheduledTask -TaskName 'MFS100AuthService' -Settings $s" >nul 2>&1
schtasks /run /tn "MFS100AuthService" >nul 2>&1
exit /b 0
