#pragma once
#include "Mfs100Common.h"
#include "Mfs100Credential.h"

class CMfs100Provider : public ICredentialProvider
{
public:
    CMfs100Provider();
    virtual ~CMfs100Provider();

    // IUnknown
    IFACEMETHODIMP QueryInterface(REFIID riid, void **ppv);
    IFACEMETHODIMP_(ULONG) AddRef();
    IFACEMETHODIMP_(ULONG) Release();

    // ICredentialProvider
    IFACEMETHODIMP SetUsageScenario(CREDENTIAL_PROVIDER_USAGE_SCENARIO cpus, DWORD dwFlags);
    IFACEMETHODIMP SetSerialization(const CREDENTIAL_PROVIDER_CREDENTIAL_SERIALIZATION *pcpcs);
    IFACEMETHODIMP Advise(ICredentialProviderEvents *pcpe, UINT_PTR upAdviseContext);
    IFACEMETHODIMP UnAdvise();
    IFACEMETHODIMP GetFieldDescriptorCount(DWORD *pdwCount);
    IFACEMETHODIMP GetFieldDescriptorAt(DWORD dwIndex, CREDENTIAL_PROVIDER_FIELD_DESCRIPTOR **ppcpfd);
    IFACEMETHODIMP GetCredentialCount(DWORD *pdwCount, DWORD *pdwDefault, BOOL *pbAutoLogonWithDefault);
    IFACEMETHODIMP GetCredentialAt(DWORD dwIndex, ICredentialProviderCredential **ppcpc);

private:
    LONG m_cRef;
    CREDENTIAL_PROVIDER_USAGE_SCENARIO m_cpus;
    DWORD m_dwFlags;
    ICredentialProviderEvents *m_pEvents;
    UINT_PTR m_upAdviseContext;

    CMfs100Credential *m_pCredential;
    WCHAR m_szUserSid[128];
    LUID m_logonId;
};
