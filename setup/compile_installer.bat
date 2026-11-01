@echo off
setlocal enabledelayedexpansion

echo ========================================================
echo       Compiling Mantra Biometric Setup.exe
echo ========================================================

cd /d "%~dp0"
set "SETUP_DIR=%CD%"

:: Locate Inno Setup Compiler
set "ISCC_PATH=C:\Users\Admin\AppData\Local\Programs\Inno Setup 6\ISCC.exe"
if not exist "%ISCC_PATH%" (
    set "ISCC_PATH=C:\Program Files (x86)\Inno Setup 6\ISCC.exe"
)
if not exist "%ISCC_PATH%" (
    set "ISCC_PATH=C:\Program Files\Inno Setup 6\ISCC.exe"
)

if not exist "%ISCC_PATH%" (
    echo [-] Error: Inno Setup compiler ISCC.exe not found!
    exit /b 1
)

echo [+] Using Inno Setup Compiler: "%ISCC_PATH%"

:: Step 1: Build all payload binaries
echo [*] Step 1: Building payload binaries...
call "%SETUP_DIR%\build_payload.bat"
if %errorlevel% neq 0 (
    echo [-] Error: Payload build failed!
    exit /b 1
)

:: Step 2: Ensure output directory exists
if not exist "%SETUP_DIR%\output" mkdir "%SETUP_DIR%\output"

:: Step 3: Run Inno Setup Compiler
echo [*] Step 2: Compiling Installer with Inno Setup...
"%ISCC_PATH%" "%SETUP_DIR%\MantraUnlockSetup.iss"
if %errorlevel% neq 0 (
    echo [-] Error: Inno Setup compilation failed!
    exit /b 1
)

echo ========================================================
echo [+] SUCCESS! Installer generated at:
echo ========================================================
dir "%SETUP_DIR%\output\*.exe"
