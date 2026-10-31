# Mantra MFS100 Fingerprint Authentication for Windows

## Project plan: Go-based enrolment application first, Windows sign-in integration second

**Document status:** Proposed implementation plan\
**Target device:** Mantra MFS100, identified in Device Manager as
`MFS100`\
**Reported USB hardware ID:** `USB\VID_2C0F&PID_1005`\
**Primary language:** Go\
**Initial target:** Windows 11 64-bit (confirm exact edition/build
during setup)

------------------------------------------------------------------------

## 1. Project vision and scope

Build a local Windows application that uses the Mantra MFS100 scanner to
enrol named users and fingerprints, verify a claimed identity, and
identify a user from enrolled templates. Only after this application is
reliable should a separate Windows sign-in integration project begin.

The work is deliberately split into two independently testable projects:

1.  **Phase A --- Fingerprint Manager:** a standalone Go desktop
    application for device testing, enrolment, verification,
    identification, user/template management, and audit history.
2.  **Phase B --- Windows Sign-in Integration:** a Windows Credential
    Provider-based integration that presents a fingerprint sign-in
    option within the real Windows sign-in experience, while retaining
    supported recovery credentials.

The first phase must not modify Windows logon settings, Credential
Provider registry keys, password/PIN providers, or system security
policy.

### Explicit non-goals for Phase A

-   It will not replace the Windows sign-in screen.
-   It will not unlock Windows merely by drawing a full-screen overlay.
-   It will not store Windows account passwords in the fingerprint
    database.
-   It will not assume that Mantra RD Service is equivalent to the
    general-purpose MFS100 SDK.
-   It will not assume that a DLL can be called directly from Go until
    its exported API and calling convention are inspected.

------------------------------------------------------------------------

## 2. Evidence and facts currently available

### 2.1 Device evidence supplied in this project

-   Device Manager shows an entry named `MFS100` under **Universal
    Serial Bus controllers**.
-   The reported hardware IDs are:
    -   `USB\VID_2C0F&PID_1005&REV_0000`
    -   `USB\VID_2C0F&PID_1005`
-   The user's MFS100 datasheet describes an optical sensor, 500 DPI, 16
    × 18 mm sensing area, USB 2.0, ANSI-378 and ISO 19794-2 template
    standards, and 32-bit/64-bit SDK/library/driver support.
-   The datasheet describes image capture, feature extraction, template
    creation/storage, and template matching.

These details are evidence for selecting the original MFS100 SDK family
as the first SDK to investigate. They do **not**, by themselves, prove
the exact SDK release, DLL names, Windows 11 compatibility, or that the
hardware ID uniquely distinguishes every MFS100 revision. Record the
device's serial/model label and confirm the hardware mapping with Mantra
before treating it as definitive.

### 2.2 Official Mantra SDK information found

Mantra's official SDK page lists **MFS100 SDK 9.0.2.5** and language
sections for **.NET, Java, C++ and Web**. Its release description says
the SDK supports use of Aadhaar-registered devices as public devices in
non-Aadhaar applications and describes a wrapper library, easier
integration, memory-management improvements, event-based capture, and
changes to AutoCapture.

Official page: https://servico.mantratecapp.com/Home/SDK.html

Mantra's official driver page separately lists MFS100 Driver, MFS100
Client Service and RD Service downloads. A driver or RD Service
installer should not be mistaken for the complete developer SDK.

Official driver page: https://servico.mantratecapp.com/Home/Drivers.html

**Download decision:** obtain the SDK from Mantra's official portal or
directly from Mantra support. Record the downloaded archive name,
version, date, architecture, checksum, license/redistribution terms, and
any prerequisites. Do not use an unverified third-party SDK mirror.

### 2.3 Windows sign-in integration information

Microsoft documents Credential Providers as the mechanism used by
Windows Logon UI to collect credentials. A custom provider implements
interfaces including `ICredentialProvider` and
`ICredentialProviderCredential`; a V2 provider also implements
`ICredentialProviderCredential2`.

Microsoft strongly recommends leaving at least one built-in/system
credential provider available so the user can recover access if a
third-party provider fails. Credential Providers collect and serialize
credentials; they are not themselves the Windows authentication
enforcement mechanism.

Microsoft documentation: -
https://learn.microsoft.com/en-us/windows/win32/secauthn/credential-providers-in-windows -
https://learn.microsoft.com/en-us/windows/win32/secauthn/winlogon-and-credential-providers

