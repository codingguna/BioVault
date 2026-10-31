#include <windows.h>
#include <shlwapi.h>
#include <stdio.h>
#include "guid.h"
#include "Mfs100Common.h"
#include "ClassFactory.h"

HINSTANCE g_hinstDll = NULL;
LONG g_cRefDll = 0;

BOOL WINAPI DllMain(HINSTANCE hinstDLL, DWORD fdwReason, LPVOID lpvReserved)
{
    if (fdwReason == DLL_PROCESS_ATTACH) {
        g_hinstDll = hinstDLL;
        DisableThreadLibraryCalls(hinstDLL);
        LogCP("DllMain: Attached to process PID=%lu", GetCurrentProcessId());
    } else if (fdwReason == DLL_PROCESS_DETACH) {
        LogCP("DllMain: Detached from process PID=%lu", GetCurrentProcessId());
    }
    return TRUE;
}

STDAPI DllGetClassObject(REFCLSID rclsid, REFIID riid, void **ppv)
{
    LogCP("DllGetClassObject called");
    if (!ppv) return E_POINTER;
    *ppv = NULL;

    if (IsEqualCLSID(rclsid, CLSID_Mfs100Provider)) {
        CClassFactory* pFactory = new CClassFactory();
        if (!pFactory) return E_OUTOFMEMORY;
        HRESULT hr = pFactory->QueryInterface(riid, ppv);
        pFactory->Release();
        return hr;
    }

    LogCP("DllGetClassObject: Unknown CLSID requested");
    return CLASS_E_CLASSNOTAVAILABLE;
}

STDAPI DllCanUnloadNow()
{
    return (g_cRefDll == 0) ? S_OK : S_FALSE;
}

static const WCHAR c_szClsidMfs100[] = L"{7C6A8723-53BC-4545-98D1-9B5F9542D4C1}";
static const WCHAR c_szProviderName[] = L"Mantra MFS100 Fingerprint Credential Provider";

STDAPI DllRegisterServer()
{
    LogCP("DllRegisterServer called");
    WCHAR szModulePath[MAX_PATH];
    if (GetModuleFileNameW(g_hinstDll, szModulePath, ARRAYSIZE(szModulePath)) == 0) {
        return HRESULT_FROM_WIN32(GetLastError());
    }

    // 1. Register CLSID in HKCR / HKLM Classes
    WCHAR szClsidKey[256];
    swprintf(szClsidKey, ARRAYSIZE(szClsidKey), L"CLSID\\%s", c_szClsidMfs100);

    HKEY hKey = NULL;
    if (RegCreateKeyExW(HKEY_CLASSES_ROOT, szClsidKey, 0, NULL, 0, KEY_WRITE, NULL, &hKey, NULL) == ERROR_SUCCESS) {
        RegSetValueExW(hKey, NULL, 0, REG_SZ, (const BYTE*)c_szProviderName, (DWORD)((wcslen(c_szProviderName) + 1) * sizeof(WCHAR)));
        
        HKEY hSubKey = NULL;
        if (RegCreateKeyExW(hKey, L"InprocServer32", 0, NULL, 0, KEY_WRITE, NULL, &hSubKey, NULL) == ERROR_SUCCESS) {
            RegSetValueExW(hSubKey, NULL, 0, REG_SZ, (const BYTE*)szModulePath, (DWORD)((wcslen(szModulePath) + 1) * sizeof(WCHAR)));
            const WCHAR szThreading[] = L"Apartment";
            RegSetValueExW(hSubKey, L"ThreadingModel", 0, REG_SZ, (const BYTE*)szThreading, (DWORD)((wcslen(szThreading) + 1) * sizeof(WCHAR)));
            RegCloseKey(hSubKey);
        }
        RegCloseKey(hKey);
    }

    // 2. Register in HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Authentication\Credential Providers
    WCHAR szCPKey[256];
    swprintf(szCPKey, ARRAYSIZE(szCPKey), L"SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Authentication\\Credential Providers\\%s", c_szClsidMfs100);

    if (RegCreateKeyExW(HKEY_LOCAL_MACHINE, szCPKey, 0, NULL, 0, KEY_WRITE, NULL, &hKey, NULL) == ERROR_SUCCESS) {
        RegSetValueExW(hKey, NULL, 0, REG_SZ, (const BYTE*)c_szProviderName, (DWORD)((wcslen(c_szProviderName) + 1) * sizeof(WCHAR)));
        RegCloseKey(hKey);
    }

    LogCP("DllRegisterServer: Registered successfully at '%ls'", szModulePath);
    return S_OK;
}

STDAPI DllUnregisterServer()
{
    LogCP("DllUnregisterServer called");

    // Remove from Credential Providers
    WCHAR szCPKey[256];
    swprintf(szCPKey, ARRAYSIZE(szCPKey), L"SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Authentication\\Credential Providers\\%s", c_szClsidMfs100);
    RegDeleteKeyW(HKEY_LOCAL_MACHINE, szCPKey);

    // Remove InprocServer32 and CLSID
    WCHAR szSubKey[256];
    swprintf(szSubKey, ARRAYSIZE(szSubKey), L"CLSID\\%s\\InprocServer32", c_szClsidMfs100);
    RegDeleteKeyW(HKEY_CLASSES_ROOT, szSubKey);

    WCHAR szClsidKey[256];
    swprintf(szClsidKey, ARRAYSIZE(szClsidKey), L"CLSID\\%s", c_szClsidMfs100);
    RegDeleteKeyW(HKEY_CLASSES_ROOT, szClsidKey);

    LogCP("DllUnregisterServer: Unregistered successfully.");
    return S_OK;
}
