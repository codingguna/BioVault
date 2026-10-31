#pragma once
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <credentialprovider.h>
#include <ntsecapi.h>
#include <shlwapi.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define IDI_FINGERPRINT 101

enum MFS_FIELD_ID {
    MFS_FIELD_LOGO   = 0,
    MFS_FIELD_TITLE  = 1,
    MFS_FIELD_STATUS = 2,
    MFS_FIELD_SUBMIT = 3,
    MFS_FIELD_COUNT  = 4
};

extern HINSTANCE g_hinstDll;
extern LONG g_cRefDll;

inline void LogCP(const char* fmt, ...) {
    char buffer[1024];
    va_list args;
    va_start(args, fmt);
    vsnprintf(buffer, sizeof(buffer), fmt, args);
    va_end(args);

    SYSTEMTIME st;
    GetLocalTime(&st);

    // Ensure ProgramData\Mfs100 exists
    CreateDirectoryW(L"C:\\ProgramData\\Mfs100", NULL);

    FILE* f = fopen("C:\\ProgramData\\Mfs100\\native_cp.log", "a");
    if (f) {
        fprintf(f, "[%04d-%02d-%02d %02d:%02d:%02d.%03d] %s\n",
            st.wYear, st.wMonth, st.wDay, st.wHour, st.wMinute, st.wSecond, st.wMilliseconds, buffer);
        fclose(f);
    }
}

struct PipeAuthResult {
    bool success;
    WCHAR user[128];
    WCHAR domain[128];
    WCHAR password[256];
    char message[256];
};

inline bool QueryAuthPipe(PipeAuthResult* pResult, DWORD timeoutMs) {
    if (!pResult) return false;
    memset(pResult, 0, sizeof(PipeAuthResult));
    pResult->success = false;

    LogCP("QueryAuthPipe: Connecting to \\\\.\\pipe\\MFS100AuthPipe...");
    WaitNamedPipeW(L"\\\\.\\pipe\\MFS100AuthPipe", 2000);

    HANDLE hPipe = CreateFileW(
        L"\\\\.\\pipe\\MFS100AuthPipe",
        GENERIC_READ | GENERIC_WRITE,
        0,
        NULL,
        OPEN_EXISTING,
        0,
        NULL
    );

    if (hPipe == INVALID_HANDLE_VALUE) {
        DWORD dwErr = GetLastError();
        LogCP("QueryAuthPipe: Failed to open pipe, error %lu", dwErr);
        snprintf(pResult->message, sizeof(pResult->message), "Auth service not running (err %lu)", dwErr);
        return false;
    }

    LogCP("QueryAuthPipe: Connected. Sending TRIGGER_UNLOCK...");
    const char req[] = "{\"command\":\"TRIGGER_UNLOCK\"}";
    DWORD written = 0;
    if (!WriteFile(hPipe, req, (DWORD)strlen(req), &written, NULL)) {
        DWORD dwErr = GetLastError();
        LogCP("QueryAuthPipe: WriteFile failed: %lu", dwErr);
        CloseHandle(hPipe);
        return false;
    }

    LogCP("QueryAuthPipe: Waiting for fingerprint response...");
    char buf[4096] = {0};
    DWORD bytesRead = 0;
    if (!ReadFile(hPipe, buf, sizeof(buf) - 1, &bytesRead, NULL) || bytesRead == 0) {
        DWORD dwErr = GetLastError();
        LogCP("QueryAuthPipe: ReadFile failed: %lu", dwErr);
        CloseHandle(hPipe);
        return false;
    }
    CloseHandle(hPipe);
    buf[bytesRead] = 0;
    LogCP("QueryAuthPipe: Got response (len=%lu): %s", bytesRead, buf);

    if (strstr(buf, "\"SUCCESS\"") != NULL) {
        pResult->success = true;
        
        // Extract user
        const char* u = strstr(buf, "\"user\":\"");
        if (u) {
            u += 8;
            const char* ue = strchr(u, '\"');
            if (ue) {
                int len = MultiByteToWideChar(CP_UTF8, 0, u, (int)(ue - u), pResult->user, 127);
                pResult->user[len] = 0;
            }
        }
        
        // Extract domain
        const char* d = strstr(buf, "\"domain\":\"");
        if (d) {
            d += 10;
            const char* de = strchr(d, '\"');
            if (de) {
                int len = MultiByteToWideChar(CP_UTF8, 0, d, (int)(de - d), pResult->domain, 127);
                pResult->domain[len] = 0;
            }
        }
        
        // Extract password
        const char* p = strstr(buf, "\"password\":\"");
        if (p) {
            p += 12;
            const char* pe = strchr(p, '\"');
            if (pe) {
                int len = MultiByteToWideChar(CP_UTF8, 0, p, (int)(pe - p), pResult->password, 255);
                pResult->password[len] = 0;
            }
        }

        LogCP("QueryAuthPipe: Successfully retrieved credentials for user '%ls' domain '%ls'", 
              pResult->user, pResult->domain);
        return true;
    } else {
        const char* m = strstr(buf, "\"message\":\"");
        if (m) {
            m += 11;
            const char* me = strchr(m, '\"');
            if (me) {
                size_t len = me - m;
                if (len > 255) len = 255;
                strncpy(pResult->message, m, len);
                pResult->message[len] = 0;
            }
        } else {
            strncpy(pResult->message, "Fingerprint verification failed", sizeof(pResult->message) - 1);
        }
        LogCP("QueryAuthPipe: Auth failed: %s", pResult->message);
        return false;
    }
}

inline HBITMAP CreateFingerprintBitmap() {
    HICON hIcon = (HICON)LoadImageW(g_hinstDll, MAKEINTRESOURCEW(IDI_FINGERPRINT), IMAGE_ICON, 48, 48, LR_DEFAULTCOLOR);
    if (!hIcon) {
        hIcon = (HICON)LoadImageW(NULL, L"C:\\Program Files\\Mantra\\WBF\\mLogon.ico", IMAGE_ICON, 48, 48, LR_LOADFROMFILE);
    }
    
    int cx = 48, cy = 48;
    HDC hdcScreen = GetDC(NULL);
    HDC hdcMem = CreateCompatibleDC(hdcScreen);

    BITMAPINFO bmi = {0};
    bmi.bmiHeader.biSize = sizeof(BITMAPINFOHEADER);
    bmi.bmiHeader.biWidth = cx;
    bmi.bmiHeader.biHeight = cy;
    bmi.bmiHeader.biPlanes = 1;
    bmi.bmiHeader.biBitCount = 32;
    bmi.bmiHeader.biCompression = BI_RGB;

    void* pBits = NULL;
    HBITMAP hbmp = CreateDIBSection(hdcMem, &bmi, DIB_RGB_COLORS, &pBits, NULL, 0);
    HBITMAP hOldBmp = (HBITMAP)SelectObject(hdcMem, hbmp);

    RECT rc = {0, 0, cx, cy};
    HBRUSH hBrush = CreateSolidBrush(RGB(30, 30, 30));
    FillRect(hdcMem, &rc, hBrush);
    DeleteObject(hBrush);

    if (hIcon) {
        DrawIconEx(hdcMem, 0, 0, hIcon, cx, cy, 0, NULL, DI_NORMAL);
        DestroyIcon(hIcon);
    }

    SelectObject(hdcMem, hOldBmp);
    DeleteDC(hdcMem);
    ReleaseDC(NULL, hdcScreen);

    return hbmp;
}