**Important scope correction:** Phase B should aim to add a secure
fingerprint sign-in tile/flow to Windows Logon UI. It should not aim to
remove or replace all Windows system sign-in UI or disable password/PIN
recovery. A custom Credential Provider must ultimately submit
credentials or an authentication response that Windows can validate
through a supported authentication path. A fingerprint match alone is
not automatically a Windows logon credential.

------------------------------------------------------------------------

## 3. Architecture overview

### Phase A --- standalone Go application

``` text
┌────────────────────────────────────────────┐
│             Go desktop application         │
│  User management / Enrol / Verify / Admin  │
└──────────────────────┬─────────────────────┘
                       │
┌──────────────────────▼─────────────────────┐
│          Scanner adapter interface         │
│  Connect / Info / Capture / Cancel / Close │
└──────────────────────┬─────────────────────┘
                       │
        ┌──────────────┴─────────────────┐
        │                                │
┌───────▼─────────┐             ┌────────▼────────┐
│ Mantra SDK      │             │ Future adapter  │
│ wrapper/bridge  │             │ / test fake     │
└───────┬─────────┘             └─────────────────┘
        │
┌───────▼────────────────────────────────────┐
│ MFS100 driver + vendor runtime components  │
└────────────────────────────────────────────┘

┌────────────────────────────────────────────┐
│ Local data layer                           │
│ SQLite metadata + protected template blobs │
│ Audit events + app configuration           │
└────────────────────────────────────────────┘
```

### Phase B --- Windows sign-in integration

``` text
Windows Logon UI
       │
       ▼
Native Windows Credential Provider (C++ COM DLL)
       │  local IPC with strict access controls
       ▼
Go authentication service / scanner adapter
       │
       ▼
Mantra SDK + MFS100
       │
       ▼
Fingerprint match + account binding
       │
       ▼
Supported Windows credential/authentication path
```

The final Phase B design must be reviewed against Microsoft's Credential
Provider model and Mantra's SDK/license/security constraints. The
provider must not invent or expose a user's Windows password. If the SDK
or Windows authentication model cannot support a secure, supported flow,
stop and reassess rather than build an unsafe password-replay mechanism.

------------------------------------------------------------------------

## 4. Phase A --- Fingerprint Manager requirements

### 4.1 Core screens

1.  **Dashboard**
    -   Scanner connection state.
    -   SDK/runtime status.
    -   Number of enrolled users and templates.
    -   Recent enrolment/verification events.
2.  **User management**
    -   Create, edit, deactivate and delete local user profiles.
    -   Display name and unique application user ID.
    -   Optional notes; avoid collecting unnecessary personal
        information.
3.  **Fingerprint enrolment**
    -   Select a user.
    -   Select finger label (right/left thumb, index, middle, ring,
        little finger, or custom label).
    -   Capture multiple samples as required by the SDK.
    -   Check capture quality and retry poor samples.
    -   Generate and store the SDK-supported template.
    -   Prevent accidental duplicate enrolment where SDK matching
        supports it.
4.  **Verify claimed user**
    -   Select or enter a user ID.
    -   Scan a finger.
    -   Compare against that user's enrolled templates.
    -   Show match/no-match and a clear confidence/quality result only
        if the SDK defines one.
5.  **Identify user**
    -   Scan a finger.
    -   Compare against the enrolled template collection using the
        supported SDK matching API.
    -   Return a user only when the SDK's threshold and ambiguity rules
        are satisfied.
    -   If multiple templates match or confidence is insufficient,
        report ambiguous/no match rather than guessing.
6.  **Template management**
    -   List a user's enrolled finger labels.
    -   Re-enrol or delete a template.
    -   Never display raw template bytes in the UI or logs.
7.  **Audit history**
    -   Record event type, timestamp, application user ID (if known),
        result and diagnostic code.
    -   Do not store fingerprint images or raw templates in logs.

### 4.2 Functional acceptance criteria

Phase A is complete only when all of the following are demonstrated on
the target PC:

-   [ ] The application detects the connected MFS100 reliably.
-   [ ] It reports device/SDK errors clearly and recovers after
    unplug/replug.
-   [ ] It can capture a fingerprint using the official SDK.
-   [ ] It can enrol at least two different users.
-   [ ] It can enrol multiple fingers for one user.
-   [ ] It can verify a finger against the selected user's templates.
-   [ ] It can identify an enrolled user from the enrolled collection,
    if the SDK supports 1:N identification.
