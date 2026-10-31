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
    , m_hScanThread(NULL)
    , m_bStopThread(FALSE)
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

    wcscpy(m_szStatusText, L"Touch Mantra MFS100 scanner to unlock");
    LogCP("CMfs100Credential constructed for SID: %ls", m_szUserSid);
}

CMfs100Credential::~CMfs100Credential()
{
    m_bStopThread = TRUE;
    if (m_hScanThread) {
        WaitForSingleObject(m_hScanThread, 1000);
        CloseHandle(m_hScanThread);
        m_hScanThread = NULL;
    }
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
    // Automatically trigger scanner listener as soon as lock screen tile is ready
    TriggerScan();
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
    TriggerScan();
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
        *pcpfs = CPFS_DISPLAY_IN_SELECTED_TILE;
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
        return SHStrDupW(L"Mantra Fingerprint", ppsz);

    case MFS_FIELD_STATUS:
        return SHStrDupW(m_szStatusText, ppsz);

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
    if (dwFieldID == MFS_FIELD_SUBMIT) {
        TriggerScan();
    }
    return S_OK;
}

HRESULT CMfs100Credential::GetUserSid(LPWSTR *sid)
{
    if (!sid) return E_POINTER;
    LogCP("Credential::GetUserSid called -> returning %ls", m_szUserSid);
    return SHStrDupW(m_szUserSid, sid);
}

void CMfs100Credential::TriggerScan()
{
    if (m_hScanThread != NULL) {
        DWORD exitCode = 0;
        if (GetExitCodeThread(m_hScanThread, &exitCode) && exitCode == STILL_ACTIVE) {
            LogCP("ScanThread already active, skipping re-trigger.");
            return;
        }
        CloseHandle(m_hScanThread);
        m_hScanThread = NULL;
    }

    m_bStopThread = FALSE;
    m_hScanThread = CreateThread(NULL, 0, ScanThreadWorker, this, 0, NULL);
    LogCP("Created ScanThread (Handle: %p)", m_hScanThread);
}

DWORD WINAPI CMfs100Credential::ScanThreadWorker(LPVOID lpParam)
{
    CMfs100Credential* pThis = (CMfs100Credential*)lpParam;
    LogCP("ScanThread started. Querying auth pipe...");

    PipeAuthResult authRes;
    bool ok = QueryAuthPipe(&authRes, 15000);

    if (pThis->m_bStopThread) {
        LogCP("ScanThread stopped by request.");
        return 0;
    }

    if (ok && authRes.success) {
        LogCP("ScanThread: Biometric authentication SUCCESS for %ls\\%ls!", authRes.domain, authRes.user);
        pThis->m_bAuthenticated = TRUE;
        wcsncpy(pThis->m_szUser, authRes.user, 127);
        wcsncpy(pThis->m_szDomain, authRes.domain, 127);
        wcsncpy(pThis->m_szPassword, authRes.password, 255);

        wcscpy(pThis->m_szStatusText, L"Fingerprint Verified! Unlocking...");
        if (pThis->m_pCredEvents) {
            pThis->m_pCredEvents->SetFieldString(pThis, MFS_FIELD_STATUS, pThis->m_szStatusText);
        }
        if (pThis->m_pProviderEvents) {
            LogCP("Notifying LogonUI via ICredentialProviderEvents::CredentialsChanged...");
            pThis->m_pProviderEvents->CredentialsChanged(pThis->m_upAdviseContext);
        }
    } else {
        LogCP("ScanThread: Auth failed or timed out: %s", authRes.message);
        if (strlen(authRes.message) > 0) {
            MultiByteToWideChar(CP_UTF8, 0, authRes.message, -1, pThis->m_szStatusText, 255);
        } else {
            wcscpy(pThis->m_szStatusText, L"Scan timed out. Touch sensor again.");
        }
        if (pThis->m_pCredEvents) {
            pThis->m_pCredEvents->SetFieldString(pThis, MFS_FIELD_STATUS, pThis->m_szStatusText);
        }
    }

    return 0;
}

