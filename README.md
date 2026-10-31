# Mantra MFS100 Fingerprint Manager & Windows PC Unlock

A complete solution for enrolling fingerprints and **unlocking Windows 11 with the Mantra MFS100 optical fingerprint scanner**.

---

## Architecture Overview

```
               [ Windows 11 Lock Screen (Win + L) ]
                               │
               ┌───────────────┴────────────────┐
               │                                │
       [ Standard PIN / Password ]      [ Mantra Fingerprint Tile ]
           (Always available)           (Mfs100CredentialProvider.dll)
                                                │
                                    Local Named Pipe (IPC)
                                                │
                                                ▼
                             [ MFS100 Background Service ]
                              (mfs100-auth-service.exe)
                                                │
                                                ▼
                                    [ Mantra MFS100 Sensor ]
                                      Scans enrolled finger
                                                │
                              Score >= 1400 (Biometric Verified)
                                                │
                                                ▼
                               Unseals DPAPI-Protected Credential
                                                │
                                                ▼
                                   Windows Unlocks Instantly!
```

---

## Quick Setup Guide for Windows PC Unlock

### Step 1: Enrol Your Fingerprint
1. Run `.\fingerprint-manager.exe`.
2. In the browser dashboard that opens, go to **User Directory** and click **+ New User** (e.g. `Admin`).
3. Switch to the **Enrol Finger** tab:
   - Select your user and finger (e.g., `Right Index`).
   - Click **Scan Finger on Sensor** and place your finger on the MFS100 scanner.
   - Click **Save Template**.

### Step 2: Bind Your Fingerprint to Windows
1. In the Web Dashboard, switch to the **Windows Sign-In** tab.
2. Select your enrolled user profile.
3. Confirm your Windows username (`Admin`) and computer name (`VFPCL-PC`).
4. Enter your Windows sign-in password and click **Bind Fingerprint to Windows Account**.
   > *Security Note:* Your password is encrypted at rest using Windows DPAPI. It is never logged and only decrypted in-memory when your physical fingerprint matches.

### Step 3: Register the Windows Credential Provider (Run as Administrator)
Open PowerShell **as Administrator** and execute:
```powershell
cd d:\software\fp
powershell -ExecutionPolicy Bypass -File .\credential_provider\register_unlock.ps1
```
*Note: Your standard PIN and Password remain 100% active side-by-side. You will never be locked out.*

### Step 4: Start the Background Unlock Service
Launch the background authentication daemon:
```powershell
.\start_auth_service.bat
```

### Step 5: Test Unlocking!
1. Press `Win + L` to lock your PC.
2. You will see the **"Mantra Fingerprint Unlock"** tile on the lock screen.
3. Touch your MFS100 fingerprint sensor with your enrolled finger.
4. **Windows unlocks instantly!**

---

## Emergency Rollback
To unregister the Credential Provider at any time and restore default Windows settings:
```powershell
powershell -ExecutionPolicy Bypass -File .\credential_provider\unregister_unlock.ps1
```

---

## File Layout

```
d:\software\fp\
├── cmd\
│   ├── fingerprint-manager\   # Desktop manager app with Web UI
│   ├── mfs100-auth-service\   # Windows background unlock daemon
│   └── probe\                 # Hardware diagnostic CLI
├── credential_provider\       # 64-bit Windows Credential Provider
│   ├── Mfs100CredentialProvider.cs
│   ├── Mfs100CredentialProvider.dll
│   ├── build_cp.ps1           # Compiler script (C# 64-bit)
│   ├── register_unlock.ps1    # Registration script (Run as Admin)
│   └── unregister_unlock.ps1  # Rollback script
├── internal\
│   ├── ipc\                   # Named Pipe IPC server (\\.\pipe\MFS100AuthPipe)
│   ├── matching\              # 1:1 and 1:N ISO template matcher
│   ├── scanner\               # Mantra MFS100 Win32 DLL adapter
│   ├── security\              # Windows DPAPI encryption
│   └── storage\               # SQLite persistent vault
├── build.ps1                  # Master automated build script
└── start_auth_service.bat     # Service launcher
```