-   [ ] It rejects an unregistered finger.
-   [ ] It handles poor-quality captures and repeated scans.
-   [ ] It persists data across application restart.
-   [ ] It supports backup and restore of the local database using a
    documented secure procedure.
-   [ ] It does not require Aadhaar authentication or transmit biometric
    data to a remote server for ordinary local enrolment/matching.
-   [ ] It has been tested with the exact MFS100 unit, driver, SDK
    version and Windows build recorded in the test report.

------------------------------------------------------------------------

## 5. Phase A --- Go implementation plan

### 5.1 Repository layout

``` text
mfs100-fingerprint-manager/
├── cmd/
│   └── fingerprint-manager/
│       └── main.go
├── internal/
│   ├── app/
│   ├── scanner/
│   │   ├── scanner.go
│   │   ├── mantra/
│   │   └── mock/
│   ├── enrollment/
│   ├── matching/
│   ├── users/
│   ├── storage/
│   ├── audit/
│   └── security/
├── ui/
├── migrations/
├── docs/
├── scripts/
├── tests/
├── go.mod
└── README.md
```

Keep the scanner behind a Go interface so the application and tests do
not depend directly on Mantra-specific functions.

Example conceptual interface (final signatures depend on the SDK):

``` go
type Scanner interface {
    Connect(ctx context.Context) error
    DeviceInfo(ctx context.Context) (DeviceInfo, error)
    Capture(ctx context.Context) (CaptureResult, error)
    Close() error
}
```

### 5.2 SDK integration strategy

Do not start by compiling the complete vendor SDK or copying large
binary folders blindly.

First inspect the official SDK package and identify:

-   .NET assemblies and sample projects.
-   Native DLLs and their exported functions.
-   32-bit versus 64-bit binaries.
-   Required Visual C++ runtimes.
-   Device driver and client-service dependencies.
-   SDK licence and redistribution restrictions.
-   Whether capture, template extraction, 1:1 verification and 1:N
    identification are supported.
-   Whether the SDK's template format is proprietary or standards-based.

Possible Go integration approaches, in preferred order:

1.  **Vendor-supported local service/API:** use it only if Mantra
    documents a supported non-Aadhaar capture API suitable for this use.
2.  **C-compatible exported DLL:** call it from Go using a carefully
    defined Windows interop layer, after verifying exports, calling
    conventions, ownership and thread-safety.
3.  **Small C++ or .NET bridge:** create a narrow adapter around the
    official SDK and communicate with the Go app through local named
    pipes or another authenticated local IPC mechanism.
4.  **Pure Go USB protocol implementation:** do not attempt unless
    Mantra documents the device protocol and permits this use. Do not
    reverse-engineer a security-sensitive proprietary protocol as the
    first approach.

The official portal lists .NET, Java, C++ and Web SDK sections; it does
not list Go as a supported SDK language. Therefore Go integration is a
project task to prove, not an assumed vendor-supported feature.

### 5.3 Database design

Use SQLite for application metadata. Suggested tables:

**users** - `id` --- application-generated UUID - `display_name` -
`created_at` - `updated_at` - `status`

**fingerprint_templates** - `id` --- UUID - `user_id` --- foreign key -
`finger_label` - `template_format` - `sdk_version` -
`template_blob_encrypted` - `created_at` - `quality_metadata` --- only
non-sensitive metadata required for diagnostics

**audit_events** - `id` - `event_time` - `event_type` - `user_id` ---
nullable - `result` - `diagnostic_code`

Store template bytes as opaque vendor/standard template data. Do not
assume templates from different SDK versions or sensors are
interchangeable. Include template format and SDK version metadata and
plan for re-enrolment when formats are incompatible.

### 5.4 Privacy and security controls

-   Obtain informed consent before enrolling a person's fingerprint.
-   Store only templates required for matching; avoid retaining
    fingerprint images unless essential for a documented SDK workflow.
-   Encrypt sensitive template blobs at rest using Windows DPAPI or a
    reviewed key-management design.
-   Restrict database and backup file permissions to authorized Windows
    users/admins.
-   Do not log templates, images, SDK secrets or authentication
    credentials.
-   Protect local IPC against other local users/processes.
-   Provide user deletion and template deletion.
-   Provide a documented backup/restore and data-retention policy.
-   Keep a non-biometric recovery method for the application itself.
-   Treat fingerprint matching as probabilistic; implement retry,
    rejection and ambiguity handling.
-   Do not use the app as the sole access-control mechanism for
    high-risk systems until independently security-reviewed.

