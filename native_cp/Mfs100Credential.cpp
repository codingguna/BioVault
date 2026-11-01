#include "Mfs100Credential.h"

CMfs100Credential::CMfs100Credential(CREDENTIAL_PROVIDER_USAGE_SCENARIO cpus, LPCWSTR pszUserSid, LUID logonId)
    : m_cRef(1)
    , m_cpus(cpus)
    , m_pCredEvents(NULL)
    , m_pProviderEvents(NULL)
    , m_upAdviseContext(0)
    , m_logonId(logonId)
    , m_bSelected(FALSE)
    , m_bAuthenticated(FALSE)
{
    InterlockedIncrement(&g_cRefDll);
    memset(m_szUser, 0, sizeof(m_szUser));
    memset(m_szDomain, 0, sizeof(m_szDomain));
    memset(m_szPassword, 0, sizeof(m_szPassword));

    if (pszUserSid && wcslen(pszUserSid) > 0) {
        wcsncpy(m_szUserSid, pszUserSid, 127);
    } else {
        wcscpy(m_szUserSid, L"S-1-5-21-4254395636-2491016078-4039007394-1001");
    }

    wcscpy(m_szStatusText, L"Touch sensor or press [->] to unlock");
    LogCP("CMfs100Credential constructed for SID: %ls", m_szUserSid);
}

CMfs100Credential::~CMfs100Credential()
{
    if (m_pCredEvents) {
        m_pCredEvents->Release();
        m_pCredEvents = NULL;
    }
    if (m_pProviderEvents) {
        m_pProviderEvents->Release();
        m_pProviderEvents = NULL;
    }
    InterlockedDecrement(&g_cRefDll);
    LogCP("CMfs100Credential destroyed");
}

HRESULT CMfs100Credential::QueryInterface(REFIID riid, void **ppv)
{
    static const QITAB qit[] = {
        QITABENT(CMfs100Credential, ICredentialProviderCredential),
        QITABENT(CMfs100Credential, ICredentialProviderCredential2),
        { 0 },
    };
    return QISearch(this, qit, riid, ppv);
}

ULONG CMfs100Credential::AddRef()
{
    return InterlockedIncrement(&m_cRef);
}

ULONG CMfs100Credential::Release()
{
    LONG c = InterlockedDecrement(&m_cRef);
    if (c == 0) {
        delete this;
    }
    return c;
}

void CMfs100Credential::SetProviderEvents(ICredentialProviderEvents *pEvents, UINT_PTR upContext)
{
    if (m_pProviderEvents) {
        m_pProviderEvents->Release();
    }
    m_pProviderEvents = pEvents;
    m_upAdviseContext = upContext;
    if (m_pProviderEvents) {
        m_pProviderEvents->AddRef();
    }
}

HRESULT CMfs100Credential::Advise(ICredentialProviderCredentialEvents *pcpce)
{
    LogCP("Credential::Advise called");
    if (m_pCredEvents) {
        m_pCredEvents->Release();
    }
    m_pCredEvents = pcpce;
    if (m_pCredEvents) {
        m_pCredEvents->AddRef();
    }
    // Scanner stays completely OFF on Advise. No automatic scan.
    return S_OK;
}

HRESULT CMfs100Credential::UnAdvise()
{
    LogCP("Credential::UnAdvise called");
    if (m_pCredEvents) {
        m_pCredEvents->Release();
        m_pCredEvents = NULL;
    }
    return S_OK;
}

HRESULT CMfs100Credential::SetSelected(BOOL *pbAutoLogon)
{
    LogCP("Credential::SetSelected called");
    if (pbAutoLogon) *pbAutoLogon = FALSE;
    m_bSelected = TRUE;
    // Scanner stays completely OFF on SetSelected. User clicks arrow [->] or presses Enter to scan.
    return S_OK;
}

HRESULT CMfs100Credential::SetDeselected()
{
    LogCP("Credential::SetDeselected called");
    m_bSelected = FALSE;
    return S_OK;
}

