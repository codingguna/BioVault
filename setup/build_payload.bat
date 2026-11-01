@echo off
setlocal enabledelayedexpansion

echo ========================================================
echo       Building BioVault Biometric Setup Components
echo ========================================================

cd /d "%~dp0.."
set "ROOT_DIR=%CD%"
set "SETUP_BIN=%ROOT_DIR%\setup\bin"

if not exist "%SETUP_BIN%" mkdir "%SETUP_BIN%"

:: Copy latest icon to manager folder
copy /y "%ROOT_DIR%\native_cp\mLogon.ico" "%ROOT_DIR%\cmd\mantra-manager\app.ico" >nul

:: 1. Compile resource file with icon + UAC admin manifest for Manager
echo [+] Compiling UAC Admin Manifest and Icon resource...
cd "%ROOT_DIR%\cmd\mantra-manager"
"%ROOT_DIR%\tools\w64devkit\bin\windres.exe" -F pe-i386 -i app.rc -o rsrc_windows_386.syso
if %errorlevel% neq 0 (
    echo [-] Failed to compile manifest resource.
    exit /b 1
)

:: 2. Build 32-bit Native Desktop Manager (No HTTP, No console window)
echo [+] Compiling BioVaultManager.exe (32-bit Windows GUI)...
cd "%ROOT_DIR%"
set GOOS=windows
set GOARCH=386
set CGO_ENABLED=0
go build -ldflags="-H windowsgui -s -w" -o "%SETUP_BIN%\BioVaultManager.exe" ./cmd/mantra-manager
if %errorlevel% neq 0 (
    echo [-] Failed to build BioVaultManager.exe.
    exit /b 1
)
:: Also keep MantraFingerprintManager.exe copy for backward-compatibility
copy /y "%SETUP_BIN%\BioVaultManager.exe" "%SETUP_BIN%\MantraFingerprintManager.exe" >nul

:: 3. Build 32-bit Background Unlock Service
echo [+] Compiling mfs100-auth-service.exe (32-bit Service)...
go build -ldflags="-s -w" -o "%SETUP_BIN%\mfs100-auth-service.exe" ./cmd/mfs100-auth-service
if %errorlevel% neq 0 (
    echo [-] Failed to build mfs100-auth-service.exe.
    exit /b 1
)

:: 4. Build 64-bit Native Credential Provider DLL
echo [+] Building 64-bit Mfs100CredentialProviderNative.dll...
cd "%ROOT_DIR%\native_cp"
call build_native.bat
if %errorlevel% neq 0 (
    echo [-] Failed to compile Credential Provider DLL.
    exit /b 1
)
copy /y "%ROOT_DIR%\native_cp\Mfs100CredentialProviderNative.dll" "%SETUP_BIN%\Mfs100CredentialProviderNative.dll" >nul

:: 5. Copy Official Driver installer
echo [+] Bundling official driver MFS100Driver_9.2.0.0.exe...
copy /y "%ROOT_DIR%\MFS100Driver_9.2.0.0.exe" "%SETUP_BIN%\MFS100Driver_9.2.0.0.exe" >nul
if %errorlevel% neq 0 (
    echo [-] Failed to copy driver installer.
    exit /b 1
)

echo ========================================================
echo [+] All setup payload components built successfully!
echo ========================================================
dir "%SETUP_BIN%"