------------------------------------------------------------------------

## 6. Phase A --- milestones and test plan

### Milestone A0: environment inventory

-   Record Windows edition, version and build.
-   Record device label, USB hardware ID, serial number if available.
-   Record installed driver provider/version/date.
-   Determine whether RD Service, Client Service or another Mantra
    runtime is installed.
-   Obtain the official MFS100 SDK package and documentation.
-   Record package version, architecture, checksums and prerequisites.

**Exit gate:** exact hardware/driver/SDK combination is documented.

### Milestone A1: SDK proof of concept

Build a small command-line or diagnostic program that:

-   Loads the selected SDK/runtime.
-   Initializes the scanner.
-   Reads device information.
-   Captures one fingerprint.
-   Reports quality/status and safely releases SDK resources.

**Exit gate:** repeated capture works; unplug/replug and error cases are
understood.

### Milestone A2: template proof of concept

-   Capture samples.
-   Generate templates using the SDK.
-   Test 1:1 verification.
-   Test whether the SDK exposes 1:N identification.
-   Confirm template format, versioning and matching thresholds.

**Exit gate:** matching is repeatable and false-accept/false-reject
behaviour is measured sufficiently for a prototype. Do not claim
biometric-grade security based on a few successful scans.

### Milestone A3: user and template database

-   Implement user CRUD.
-   Enrol multiple fingers per user.
-   Store encrypted templates.
-   Add deletion and backup/restore.
-   Add unit and integration tests.

**Exit gate:** data persists safely and can be recovered from a test
backup.

### Milestone A4: desktop UI

-   Build dashboard, user list, enrolment, verification, identification
    and history screens.
-   Add scanner status and troubleshooting.
-   Add cancellation, timeouts and safe device cleanup.

**Exit gate:** non-technical test user can complete enrolment and
verification without developer tools.

### Milestone A5: release candidate

-   Test on the target Windows PC.
-   Test multiple users/fingers and unregistered fingers.
-   Test poor capture, scanner disconnect, reboot, app crash and
    database recovery.
-   Create installer and uninstall process.
-   Document SDK/runtime dependencies and redistribution obligations.

**Exit gate:** Phase A acceptance checklist passes.

------------------------------------------------------------------------

## 7. Phase B --- genuine Windows sign-in integration

This phase begins only after Phase A is stable and Mantra SDK
compatibility is proven.

### 7.1 Correct Windows integration model

Use a native Windows Credential Provider (typically a C++ COM DLL) to
integrate with Windows Logon UI. Keep Go responsible for the biometric
service and application logic where feasible.

The provider may present a fingerprint sign-in tile and coordinate
scanning, but it must ultimately participate in a supported Windows
authentication flow. A fingerprint template match is not itself a
Windows password, PIN, or logon credential.

### 7.2 Critical design questions to resolve before coding

1.  What Windows account types are in scope: local accounts, Microsoft
    accounts, domain accounts, or Entra ID accounts?
2.  What supported authentication mechanism will validate the user after
    the fingerprint match?
3.  Can Mantra SDK run safely in the logon environment, or must a
    service/bridge run separately?
4.  How will account-to-biometric-template binding be provisioned and
    protected?
5.  What happens when the scanner is missing, damaged, disconnected, or
    the finger cannot be read?
6.  How will Windows updates, safe mode, remote access, credential UI
    and unlock scenarios behave?
7.  What are the SDK licensing and redistribution terms for a Credential
    Provider product?

If no supported secure authentication handoff is available, Phase B must
stop at a demonstrator and must not simulate Windows logon by
storing/replaying passwords.

### 7.3 Phase B architecture

-   Native C++ Credential Provider implementing the Windows interfaces.
-   Go service for scanner interaction and matching, if compatible with
    logon-time operation.
-   Authenticated local IPC between provider and service.
-   Per-Windows-account enrolment and binding workflow.
-   Strict timeouts and cancellation.
-   Secure handling of service lifecycle and scanner contention.
-   Signed binaries and controlled installer/rollback.
-   Event logging that excludes biometric templates and secrets.

### 7.4 Windows safety requirements

-   Keep Windows password and/or PIN recovery available.
-   Never disable all built-in credential providers.
-   Never test first on the only machine/account needed for work.
-   Use a VM or spare test PC and retain a known-good administrator
    account.
-   Create a recovery plan and tested uninstall/rollback procedure
    before registering the provider.
-   Do not write experimental provider registry entries on the primary
    PC.