HRESULT CMfs100Credential::GetFieldState(DWORD dwFieldID, CREDENTIAL_PROVIDER_FIELD_STATE *pcpfs, CREDENTIAL_PROVIDER_FIELD_INTERACTIVE_STATE *pcpfis)
{
    if (!pcpfs || !pcpfis) return E_POINTER;

    switch (dwFieldID) {
    case MFS_FIELD_LOGO:
    case MFS_FIELD_TITLE:
    case MFS_FIELD_STATUS:
        *pcpfs = CPFS_DISPLAY_IN_BOTH;
        *pcpfis = CPFIS_NONE;
        return S_OK;

    case MFS_FIELD_SUBMIT:
        *pcpfs = CPFS_DISPLAY_IN_BOTH;
        *pcpfis = CPFIS_FOCUSED;
        return S_OK;

    default:
        *pcpfs = CPFS_HIDDEN;
        *pcpfis = CPFIS_NONE;
        return E_INVALIDARG;
    }
}

HRESULT CMfs100Credential::GetStringValue(DWORD dwFieldID, LPWSTR *ppsz)
{
    if (!ppsz) return E_POINTER;
    *ppsz = NULL;

    switch (dwFieldID) {
    case MFS_FIELD_TITLE:
        return SHStrDupW(L"BioVault™ Logon", ppsz);

    case MFS_FIELD_STATUS:
        return SHStrDupW(m_szStatusText, ppsz);

    case MFS_FIELD_SUBMIT:
        return SHStrDupW(L"Submit", ppsz);

    default:
        return E_INVALIDARG;
    }
}

HRESULT CMfs100Credential::GetBitmapValue(DWORD dwFieldID, HBITMAP *phbmp)
{
    if (!phbmp) return E_POINTER;
    *phbmp = NULL;

    if (dwFieldID == MFS_FIELD_LOGO) {
        *phbmp = CreateFingerprintBitmap();
        return *phbmp ? S_OK : E_FAIL;
    }
    return E_INVALIDARG;
}

HRESULT CMfs100Credential::GetCheckboxValue(DWORD dwFieldID, BOOL *pbChecked, LPWSTR *ppszLabel)
{
    return E_NOTIMPL;
}

HRESULT CMfs100Credential::GetSubmitButtonValue(DWORD dwFieldID, DWORD *pdwAdjacentTo)
{
    if (!pdwAdjacentTo) return E_POINTER;
    if (dwFieldID == MFS_FIELD_SUBMIT) {
        *pdwAdjacentTo = MFS_FIELD_STATUS;
        return S_OK;
    }
    return E_INVALIDARG;
}

HRESULT CMfs100Credential::GetComboBoxValueCount(DWORD dwFieldID, DWORD *pcItems, DWORD *pdwSelectedItem)
{
    return E_NOTIMPL;
}

HRESULT CMfs100Credential::GetComboBoxValueAt(DWORD dwFieldID, DWORD dwItem, LPWSTR *ppszItem)
{
    return E_NOTIMPL;
}

HRESULT CMfs100Credential::SetStringValue(DWORD dwFieldID, LPCWSTR psz)
{
    return S_OK;
}

HRESULT CMfs100Credential::SetCheckboxValue(DWORD dwFieldID, BOOL bChecked)
{
    return E_NOTIMPL;
}

HRESULT CMfs100Credential::SetComboBoxSelectedValue(DWORD dwFieldID, DWORD dwSelectedItem)
{
    return E_NOTIMPL;
}

HRESULT CMfs100Credential::CommandLinkClicked(DWORD dwFieldID)
{
    return S_OK;
}

HRESULT CMfs100Credential::GetUserSid(LPWSTR *sid)
{
    if (!sid) return E_POINTER;
    LogCP("Credential::GetUserSid called -> returning %ls", m_szUserSid);
    return SHStrDupW(m_szUserSid, sid);
}

