#include "Mfs100Provider.h"

CMfs100Provider::CMfs100Provider()
    : m_cRef(1)
    , m_cpus(CPUS_INVALID)
    , m_dwFlags(0)
    , m_pEvents(NULL)
    , m_upAdviseContext(0)
    , m_pCredential(NULL)
{
    InterlockedIncrement(&g_cRefDll);
    memset(&m_logonId, 0, sizeof(m_logonId));
    wcscpy(m_szUserSid, L"S-1-5-21-4254395636-2491016078-4039007394-1001");
    LogCP("CMfs100Provider constructed");
}

CMfs100Provider::~CMfs100Provider()
{
    if (m_pCredential) {
        m_pCredential->Release();
        m_pCredential = NULL;
    }
    if (m_pEvents) {
        m_pEvents->Release();
        m_pEvents = NULL;
    }
    InterlockedDecrement(&g_cRefDll);
    LogCP("CMfs100Provider destroyed");
}

HRESULT CMfs100Provider::QueryInterface(REFIID riid, void **ppv)
{
    static const QITAB qit[] = {
        QITABENT(CMfs100Provider, ICredentialProvider),
        { 0 },
    };
    return QISearch(this, qit, riid, ppv);
}

ULONG CMfs100Provider::AddRef()
{
    return InterlockedIncrement(&m_cRef);
}

ULONG CMfs100Provider::Release()
{
    LONG c = InterlockedDecrement(&m_cRef);
    if (c == 0) {
        delete this;
    }
    return c;
}

HRESULT CMfs100Provider::SetUsageScenario(CREDENTIAL_PROVIDER_USAGE_SCENARIO cpus, DWORD dwFlags)
{
    LogCP("Provider::SetUsageScenario called (scenario=%d, flags=0x%08X)", cpus, dwFlags);
    m_cpus = cpus;
    m_dwFlags = dwFlags;

    switch (cpus) {
    case CPUS_UNLOCK_WORKSTATION:
    case CPUS_LOGON:
    case CPUS_CREDUI:
        return S_OK;
    default:
        return E_NOTIMPL;
    }
}

HRESULT CMfs100Provider::SetSerialization(const CREDENTIAL_PROVIDER_CREDENTIAL_SERIALIZATION *pcpcs)
{
    LogCP("Provider::SetSerialization called");
    if (!pcpcs) return S_OK;

    if (pcpcs->rgbSerialization && pcpcs->cbSerialization >= sizeof(KERB_INTERACTIVE_UNLOCK_LOGON)) {
        PKERB_INTERACTIVE_UNLOCK_LOGON pUnlock = (PKERB_INTERACTIVE_UNLOCK_LOGON)pcpcs->rgbSerialization;
        m_logonId = pUnlock->LogonId;
        LogCP("SetSerialization: Preserved LogonId (0x%08X:0x%08X)", m_logonId.HighPart, m_logonId.LowPart);
    }
    return S_OK;
}

HRESULT CMfs100Provider::Advise(ICredentialProviderEvents *pcpe, UINT_PTR upAdviseContext)
{
    LogCP("Provider::Advise called");
    if (m_pEvents) {
        m_pEvents->Release();
    }
    m_pEvents = pcpe;
    m_upAdviseContext = upAdviseContext;
    if (m_pEvents) {
        m_pEvents->AddRef();
    }
    if (m_pCredential) {
        m_pCredential->SetProviderEvents(m_pEvents, m_upAdviseContext);
    }
    return S_OK;
}

HRESULT CMfs100Provider::UnAdvise()
{
    LogCP("Provider::UnAdvise called");
    if (m_pEvents) {
        m_pEvents->Release();
        m_pEvents = NULL;
    }
    m_upAdviseContext = 0;
    if (m_pCredential) {
        m_pCredential->SetProviderEvents(NULL, 0);
    }
    return S_OK;
}

HRESULT CMfs100Provider::GetFieldDescriptorCount(DWORD *pdwCount)
{
    if (!pdwCount) return E_POINTER;
    *pdwCount = MFS_FIELD_COUNT;
    return S_OK;
}

HRESULT CMfs100Provider::GetFieldDescriptorAt(DWORD dwIndex, CREDENTIAL_PROVIDER_FIELD_DESCRIPTOR **ppcpfd)
{
    if (!ppcpfd) return E_POINTER;
    *ppcpfd = NULL;

    if (dwIndex >= MFS_FIELD_COUNT) {
        return E_INVALIDARG;
    }

    CREDENTIAL_PROVIDER_FIELD_DESCRIPTOR* pcpfd = (CREDENTIAL_PROVIDER_FIELD_DESCRIPTOR*)CoTaskMemAlloc(sizeof(CREDENTIAL_PROVIDER_FIELD_DESCRIPTOR));
    if (!pcpfd) return E_OUTOFMEMORY;
    ZeroMemory(pcpfd, sizeof(CREDENTIAL_PROVIDER_FIELD_DESCRIPTOR));

    pcpfd->dwFieldID = dwIndex;

    switch (dwIndex) {
    case MFS_FIELD_LOGO:
        pcpfd->cpft = CPFT_TILE_IMAGE;
        pcpfd->pszLabel = NULL;
        break;

    case MFS_FIELD_TITLE:
        pcpfd->cpft = CPFT_LARGE_TEXT;
        SHStrDupW(L"Mantra Fingerprint", &pcpfd->pszLabel);
        break;

    case MFS_FIELD_STATUS:
        pcpfd->cpft = CPFT_SMALL_TEXT;
        SHStrDupW(L"Press arrow to start capture", &pcpfd->pszLabel);
        break;

    case MFS_FIELD_SUBMIT:
        pcpfd->cpft = CPFT_SUBMIT_BUTTON;
        SHStrDupW(L"Submit", &pcpfd->pszLabel);
        break;
    }

    *ppcpfd = pcpfd;
    return S_OK;
}

HRESULT CMfs100Provider::GetCredentialCount(DWORD *pdwCount, DWORD *pdwDefault, BOOL *pbAutoLogonWithDefault)
{
    if (!pdwCount || !pdwDefault || !pbAutoLogonWithDefault) return E_POINTER;
    *pdwCount = 1;
    *pdwDefault = 0;
    *pbAutoLogonWithDefault = FALSE;
    LogCP("Provider::GetCredentialCount count=1, default=0");
    return S_OK;
}

HRESULT CMfs100Provider::GetCredentialAt(DWORD dwIndex, ICredentialProviderCredential **ppcpc)
{
    if (!ppcpc) return E_POINTER;
    *ppcpc = NULL;

    if (dwIndex != 0) {
        return E_INVALIDARG;
    }

    if (!m_pCredential) {
        m_pCredential = new CMfs100Credential(m_cpus, m_szUserSid, m_logonId);
        if (!m_pCredential) return E_OUTOFMEMORY;
        if (m_pEvents) {
            m_pCredential->SetProviderEvents(m_pEvents, m_upAdviseContext);
        }
    }

    LogCP("Provider::GetCredentialAt index=0 -> success");
    return m_pCredential->QueryInterface(IID_ICredentialProviderCredential, (void**)ppcpc);
}