-   Test after Windows updates and driver updates.
-   Have an independent Windows security review before production use.

### 7.5 Phase B test matrix

-   Windows lock (`Win + L`) and unlock.
-   Sign-out and fresh sign-in.
-   Restart and cold boot.
-   Scanner unplugged before sign-in.
-   Scanner unplugged during scan.
-   Unknown finger and poor-quality finger.
-   Multiple enrolled users and account selection.
-   Password/PIN fallback.
-   Remote Desktop and Credential UI scenarios, where applicable.
-   Safe mode/recovery and provider failure.
-   Windows update, driver update and application upgrade.
-   Uninstall and restoration of normal Windows sign-in.

------------------------------------------------------------------------

## 8. Risks and mitigations

  -----------------------------------------------------------------------
  Risk                                Mitigation
  ----------------------------------- -----------------------------------
  SDK download is unavailable         Request SDK 9.0.2.5 and samples
                                      directly from Mantra; record
                                      official response

  Hardware ID/model revision mismatch Confirm label/serial and SDK
                                      compatibility with Mantra

  SDK only offers .NET/C++ APIs       Use a small bridge; keep business
                                      logic in Go

  SDK DLL architecture mismatch       Match process architecture to
                                      vendor binaries

  RD Service is mistaken for general  Use documented SDK capture/matching
  SDK                                 APIs; distinguish RD workflows

  1:N identification is not exposed   Use supported matching APIs or
                                      redesign UX around 1:1 verification

  Proprietary templates change by SDK Store format/version metadata and
  version                             require re-enrolment when needed

  Credential Provider cannot securely Stop Phase B or use a supported
  authenticate Windows account        authentication architecture; never
                                      replay stored passwords

  Provider failure blocks access      Keep built-in password/PIN
                                      providers and test rollback

  Biometric false                     Use conservative SDK thresholds,
  acceptance/rejection                fallback, testing and security
                                      review
  -----------------------------------------------------------------------

------------------------------------------------------------------------

## 9. Immediate action list

The next work should be done in this order:

1.  Confirm Windows edition/build and 64-bit architecture.
2.  Capture the MFS100 device label/serial and driver details from
    Device Manager.
3.  Obtain the official MFS100 SDK 9.0.2.5 package from Mantra's SDK
    portal or support.
4.  Extract the SDK to a separate development folder; do not replace the
    installed driver yet.
5.  Inspect the SDK archive and identify its sample app, wrapper
    library, native DLLs, runtime dependencies and documentation.
6.  Build a minimal Go scanner diagnostic using a mock adapter first.
7.  Build the smallest real-device capture proof of concept through the
    supported SDK interface.
8.  Prove template generation and matching before implementing the
    UI/database.
9.  Complete Phase A and its acceptance tests.
10. Only then design and prototype Phase B with a recovery-safe
    Credential Provider.

### Information to collect before implementation

-   Windows edition, version and build number.
-   MFS100 label/serial and USB hardware ID.
-   Device Manager driver provider, version and date.
-   Whether Mantra RD Service or MFS100 Client Service is installed.
-   Official SDK archive and version.
-   SDK architecture and sample project files.
-   Any SDK licence or redistribution restrictions.

------------------------------------------------------------------------

## 10. Reference links

### Mantra

-   MFS100 SDK portal: https://servico.mantratecapp.com/Home/SDK.html
-   Mantra driver downloads:
    https://servico.mantratecapp.com/Home/Drivers.html
-   MFS100 product datasheet:
    https://www.mantratec.com/images/Products-Download-Brochure/MFS100.pdf

### Microsoft

-   Credential Providers in Windows:
    https://learn.microsoft.com/en-us/windows/win32/secauthn/credential-providers-in-windows
-   Winlogon and Credential Providers:
    https://learn.microsoft.com/en-us/windows/win32/secauthn/winlogon-and-credential-providers

------------------------------------------------------------------------

## 11. Definition of success

**Phase A success:** A working, documented Go Windows application can
register named users, enrol multiple fingers per user, verify a claimed
user, identify enrolled users if supported by the SDK, and manage local
templates securely with the MFS100.

**Phase B success:** A separately tested Windows Credential Provider
offers a fingerprint sign-in path through Windows Logon UI, uses a
supported authentication handoff, preserves password/PIN recovery, and
has a tested rollback process.

**Project principle:** Prove the scanner SDK and biometric matching
first. Build a reliable fingerprint manager second. Attempt real Windows
sign-in integration only after the first two are demonstrably working.