HRESULT CMfs100Credential::GetSerialization(
    CREDENTIAL_PROVIDER_GET_SERIALIZATION_RESPONSE *pcpgsr,
    CREDENTIAL_PROVIDER_CREDENTIAL_SERIALIZATION *pcpcs,
    LPWSTR *ppszOptionalStatusText,
    CREDENTIAL_PROVIDER_STATUS_ICON *pcpsiOptionalStatusIcon)
{
    LogCP("Credential::GetSerialization called: User initiated scan via Submit Arrow [->] or Enter.");
    if (!pcpgsr || !pcpcs) return E_POINTER;

    if (ppszOptionalStatusText) *ppszOptionalStatusText = NULL;
    if (pcpsiOptionalStatusIcon) *pcpsiOptionalStatusIcon = CPSI_NONE;

    // Update status text on UI
    wcscpy(m_szStatusText, L"Place finger on Mantra scanner...");
    if (m_pCredEvents) {
        m_pCredEvents->SetFieldString(this, MFS_FIELD_STATUS, m_szStatusText);
    }

    // Call background service to trigger capture and matching
    PipeAuthResult authRes;
    bool ok = QueryAuthPipe(&authRes, 15000);

    if (!ok || !authRes.success) {
        LogCP("GetSerialization: Biometric verification failed or timed out: %s", authRes.message);
        *pcpgsr = CPGSR_NO_CREDENTIAL_FINISHED;
        m_bAuthenticated = FALSE;

        if (strstr(authRes.message, "timed out") != NULL || strstr(authRes.message, "timeout") != NULL) {
            wcscpy(m_szStatusText, L"Scan timed out. Press arrow to retry.");
        } else if (strlen(authRes.message) > 0) {
            MultiByteToWideChar(CP_UTF8, 0, authRes.message, -1, m_szStatusText, 255);
        } else {
            wcscpy(m_szStatusText, L"Fingerprint not recognized. Press arrow to retry.");
        }

        if (ppszOptionalStatusText) {
            SHStrDupW(m_szStatusText, ppszOptionalStatusText);
        }
        if (pcpsiOptionalStatusIcon) {
            *pcpsiOptionalStatusIcon = CPSI_ERROR;
        }
        if (m_pCredEvents) {
            m_pCredEvents->SetFieldString(this, MFS_FIELD_STATUS, m_szStatusText);
        }
        return S_OK;
    }

    // Biometric match SUCCESSFUL!
    LogCP("GetSerialization: Biometric scan SUCCESS for %ls\\%ls! Proceeding to desktop...", authRes.domain, authRes.user);
    m_bAuthenticated = TRUE;
    wcsncpy(m_szUser, authRes.user, 127);
    wcsncpy(m_szDomain, authRes.domain, 127);
    wcsncpy(m_szPassword, authRes.password, 255);

    wcscpy(m_szStatusText, L"Fingerprint Verified! Unlocking...");
    if (m_pCredEvents) {
        m_pCredEvents->SetFieldString(this, MFS_FIELD_STATUS, m_szStatusText);
    }

    // Resolve domain for local SAM accounts
    WCHAR szTargetDomain[128] = {0};
    if (wcslen(m_szDomain) == 0 || wcscmp(m_szDomain, L".") == 0) {
        DWORD dwSize = ARRAYSIZE(szTargetDomain);
        if (!GetComputerNameW(szTargetDomain, &dwSize)) {
            wcscpy(szTargetDomain, L".");
        }
    } else {
        wcsncpy(szTargetDomain, m_szDomain, ARRAYSIZE(szTargetDomain) - 1);
    }

    LogCP("GetSerialization: Packing KERB_INTERACTIVE_UNLOCK_LOGON for user '%ls', domain '%ls'", m_szUser, szTargetDomain);

    UNICODE_STRING usDomain, usUser, usPass;
    usDomain.Length = (USHORT)(wcslen(szTargetDomain) * sizeof(WCHAR));
    usDomain.MaximumLength = usDomain.Length;
    usDomain.Buffer = szTargetDomain;

    usUser.Length = (USHORT)(wcslen(m_szUser) * sizeof(WCHAR));
    usUser.MaximumLength = usUser.Length;
    usUser.Buffer = m_szUser;

    usPass.Length = (USHORT)(wcslen(m_szPassword) * sizeof(WCHAR));
    usPass.MaximumLength = usPass.Length;
    usPass.Buffer = m_szPassword;

    // Total size = struct size + string bytes
    DWORD cbTotal = sizeof(KERB_INTERACTIVE_UNLOCK_LOGON) + usDomain.Length + usUser.Length + usPass.Length;

    BYTE* pBuffer = (BYTE*)CoTaskMemAlloc(cbTotal);
    if (!pBuffer) {
        LogCP("GetSerialization: CoTaskMemAlloc failed for %lu bytes", cbTotal);
        return E_OUTOFMEMORY;
    }
    ZeroMemory(pBuffer, cbTotal);

    PKERB_INTERACTIVE_UNLOCK_LOGON pUnlock = (PKERB_INTERACTIVE_UNLOCK_LOGON)pBuffer;
    pUnlock->LogonId = m_logonId;

    if (m_cpus == CPUS_UNLOCK_WORKSTATION) {
        pUnlock->Logon.MessageType = KerbWorkstationUnlockLogon;
    } else {
        pUnlock->Logon.MessageType = KerbInteractiveLogon;
    }

    // Extra space directly follows the struct
    BYTE* pbExtra = pBuffer + sizeof(KERB_INTERACTIVE_UNLOCK_LOGON);

    // 1. Domain (relative byte offset from pBuffer)
    if (usDomain.Length > 0) {
        CopyMemory(pbExtra, usDomain.Buffer, usDomain.Length);
        pUnlock->Logon.LogonDomainName.Length = usDomain.Length;
        pUnlock->Logon.LogonDomainName.MaximumLength = usDomain.Length;
        pUnlock->Logon.LogonDomainName.Buffer = (PWSTR)(pbExtra - pBuffer);
        pbExtra += usDomain.Length;
    }

    // 2. User (relative byte offset from pBuffer)
    if (usUser.Length > 0) {
        CopyMemory(pbExtra, usUser.Buffer, usUser.Length);
        pUnlock->Logon.UserName.Length = usUser.Length;
        pUnlock->Logon.UserName.MaximumLength = usUser.Length;
        pUnlock->Logon.UserName.Buffer = (PWSTR)(pbExtra - pBuffer);
        pbExtra += usUser.Length;
    }

    // 3. Password (relative byte offset from pBuffer)
    if (usPass.Length > 0) {
        CopyMemory(pbExtra, usPass.Buffer, usPass.Length);
        pUnlock->Logon.Password.Length = usPass.Length;
        pUnlock->Logon.Password.MaximumLength = usPass.Length;
        pUnlock->Logon.Password.Buffer = (PWSTR)(pbExtra - pBuffer);
        pbExtra += usPass.Length;
    }

    // Lookup Negotiate Auth Package (NEGOSSP_NAME_A)
    ULONG ulAuthPackage = 0;
    HANDLE hLsa = NULL;
    NTSTATUS status = LsaConnectUntrusted(&hLsa);
    if (status >= 0) {
        LSA_STRING pkgName;
        pkgName.Buffer = (PCHAR)NEGOSSP_NAME_A;
        pkgName.Length = (USHORT)strlen(NEGOSSP_NAME_A);
        pkgName.MaximumLength = pkgName.Length + 1;
        LsaLookupAuthenticationPackage(hLsa, &pkgName, &ulAuthPackage);
        LsaDeregisterLogonProcess(hLsa);
        LogCP("Negotiate Auth Package ID: %lu", ulAuthPackage);
    } else {
        LogCP("LsaConnectUntrusted failed: 0x%08X", status);
    }

    extern const GUID CLSID_Mfs100Provider;
    pcpcs->ulAuthenticationPackage = ulAuthPackage;
    pcpcs->clsidCredentialProvider = CLSID_Mfs100Provider;
    pcpcs->cbSerialization = cbTotal;
    pcpcs->rgbSerialization = pBuffer;

    *pcpgsr = CPGSR_RETURN_CREDENTIAL_FINISHED;
    LogCP("GetSerialization: Returned CPGSR_RETURN_CREDENTIAL_FINISHED (total_bytes=%lu, domain_offset=%p, user_offset=%p, pass_offset=%p)!",
          cbTotal, pUnlock->Logon.LogonDomainName.Buffer, pUnlock->Logon.UserName.Buffer, pUnlock->Logon.Password.Buffer);
    return S_OK;
}

HRESULT CMfs100Credential::ReportResult(
    NTSTATUS ntsStatus,
    NTSTATUS ntsSubstatus,
    LPWSTR *ppszOptionalStatusText,
    CREDENTIAL_PROVIDER_STATUS_ICON *pcpsiOptionalStatusIcon)
{
    LogCP("Credential::ReportResult status=0x%08X, substatus=0x%08X", ntsStatus, ntsSubstatus);
    if (ntsStatus == 0) {
        LogCP("ReportResult: Workstation unlock SUCCESS! Entering desktop.");
    } else {
        LogCP("ReportResult: Unlock failed with status 0x%08X", ntsStatus);
    }
    return S_OK;
}
