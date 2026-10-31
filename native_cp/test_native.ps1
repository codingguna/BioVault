$src = @"
using System;
using System.Runtime.InteropServices;

public class NativeTest {
    [DllImport(@"d:\software\fp\native_cp\Mfs100CredentialProviderNative.dll", EntryPoint = "DllGetClassObject")]
    public static extern int DllGetClassObject(ref Guid clsid, ref Guid riid, out IntPtr ppv);
}
"@
Add-Type -TypeDefinition $src
$clsid = New-Object Guid "7C6A8723-53BC-4545-98D1-9B5F9542D4C1"
$riid = New-Object Guid "00000001-0000-0000-C000-000000000046"
$ptr = [IntPtr]::Zero
$hr = [NativeTest]::DllGetClassObject([ref]$clsid, [ref]$riid, [ref]$ptr)
Write-Host "DllGetClassObject HR: 0x$($hr.ToString('X8')), FactoryPtr: $ptr"
