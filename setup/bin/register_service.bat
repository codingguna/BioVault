@echo off
setlocal
cd /d "%~dp0"
set "EXE_PATH=%~dp0mfs100-auth-service.exe"

echo [*] Stopping any existing auth service...
taskkill /f /im mfs100-auth-service.exe >nul 2>&1
schtasks /end /tn "MFS100AuthService" >nul 2>&1

echo [*] Registering MFS100AuthService under NT AUTHORITY\SYSTEM...
schtasks /create /f /tn "MFS100AuthService" /tr "\"%EXE_PATH%\"" /sc ONSTART /ru "NT AUTHORITY\SYSTEM" /rl HIGHEST

echo [*] Starting MFS100AuthService now...
schtasks /run /tn "MFS100AuthService" >nul 2>&1
exit /b 0
