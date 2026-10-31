@echo off
setlocal
cd /d "%~dp0"
echo [*] Compiling resource.rc with windres...
..\tools\w64devkit\bin\windres.exe -i resource.rc -o resource.o
if %ERRORLEVEL% NEQ 0 (
    echo [-] Resource compilation failed.
    pause
    exit /b %ERRORLEVEL%
)

echo [*] Compiling Mfs100CredentialProviderNative.dll with g++ 64-bit...
..\tools\w64devkit\bin\g++.exe -O2 -shared -m64 -o Mfs100CredentialProviderNative.dll Dll.cpp Mfs100Provider.cpp Mfs100Credential.cpp ClassFactory.cpp resource.o Mfs100CredentialProvider.def -lole32 -lshlwapi -lsecur32 -lgdi32 -static-libgcc -static-libstdc++

if %ERRORLEVEL% NEQ 0 (
    echo [-] DLL compilation failed.
    pause
    exit /b %ERRORLEVEL%
)

echo [+] Build successful! Mfs100CredentialProviderNative.dll is ready.
