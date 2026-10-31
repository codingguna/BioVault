@echo off
title MFS100 Auth Service Status
echo ========================================================
echo        MFS100 Background Auth Service Status
echo ========================================================
tasklist /FI "IMAGENAME eq mfs100-auth-service.exe" 2>nul | findstr /i "mfs100-auth-service.exe" >nul
if %ERRORLEVEL% EQU 0 (
    echo [STATUS] RUNNING in background - Silent GUI mode
) else (
    echo [STATUS] NOT RUNNING
)

echo.
echo ========================================================
echo        Recent Service Log (service.log)
echo ========================================================
if exist "C:\ProgramData\Mfs100\service.log" (
    powershell -NoProfile -Command "Get-Content -Path 'C:\ProgramData\Mfs100\service.log' -Tail 10"
) else (
    echo No service.log found yet.
)

echo.
echo ========================================================
echo        Recent Lock Screen Log (cp_log.txt)
echo ========================================================
if exist "C:\ProgramData\Mfs100\cp_log.txt" (
    powershell -NoProfile -Command "Get-Content -Path 'C:\ProgramData\Mfs100\cp_log.txt' -Tail 10"
) else (
    echo No cp_log.txt found yet.
)

echo.
pause
