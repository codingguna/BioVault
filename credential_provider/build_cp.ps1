# Compile 64-bit Windows Credential Provider DLL
$ErrorActionPreference = "Stop"

$csc = "C:\Windows\Microsoft.NET\Framework64\v4.0.30319\csc.exe"
$outDll = "$PSScriptRoot\Mfs100CredentialProvider.dll"
$src = "$PSScriptRoot\Mfs100CredentialProvider.cs"

Write-Host "Compiling 64-bit Windows Credential Provider..." -ForegroundColor Cyan

& $csc /target:library /platform:x64 /optimize+ /out:$outDll $src

if (Test-Path $outDll) {
    Write-Host "Compiled successfully: $outDll" -ForegroundColor Green
    Get-Item $outDll | Format-Table Name, Length, LastWriteTime
} else {
    Write-Error "Compilation failed."
}
