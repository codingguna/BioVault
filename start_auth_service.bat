@echo off
title MFS100 Windows Unlock Service
echo ========================================================
echo   MFS100 Windows Background Authentication Service
echo ========================================================
echo Running background biometric authentication daemon...
echo Keep this window running in background (or install as a Windows service).
cd /d "%~dp0"
mfs100-auth-service.exe
pause
