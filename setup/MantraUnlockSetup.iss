; =====================================================================
; BioVault™ Biometric Windows Logon Setup Script
; Unified Standard Installer & Uninstaller for Windows 10 & 11 (64-bit)
; =====================================================================

#define MyAppName "BioVault Biometric Logon"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "BioVault Security Systems"
#define MyAppExeName "BioVaultManager.exe"
#define MyServiceExeName "mfs100-auth-service.exe"
#define MyClsid "{7C6A8723-53BC-4545-98D1-9B5F9542D4C1}"

[Setup]
AppId={{7C6A8723-53BC-4545-98D1-9B5F9542D4C1}}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\BioVault
DefaultGroupName=BioVault Biometric Security
PrivilegesRequired=admin
PrivilegesRequiredOverridesAllowed=dialog
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
SetupIconFile=app.ico
OutputDir=output
OutputBaseFilename=BioVault_Setup_v1.0.0
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
DisableWelcomePage=no
CloseApplications=yes
RestartApplications=no

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"

[Files]
; Main Elevated GUI Application (No HTTP server, UAC Admin manifest, BioVault brand)
Source: "bin\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion
Source: "bin\MantraFingerprintManager.exe"; DestDir: "{app}"; Flags: ignoreversion

; 32-bit Background Authentication Service
Source: "bin\{#MyServiceExeName}"; DestDir: "{app}"; Flags: ignoreversion

; Service Registration Helper Script
Source: "bin\register_service.bat"; DestDir: "{app}"; Flags: ignoreversion

; 64-bit Native Credential Provider DLL (installed into System32)
Source: "bin\Mfs100CredentialProviderNative.dll"; DestDir: "{sys}"; Flags: 64bit restartreplace uninsrestartdelete

; Bundled Official Mantra Driver Setup (extracted to temp and executed silently)
Source: "bin\MFS100Driver_9.2.0.0.exe"; DestDir: "{tmp}"; Flags: deleteafterinstall

[Dirs]
; Protected vault directory for SQLite database and logs
Name: "{commonappdata}\MantraUnlock"; Flags: uninsneveruninstall
Name: "{commonappdata}\MantraUnlock\Logs"; Flags: uninsneveruninstall

[Icons]
Name: "{group}\{#MyAppName} Manager"; Filename: "{app}\{#MyAppExeName}"; Comment: "Configure fingerprint biometrics and Windows sign-in credentials"
Name: "{group}\Uninstall {#MyAppName}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#MyAppName} Manager"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Registry]
; Register 64-bit COM InprocServer32 for the Native Credential Provider
Root: HKLM64; Subkey: "SOFTWARE\Classes\CLSID\{{7C6A8723-53BC-4545-98D1-9B5F9542D4C1}"; ValueType: string; ValueData: "BioVaultCredentialProvider"; Flags: uninsdeletekey
Root: HKLM64; Subkey: "SOFTWARE\Classes\CLSID\{{7C6A8723-53BC-4545-98D1-9B5F9542D4C1}\InprocServer32"; ValueType: string; ValueData: "{sys}\Mfs100CredentialProviderNative.dll"; Flags: uninsdeletekey
Root: HKLM64; Subkey: "SOFTWARE\Classes\CLSID\{{7C6A8723-53BC-4545-98D1-9B5F9542D4C1}\InprocServer32"; ValueType: string; ValueName: "ThreadingModel"; ValueData: "Apartment"; Flags: uninsdeletekey

; Register in Windows LogonUI Credential Providers list
Root: HKLM64; Subkey: "SOFTWARE\Microsoft\Windows\CurrentVersion\Authentication\Credential Providers\{{7C6A8723-53BC-4545-98D1-9B5F9542D4C1}"; ValueType: string; ValueData: "BioVaultCredentialProvider"; Flags: uninsdeletekey

[Run]
; Launch the Manager application at the end of setup (elevated with current admin token)
Filename: "{app}\{#MyAppExeName}"; Description: "Launch {#MyAppName} Manager"; Flags: nowait postinstall skipifsilent runascurrentuser

[UninstallRun]
; Terminate processes before file removal
Filename: "taskkill"; Parameters: "/f /im {#MyServiceExeName} /im {#MyAppExeName} /im MantraFingerprintManager.exe"; Flags: runhidden; RunOnceId: "KillAppProcs"
Filename: "schtasks"; Parameters: "/end /tn ""MFS100AuthService"""; Flags: runhidden; RunOnceId: "StopAuthTask"
Filename: "schtasks"; Parameters: "/delete /f /tn ""MFS100AuthService"""; Flags: runhidden; RunOnceId: "DeleteAuthTask"