HRESULT CMfs100Credential::GetSerialization(
    CREDENTIAL_PROVIDER_GET_SERIALIZATION_RESPONSE *pcpgsr,
    CREDENTIAL_PROVIDER_CREDENTIAL_SERIALIZATION *pcpcs,
    LPWSTR *ppszOptionalStatusText,
    CREDENTIAL_PROVIDER_STATUS_ICON *pcpsiOptionalStatusIcon)
{
    LogCP("Credential::GetSerialization called");
    if (!pcpgsr || !pcpcs) return E_POINTER;

    if (ppszOptionalStatusText) *ppszOptionalStatusText = NULL;
    if (pcpsiOptionalStatusIcon) *pcpsiOptionalStatusIcon = CPSI_NONE;

    if (!m_bAuthenticated) {
        LogCP("GetSerialization: Not authenticated yet. Returning CPGSR_NO_CREDENTIAL_FINISHED.");
        *pcpgsr = CPGSR_NO_CREDENTIAL_FINISHED;
        return S_OK;
    }

    LogCP("GetSerialization: Building KERB_INTERACTIVE_UNLOCK_LOGON for %ls\\%ls", m_szDomain, m_szUser);

    DWORD cbLogon  = sizeof(KERB_INTERACTIVE_UNLOCK_LOGON);
    DWORD cbDomain = (DWORD)wcslen(m_szDomain) * sizeof(WCHAR);
    DWORD cbUser   = (DWORD)wcslen(m_szUser) * sizeof(WCHAR);
    DWORD cbPass   = (DWORD)wcslen(m_szPassword) * sizeof(WCHAR);
    DWORD cbTotal  = cbLogon + cbDomain + cbUser + cbPass;

    BYTE* pBuffer = (BYTE*)CoTaskMemAlloc(cbTotal);
    if (!pBuffer) {
        LogCP("GetSerialization: CoTaskMemAlloc failed for %lu bytes", cbTotal);
        return E_OUTOFMEMORY;
    }
    ZeroMemory(pBuffer, cbTotal);

    PKERB_INTERACTIVE_UNLOCK_LOGON pLogon = (PKERB_INTERACTIVE_UNLOCK_LOGON)pBuffer;
    if (m_cpus == CPUS_UNLOCK_WORKSTATION) {
        pLogon->Logon.MessageType = KerbWorkstationUnlockLogon;
        pLogon->LogonId = m_logonId;
    } else {
        pLogon->Logon.MessageType = KerbInteractiveLogon;
    }

    BYTE* pCursor = pBuffer + cbLogon;

    // Domain
    pLogon->Logon.LogonDomainName.Length = (USHORT)cbDomain;
    pLogon->Logon.LogonDomainName.MaximumLength = (USHORT)cbDomain;
    pLogon->Logon.LogonDomainName.Buffer = (PWSTR)pCursor;
    memcpy(pCursor, m_szDomain, cbDomain);
    pCursor += cbDomain;

    // User
    pLogon->Logon.UserName.Length = (USHORT)cbUser;
    pLogon->Logon.UserName.MaximumLength = (USHORT)cbUser;
    pLogon->Logon.UserName.Buffer = (PWSTR)pCursor;
    memcpy(pCursor, m_szUser, cbUser);
    pCursor += cbUser;

    // Password
    pLogon->Logon.Password.Length = (USHORT)cbPass;
    pLogon->Logon.Password.MaximumLength = (USHORT)cbPass;
    pLogon->Logon.Password.Buffer = (PWSTR)pCursor;
    memcpy(pCursor, m_szPassword, cbPass);
    pCursor += cbPass;

    // Lookup Kerberos Package ID
    ULONG ulAuthPackage = 0;
    HANDLE hLsa = NULL;
    NTSTATUS status = LsaConnectUntrusted(&hLsa);
    if (status >= 0) {
        LSA_STRING pkgName;
        pkgName.Buffer = (PCHAR)MICROSOFT_KERBEROS_NAME_A;
        pkgName.Length = (USHORT)strlen(MICROSOFT_KERBEROS_NAME_A);
        pkgName.MaximumLength = pkgName.Length + 1;
        LsaLookupAuthenticationPackage(hLsa, &pkgName, &ulAuthPackage);
        LsaDeregisterLogonProcess(hLsa);
        LogCP("Kerberos Auth Package ID: %lu", ulAuthPackage);
    } else {
        LogCP("LsaConnectUntrusted failed: 0x%08X", status);
    }

    extern const GUID CLSID_Mfs100Provider;
    pcpcs->ulAuthenticationPackage = ulAuthPackage;
    pcpcs->clsidCredentialProvider = CLSID_Mfs100Provider;
    pcpcs->cbSerialization = cbTotal;
    pcpcs->rgbSerialization = pBuffer;

    *pcpgsr = CPGSR_RETURN_CREDENTIAL_FINISHED;
    LogCP("GetSerialization: Returned CPGSR_RETURN_CREDENTIAL_FINISHED successfully!");
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
        LogCP("ReportResult: Workstation unlock SUCCESS!");
    } else {
        LogCP("ReportResult: Unlock failed with status 0x%08X", ntsStatus);
    }
    return S_OK;
}
