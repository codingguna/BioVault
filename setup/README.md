# Mantra MFS100 Biometric Windows Logon - Setup & Distribution Package

This directory contains the production-ready distribution package for **Mantra MFS100 Biometric Windows Logon**.

---

## Output Installer

- **File:** `output\MantraUnlock_Setup_v1.0.0.exe`
- **Size:** ~26.8 MB (Self-contained, LZMA2 compressed)
- **Target OS:** Windows 10 & Windows 11 (64-bit)

---

## Single Setup File Handles Both Install & Uninstall

The single executable **`MantraUnlock_Setup_v1.0.0.exe`** intelligently handles both operations:

1. **On a Clean Machine (Not Installed):**
   - Automatically runs the full setup wizard.
   - Silently installs the bundled Mantra MFS100 driver.
   - Deploys the native GUI Manager and Background Service to `C:\Program Files\MantraUnlock\`.
   - Copies and registers the Credential Provider in `C:\Windows\System32\`.
   - Hardens `C:\ProgramData\MantraUnlock\` with strict NTFS ACLs (SYSTEM & Admin only).
   - Configures the background service to start automatically on Windows boot under `NT AUTHORITY\SYSTEM`.
   - Creates Start Menu & Desktop shortcuts.

2. **On an Already Installed Machine:**
   - Detects the existing installation automatically.
   - Prompts the user:
     - **[YES]:** Completely **UNINSTALLS** Mantra Biometric Logon (stops service, deletes scheduled task, unregisters Credential Provider from lock screen, removes files, and asks whether to keep or delete biometric templates).
     - **[NO]:** **REINSTALLS / UPDATES** the application.
     - **[CANCEL]:** Aborts without making changes.

---

## Alternative Ways to Uninstall

You also have three other standard ways to uninstall at any time:
1. **Windows Settings:** Open *Settings -> Apps -> Installed Apps (or Add or Remove Programs)*, find **Mantra MFS100 Biometric Logon**, and click **Uninstall**.
2. **Start Menu Shortcut:** Click *Start -> Mantra Fingerprint Security -> Uninstall Mantra MFS100 Biometric Logon*.
3. **Standalone Script:** Double-click **`Uninstall_MantraUnlock.bat`** (prompts for UAC elevation and cleans up all components).

---

## Directory Contents

```
setup/
├── bin/                                  # Built payload binaries
│   ├── MantraFingerprintManager.exe      # 32-bit Native Elevated Desktop GUI App (No HTTP/ports)
│   ├── mfs100-auth-service.exe           # 32-bit Background Unlock Service (Runs under SYSTEM)
│   ├── Mfs100CredentialProviderNative.dll# 64-bit Native Credential Provider DLL (System32)
│   └── MFS100Driver_9.2.0.0.exe          # Official Mantra Device Driver Setup
├── output/
│   └── MantraUnlock_Setup_v1.0.0.exe     # Unified Installer & Uninstaller
├── MantraUnlockSetup.iss                 # Inno Setup Script
├── build_payload.bat                     # Compiles all 4 payload components
├── compile_installer.bat                 # Recompiles Setup.exe
├── Uninstall_MantraUnlock.bat            # Direct administrative uninstaller script
└── README.md                             # Documentation
```