[Code]
// -------------------------------------------------------------------
// Helper: Execute a command and wait for completion
// -------------------------------------------------------------------
function ExecCommand(const Filename, Params: String; var ResultCode: Integer): Boolean;
begin
  Result := Exec(Filename, Params, '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
end;

// -------------------------------------------------------------------
// Helper: Check registry for existing installation across both CLSIDs and 32/64 bit views
// -------------------------------------------------------------------
function GetInstalledUninstaller(var UninstallerPath: String): Boolean;
var
  Clsids: array[0..1] of String;
  I: Integer;
  SubKey: String;
begin
  Result := False;
  Clsids[0] := '{7C6A8723-53BC-4545-98D1-9B5F9542D4C1}';
  Clsids[1] := '{7B1E4398-75F9-4670-9831-C58DB1F25832}';

  for I := 0 to 1 do
  begin
    SubKey := 'SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\' + Clsids[I] + '_is1';
    if RegQueryStringValue(HKLM64, SubKey, 'UninstallString', UninstallerPath) then
    begin
      Result := True;
      Exit;
    end;
    if RegQueryStringValue(HKLM32, SubKey, 'UninstallString', UninstallerPath) then
    begin
      Result := True;
      Exit;
    end;
  end;
end;

// -------------------------------------------------------------------
// Single Setup Handler: Detect existing installation & handle Install / Uninstall / Reinstall
// -------------------------------------------------------------------
function InitializeSetup(): Boolean;
var
  UninstallerPath: String;
  ResultCode: Integer;
  Choice: Integer;
begin
  Result := True;

  // Check if BioVault / Mantra Biometric Logon is already installed on this PC
  if GetInstalledUninstaller(UninstallerPath) then
  begin
    Choice := MsgBox('BioVault Biometric Logon is ALREADY INSTALLED on this computer.'#13#10#13#10 +
                     'What action would you like to perform?'#13#10#13#10 +
                     '• Click [YES] to completely UNINSTALL BioVault Biometric Logon.'#13#10 +
                     '• Click [NO] to REINSTALL / UPDATE the installation.'#13#10 +
                     '• Click [CANCEL] to abort and exit.',
                     mbConfirmation, MB_YESNOCANCEL);

    if Choice = IDYES then
    begin
      // Launch standard uninstaller
      Exec(RemoveQuotes(UninstallerPath), '', '', SW_SHOW, ewWaitUntilTerminated, ResultCode);
      Result := False; // Exit setup after uninstaller completes
      Exit;
    end
    else if Choice = IDCANCEL then
    begin
      Result := False; // Abort setup
      Exit;
    end;
    // If Choice = IDNO, continue with Reinstall/Upgrade!
  end;
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  ResultCode: Integer;
  DriverDll: String;
  DriverInstaller: String;
  ProgramDataDir: String;
begin
  if CurStep = ssInstall then
  begin
    // Clean up any old CLSID registration from previous iterations
    RegDeleteKeyIncludingSubkeys(HKLM64, 'SOFTWARE\Classes\CLSID\{7B1E4398-75F9-4670-9831-C58DB1F25832}');
    RegDeleteKeyIncludingSubkeys(HKLM64, 'SOFTWARE\Microsoft\Windows\CurrentVersion\Authentication\Credential Providers\{7B1E4398-75F9-4670-9831-C58DB1F25832}');
    RegDeleteKeyIncludingSubkeys(HKLM64, 'SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\{7B1E4398-75F9-4670-9831-C58DB1F25832}_is1');

    // Check if Mantra MFS100 driver DLL is already present on the target system
    DriverDll := ExpandConstant('{pf}\Mantra\MFS100\Driver\MFS100Test\MFS100Dll.dll');
    if not FileExists(DriverDll) then
    begin
      DriverInstaller := ExpandConstant('{tmp}\MFS100Driver_9.2.0.0.exe');
      if FileExists(DriverInstaller) then
      begin
        Log('Installing bundled Mantra MFS100 Driver silently...');
        ExecCommand(DriverInstaller, '/VERYSILENT /SUPPRESSMSGBOXES /NORESTART', ResultCode);
        Log('Driver installer completed with code: ' + IntToStr(ResultCode));
      end;
    end
    else
    begin
      Log('Mantra MFS100 Driver is already installed.');
    end;
  end
  else if CurStep = ssPostInstall then
  begin
    ProgramDataDir := ExpandConstant('{commonappdata}\MantraUnlock');

    // 1. Enforce strict NTFS ACLs on Vault: only SYSTEM and Administrators allowed
    Log('Hardening NTFS permissions on ' + ProgramDataDir);
    ExecCommand('icacls.exe', '"' + ProgramDataDir + '" /inheritance:r /grant:r "NT AUTHORITY\SYSTEM":(OI)(CI)F /grant:r "BUILTIN\Administrators":(OI)(CI)F', ResultCode);

    // 2. Register & run the background service via register_service.bat
    Log('Registering and starting MFS100AuthService background service...');
    ExecCommand(ExpandConstant('{cmd}'), '/c "' + ExpandConstant('{app}\register_service.bat') + '"', ResultCode);
    Log('register_service.bat exited with code: ' + IntToStr(ResultCode));
  end;
end;

// -------------------------------------------------------------------
// Post-Uninstall Cleanup Handler
// -------------------------------------------------------------------
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  DataDir: String;
  DelChoice: Integer;
begin
  if CurUninstallStep = usPostUninstall then
  begin
    // Clean up registry keys if any remain
    RegDeleteKeyIncludingSubkeys(HKLM64, 'SOFTWARE\Classes\CLSID\{7C6A8723-53BC-4545-98D1-9B5F9542D4C1}');
    RegDeleteKeyIncludingSubkeys(HKLM64, 'SOFTWARE\Microsoft\Windows\CurrentVersion\Authentication\Credential Providers\{7C6A8723-53BC-4545-98D1-9B5F9542D4C1}');
    RegDeleteKeyIncludingSubkeys(HKLM64, 'SOFTWARE\Classes\CLSID\{7B1E4398-75F9-4670-9831-C58DB1F25832}');
    RegDeleteKeyIncludingSubkeys(HKLM64, 'SOFTWARE\Microsoft\Windows\CurrentVersion\Authentication\Credential Providers\{7B1E4398-75F9-4670-9831-C58DB1F25832}');

    DataDir := ExpandConstant('{commonappdata}\MantraUnlock');
    if DirExists(DataDir) then
    begin
      DelChoice := MsgBox('Do you also want to DELETE all saved fingerprint templates and user credentials in ' + DataDir + '?'#13#10#13#10 +
                          '(Click [Yes] to permanently delete biometric data, or [No] to keep them for future reinstall)',
                          mbConfirmation, MB_YESNO);
      if DelChoice = IDYES then
      begin
        DelTree(DataDir, True, True, True);
      end;
    end;
  end;
end;
