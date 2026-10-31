@echo off
cd /d "%~dp0"
wscript.exe "%~dp0start_service_silent.vbs"
echo [OK] MFS100 Background Auth Service started silently.
echo Check logs at: C:\ProgramData\Mfs100\service.log
timeout /t 2 >nul
