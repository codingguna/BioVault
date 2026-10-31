@echo off
title Stop MFS100 Auth Service
echo Stopping MFS100 Background Auth Service...
taskkill /F /IM mfs100-auth-service.exe >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    echo [OK] MFS100 Background Auth Service stopped successfully.
) else (
    echo [INFO] MFS100 Background Auth Service was not running.
)
timeout /t 2 >nul
