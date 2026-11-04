<p align="center">
  <img src="./logo.jpg" alt="BioVault Logo" width="220" />
</p>

<h1 align="center">BioVault</h1>

<p align="center">
  <img src="https://img.shields.io/badge/C-00599C?logo=c&logoColor=white" alt="C" />
  <img src="https://img.shields.io/badge/C%2B%2B-00599C?logo=c%2B%2B&logoColor=white" alt="C++" />
  <img src="https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/PowerShell-5391FE?logo=powershell&logoColor=white" alt="PowerShell" />
  <img src="https://img.shields.io/badge/Batch-4B8BBE?logo=windows&logoColor=white" alt="Batch" />
  <img src="https://img.shields.io/badge/CMake-064F8C?logo=cmake&logoColor=white" alt="CMake" />
  <img src="https://img.shields.io/badge/NSIS-1E1E1E?logo=windows&logoColor=white" alt="NSIS" />
  <img src="https://img.shields.io/badge/Vim%20Script-019733?logo=vim&logoColor=white" alt="Vim Script" />
  <img src="https://img.shields.io/badge/Roff-6A5ACD?logo=linux&logoColor=white" alt="Roff" />
</p>

<p align="center">
  <strong>Secure biometric Windows authentication for Mantra MFS100 fingerprints</strong>
</p>

BioVault is a Windows-first biometric authentication platform that lets users unlock a PC with a Mantra MFS100 fingerprint scanner. It combines a secure credential provider, local background service, encrypted credential storage, and a native desktop management interface to provide a fast and privacy-conscious alternative to standard password/PIN sign-in.

## Overview

This project delivers a complete fingerprint-based unlocking flow for Windows 11/10 environments using Mantra hardware. The solution includes:

- A fingerprint enrollment and management dashboard
- A Windows credential provider that appears in the lock screen
- A background authentication service that validates fingerprints locally
- DPAPI-backed secure storage for Windows account secrets
- SQLite-based persistence for enrolled users and template data
- Native support for the Mantra MFS100 sensor and Windows IPC mechanisms
- Installer and uninstaller packaging for distribution

## Key Features

- Biometric unlock for Windows lock screens using Mantra MFS100
- Retains the standard Windows PIN/password sign-in path without replacing it
- User fingerprint enrollment and template management
- Local verification using the MFS100 sensor and optional matching pipeline
- Encrypted storage of sign-in credentials using Windows DPAPI
- Installer-ready deployment for enterprise or personal desktop use
- Native Windows integration through credential provider registration
- Service-based authentication model for background processing

## Architecture

```text
[ Windows 11 Lock Screen ]
          │
          ├── Standard PIN / Password (always available)
          │
          └── Mantra Fingerprint Tile
                    │
                    ▼
       MFS100 Credential Provider (native)
                    │
                    ▼
             Local Named Pipe IPC
                    │
                    ▼
       mfs100-auth-service.exe
                    │
                    ▼
             Mantra MFS100 Sensor
                    │
                    ▼
        Fingerprint template match / verification
                    │
                    ▼
     DPAPI-protected Windows credential unlock
                    │
                    ▼
             Windows session unlocks instantly
```

## Repository Structure

```text
BioVault/
├── cmd/
│   ├── fingerprint-manager/      # Desktop manager with web UI
│   ├── mfs100-auth-service/      # Background unlock daemon
│   └── probe/                    # Hardware diagnostic CLI
├── credential_provider/          # Windows Credential Provider assets and scripts
├── internal/
│   ├── ipc/                      # Named pipe communication
│   ├── matching/                 # ISO template matching logic
│   ├── scanner/                  # Mantra MFS100 driver adapter
│   ├── security/                 # DPAPI and protection layers
│   └── storage/                  # SQLite-backed data storage
├── setup/                       # Installer package and setup documentation
├── tools/
│   └── w64devkit/                # Windows cross-toolchain utilities
├── build.ps1                    # Build orchestration
├── go.mod                       # Go module definition
├── go.sum                       # Go dependency lockfile
├── logo.jpg                     # Project branding/logo
├── MFS100 - Datasheet_1.0.0.pdf
├── MFS100Driver_9.2.0.0.exe
├── MFS100_Go_Windows_Fingerprint_Project_Plan.md
├── README.md                    # Project documentation
├── install_native_provider.bat
├── uninstall_native_provider.bat
├── start_auth_service.bat
├── stop_auth_service.bat
├── update_service.bat
├── test.reg
└── mfs100.db                    # Local database snapshot
```

## Supported Technologies

BioVault is built around a combination of:

- C and C++ for native Windows integration and hardware interaction
- Go for service and application orchestration
- CMake and NSIS for build and package lifecycle support
- PowerShell and batch scripts for deployment and service control
- SQLite for local biometric metadata persistence
- Windows DPAPI for secure credential protection

## Quick Start

### 1) Prepare the environment

Install the required Mantra hardware driver and ensure the system is running on a supported Windows 10/11 64-bit platform.

### 2) Enroll a fingerprint

Run the desktop manager:

```powershell
.ingerprint-manager.exe
```

Then:

1. Open the User Directory
2. Create a new user profile
3. Navigate to Enrol Finger
4. Scan the selected finger on the MFS100 sensor
5. Save the template

### 3) Bind to the Windows account

Use the Windows Sign-In panel in the dashboard to associate the enrolled user with the target Windows account and provide the local account password. The password is protected at rest via Windows DPAPI.

### 4) Register the credential provider

Run PowerShell as Administrator:

```powershell
cd <project-directory>
powershell -ExecutionPolicy Bypass -File .\credential_provider\register_unlock.ps1
```

### 5) Start the service

```powershell
.\start_auth_service.bat
```

### 6) Unlock your PC

Press `Win + L`, then place the enrolled finger on the sensor. If verification succeeds, the system unlocks the workstation.

## Security Model

BioVault is designed around local validation and secure handling:

- Fingerprint matching occurs on the local machine
- The Windows password is not stored in plaintext
- Sensitive credential material is protected using Windows DPAPI
- The credential provider remains a parallel sign-in option rather than a hard replacement
- Standard PIN/password authentication remains available as a fallback

## Build and Packaging

The repository includes scripts for building and distributing the solution:

- `build.ps1` — overall build orchestration
- `install_native_provider.bat` — credential provider installation helper
- `uninstall_native_provider.bat` — credential provider rollback
- `setup/` — packaged installer assets and deployment instructions

## Documentation

Additional project references are included in the repository:

- `MFS100_Go_Windows_Fingerprint_Project_Plan.md` — project planning and design notes
- `setup/README.md` — distribution package and setup details
- `tools/w64devkit/README.md` — Windows cross-compilation toolchain documentation

## Notes

This repository contains both developer-facing source components and packaged Windows deployment artifacts. It is intended for secure biometric enrollment and local computer unlock scenarios using the Mantra MFS100 fingerprint device.

## Status

This project is a specialized Windows biometric authentication implementation focused on fingerprint-based session unlock and local credential protection.

---

<p align="center">
  <sub>Built for secure local identity verification on Windows with Mantra MFS100 hardware.</sub>
</p>
