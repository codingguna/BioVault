#include "ClassFactory.h"
#include "Mfs100Provider.h"

CClassFactory::CClassFactory() : m_cRef(1)
{
    InterlockedIncrement(&g_cRefDll);
}

CClassFactory::~CClassFactory()
{
    InterlockedDecrement(&g_cRefDll);
}

HRESULT CClassFactory::QueryInterface(REFIID riid, void **ppv)
{
    static const QITAB qit[] = {
        QITABENT(CClassFactory, IClassFactory),
        { 0 },
    };
    return QISearch(this, qit, riid, ppv);
}

ULONG CClassFactory::AddRef()
{
    return InterlockedIncrement(&m_cRef);
}

ULONG CClassFactory::Release()
{
    LONG c = InterlockedDecrement(&m_cRef);
    if (c == 0) {
        delete this;
    }
    return c;
}

HRESULT CClassFactory::CreateInstance(IUnknown *pUnkOuter, REFIID riid, void **ppvObject)
{
    LogCP("ClassFactory::CreateInstance called");
    if (!ppvObject) return E_POINTER;
    *ppvObject = NULL;

    if (pUnkOuter != NULL) {
        return CLASS_E_NOAGGREGATION;
    }

    CMfs100Provider* pProvider = new CMfs100Provider();
    if (!pProvider) {
        return E_OUTOFMEMORY;
    }

    HRESULT hr = pProvider->QueryInterface(riid, ppvObject);
    pProvider->Release();
    LogCP("ClassFactory::CreateInstance returned 0x%08X", hr);
    return hr;
}

HRESULT CClassFactory::LockServer(BOOL fLock)
{
    if (fLock) {
        InterlockedIncrement(&g_cRefDll);
    } else {
        InterlockedDecrement(&g_cRefDll);
    }
    return S_OK;
}
