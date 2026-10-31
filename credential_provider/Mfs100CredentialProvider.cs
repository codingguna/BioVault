using System;
using System.Collections.Generic;
using System.Drawing;
using System.IO;
using System.IO.Pipes;
using System.Runtime.InteropServices;
using System.Text;
using System.Threading;

namespace Mantra.MFS100.CredentialProvider
{
    // =========================================================================
    // COM Interface Definitions for Windows 10 / 11 LogonUI (V2 Provider)
    // =========================================================================

    public enum CREDENTIAL_PROVIDER_USAGE_SCENARIO : uint
    {
        CPUS_LOGON = 0,
        CPUS_UNLOCK_WORKSTATION = 1,
        CPUS_CHANGE_PASSWORD = 2,
        CPUS_CREDUI = 3,
        CPUS_PLAP = 4,
        CPUS_INVALID = 5
    }

    public enum CREDENTIAL_PROVIDER_FIELD_TYPE : uint
    {
        CPFT_INVALID = 0,
        CPFT_LARGE_TEXT = 1,
        CPFT_SMALL_TEXT = 2,
        CPFT_COMMAND_LINK = 3,
        CPFT_EDIT_TEXT = 4,
        CPFT_PASSWORD_TEXT = 5,
        CPFT_TILE_IMAGE = 6,
        CPFT_CHECKBOX = 7,
        CPFT_COMBOBOX = 8,
        CPFT_SUBMIT_BUTTON = 9
    }

    public enum CREDENTIAL_PROVIDER_FIELD_STATE : uint
    {
        CPFS_HIDDEN = 0,
        CPFS_DISPLAY_IN_SELECTED_TILE = 1,
        CPFS_DISPLAY_IN_DESELECTED_TILE = 2,
        CPFS_DISPLAY_IN_BOTH = 3
    }

    public enum CREDENTIAL_PROVIDER_FIELD_INTERACTIVE_STATE : uint
    {
        CPFIS_NONE = 0,
        CPFIS_READONLY = 1,
        CPFIS_FOCUSED = 2
    }

    public enum CREDENTIAL_PROVIDER_GET_SERIALIZATION_RESPONSE : uint
    {
        CPGSR_NO_CREDENTIAL_NOT_FINISHED = 0,
        CPGSR_RETURN_CREDENTIAL_FINISHED = 1
    }

    public enum CREDENTIAL_PROVIDER_STATUS_ICON : uint
    {
        CPSI_NONE = 0,
        CPSI_ERROR = 1,
        CPSI_WARNING = 2,
        CPSI_SUCCESS = 3
    }

    [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
    public struct CREDENTIAL_PROVIDER_FIELD_DESCRIPTOR
    {
        public uint dwFieldID;
        public CREDENTIAL_PROVIDER_FIELD_TYPE cpft;
        [MarshalAs(UnmanagedType.LPWStr)]
        public string pszLabel;
        public Guid guidFieldType;
    }

    [StructLayout(LayoutKind.Sequential)]
    public struct CREDENTIAL_PROVIDER_CREDENTIAL_SERIALIZATION
    {
        public uint ulAuthenticationPackage;
        public Guid clsipCredentialProvider;
        public uint cbSerialization;
        public IntPtr rgbSerialization;
    }

    [ComImport]
    [Guid("60109359-70a0-4e2c-ac5f-75c3260c6488")]
    [InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface ICredentialProviderCredentialEvents
    {
        [PreserveSig]
        int SetFieldStatePair(ICredentialProviderCredential pcpc, uint dwFieldID, CREDENTIAL_PROVIDER_FIELD_STATE cpfs, CREDENTIAL_PROVIDER_FIELD_INTERACTIVE_STATE cpfis);
        [PreserveSig]
        int SetFieldString(ICredentialProviderCredential pcpc, uint dwFieldID, [MarshalAs(UnmanagedType.LPWStr)] string psz);
        [PreserveSig]
        int OnCredentialsChanged(ICredentialProviderCredential pcpc);
    }

    [ComImport]
    [Guid("63910a34-4083-4c7a-9c49-ac28697e4073")]
    [InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface ICredentialProviderCredential
    {
        [PreserveSig]
        int Advise(ICredentialProviderCredentialEvents pcpce);
        [PreserveSig]
        int Unadvise();
        [PreserveSig]
        int SetSelected(out int pbAutoLogon);
        [PreserveSig]
        int SetDeselected();
        [PreserveSig]
        int GetFieldState(uint dwIndex, out CREDENTIAL_PROVIDER_FIELD_STATE pcpfs, out CREDENTIAL_PROVIDER_FIELD_INTERACTIVE_STATE pcpfis);
        [PreserveSig]
        int GetStringValue(uint dwIndex, [MarshalAs(UnmanagedType.LPWStr)] out string ppsz);
        [PreserveSig]
        int GetBitmapValue(uint dwIndex, out IntPtr phbmp);
        [PreserveSig]
        int GetCheckboxValue(uint dwIndex, out int pbChecked, [MarshalAs(UnmanagedType.LPWStr)] out string ppszLabel);
        [PreserveSig]
        int SetStringValue(uint dwIndex, [MarshalAs(UnmanagedType.LPWStr)] string psz);
        [PreserveSig]
        int SetCheckboxValue(uint dwIndex, int bChecked);
        [PreserveSig]
        int CommandLinkClicked(uint dwIndex);
        [PreserveSig]
        int GetSerialization(
            out CREDENTIAL_PROVIDER_GET_SERIALIZATION_RESPONSE pcpgsr,
            out CREDENTIAL_PROVIDER_CREDENTIAL_SERIALIZATION pcpcs,
            [MarshalAs(UnmanagedType.LPWStr)] out string ppszOptionalStatusText,
            out CREDENTIAL_PROVIDER_STATUS_ICON pcpsiOptionalStatusIcon);
        [PreserveSig]
        int ReportResult(int status, int substatus, [MarshalAs(UnmanagedType.LPWStr)] out string ppszOptionalStatusText, out CREDENTIAL_PROVIDER_STATUS_ICON pcpsiOptionalStatusIcon);
    }

    [ComImport]
    [Guid("fd672c54-40ea-4d6e-9b49-cfb1a7507bd7")]
    [InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface ICredentialProviderCredential2 : ICredentialProviderCredential
    {
        [PreserveSig]
        new int Advise(ICredentialProviderCredentialEvents pcpce);
        [PreserveSig]
        new int Unadvise();
        [PreserveSig]
        new int SetSelected(out int pbAutoLogon);
        [PreserveSig]
        new int SetDeselected();
        [PreserveSig]
        new int GetFieldState(uint dwIndex, out CREDENTIAL_PROVIDER_FIELD_STATE pcpfs, out CREDENTIAL_PROVIDER_FIELD_INTERACTIVE_STATE pcpfis);
        [PreserveSig]
        new int GetStringValue(uint dwIndex, [MarshalAs(UnmanagedType.LPWStr)] out string ppsz);
        [PreserveSig]
        new int GetBitmapValue(uint dwIndex, out IntPtr phbmp);
        [PreserveSig]
        new int GetCheckboxValue(uint dwIndex, out int pbChecked, [MarshalAs(UnmanagedType.LPWStr)] out string ppszLabel);
        [PreserveSig]
        new int SetStringValue(uint dwIndex, [MarshalAs(UnmanagedType.LPWStr)] string psz);
        [PreserveSig]
        new int SetCheckboxValue(uint dwIndex, int bChecked);
        [PreserveSig]
        new int CommandLinkClicked(uint dwIndex);
        [PreserveSig]
        new int GetSerialization(
            out CREDENTIAL_PROVIDER_GET_SERIALIZATION_RESPONSE pcpgsr,
            out CREDENTIAL_PROVIDER_CREDENTIAL_SERIALIZATION pcpcs,
            [MarshalAs(UnmanagedType.LPWStr)] out string ppszOptionalStatusText,
            out CREDENTIAL_PROVIDER_STATUS_ICON pcpsiOptionalStatusIcon);
        [PreserveSig]
        new int ReportResult(int status, int substatus, [MarshalAs(UnmanagedType.LPWStr)] out string ppszOptionalStatusText, out CREDENTIAL_PROVIDER_STATUS_ICON pcpsiOptionalStatusIcon);

        [PreserveSig]
        int GetUserSid([MarshalAs(UnmanagedType.LPWStr)] out string ppszSid);
    }

    [ComImport]
    [Guid("d27c3481-5a1f-46bc-9a7d-3b0e376e27f2")]
    [InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface ICredentialProvider
    {
        [PreserveSig]
        int SetUsageScenario(CREDENTIAL_PROVIDER_USAGE_SCENARIO cpus, uint dwFlags);
        [PreserveSig]
        int SetSerialization(ref CREDENTIAL_PROVIDER_CREDENTIAL_SERIALIZATION pcpcs);
        [PreserveSig]
        int Advise(ICredentialProviderEvents pcpe, UIntPtr upAdviseContext);
        [PreserveSig]
        int Unadvise();
        [PreserveSig]
        int GetFieldDescriptorAt(uint dwIndex, out IntPtr ppcpfd);
        [PreserveSig]
        int GetFieldCategoryAt(uint dwIndex, out uint pcpfc);
        [PreserveSig]
        int GetCredentialCount(out uint pdwCount, out uint pdwDefault, out int pbAutoLogonWithDefault);
        [PreserveSig]
        int GetCredentialAt(uint dwIndex, out ICredentialProviderCredential ppcpc);
    }

    [ComImport]
    [Guid("095c1484-1c0c-4388-9c6d-500e61bf84bd")]
    [InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface ICredentialProviderSetUserArray
    {
        [PreserveSig]
        int SetUserArray(ICredentialProviderUserArray users);
    }

    [ComImport]
    [Guid("90c119ae-0f18-4520-a1f1-114366a40fe8")]
    [InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface ICredentialProviderUserArray
    {
        [PreserveSig]
        int SetProviderFilter(ref Guid guidProviderToFilterTo);
        [PreserveSig]
        int GetAccountOptions(out uint pdwOptions);
        [PreserveSig]
        int GetCount(out uint pdwCount);
        [PreserveSig]
        int GetAt(uint userIndex, out ICredentialProviderUser user);
    }

    [ComImport]
    [Guid("13793285-3ea6-40fd-b420-15f47da41fbb")]
    [InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface ICredentialProviderUser
    {
        [PreserveSig]
        int GetSid([MarshalAs(UnmanagedType.LPWStr)] out string ppszSid);
        [PreserveSig]
        int GetProviderID(out Guid pProviderID);
        [PreserveSig]
        int GetStringValue(uint dwIndex, [MarshalAs(UnmanagedType.LPWStr)] out string ppsz);
        [PreserveSig]
        int GetValue(uint dwIndex, out IntPtr pvar);
    }

    [ComImport]
    [Guid("f30e70a2-28e4-4d1a-be06-1dec581172a6")]
    [InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface ICredentialProviderEvents
    {
        [PreserveSig]
        int CredentialsChanged(UIntPtr upAdviseContext);
    }

    // =========================================================================
    // V2 Credential Provider Class
    // =========================================================================

    [Guid("7C6A8723-53BC-4545-98D1-9B5F9542D4C1")]
    [ClassInterface(ClassInterfaceType.None)]
    [ComVisible(true)]
    public class Mfs100Provider : ICredentialProvider, ICredentialProviderSetUserArray
    {
        public static readonly Guid ProviderGuid = new Guid("7C6A8723-53BC-4545-98D1-9B5F9542D4C1");

        private CREDENTIAL_PROVIDER_USAGE_SCENARIO _scenario = CREDENTIAL_PROVIDER_USAGE_SCENARIO.CPUS_INVALID;
        private ICredentialProviderEvents _events;
        private UIntPtr _adviseContext;
        private readonly List<Mfs100Credential> _credentials = new List<Mfs100Credential>();

        private static readonly CREDENTIAL_PROVIDER_FIELD_DESCRIPTOR[] Fields = new CREDENTIAL_PROVIDER_FIELD_DESCRIPTOR[]
        {
            new CREDENTIAL_PROVIDER_FIELD_DESCRIPTOR { dwFieldID = 0, cpft = CREDENTIAL_PROVIDER_FIELD_TYPE.CPFT_TILE_IMAGE, pszLabel = "Fingerprint Icon" },
            new CREDENTIAL_PROVIDER_FIELD_DESCRIPTOR { dwFieldID = 1, cpft = CREDENTIAL_PROVIDER_FIELD_TYPE.CPFT_LARGE_TEXT, pszLabel = "Title" },
            new CREDENTIAL_PROVIDER_FIELD_DESCRIPTOR { dwFieldID = 2, cpft = CREDENTIAL_PROVIDER_FIELD_TYPE.CPFT_SMALL_TEXT, pszLabel = "Status" },
            new CREDENTIAL_PROVIDER_FIELD_DESCRIPTOR { dwFieldID = 3, cpft = CREDENTIAL_PROVIDER_FIELD_TYPE.CPFT_COMMAND_LINK, pszLabel = "Scan Finger" }
        };

        public int SetUsageScenario(CREDENTIAL_PROVIDER_USAGE_SCENARIO cpus, uint dwFlags)
        {
            _scenario = cpus;
            if (cpus == CREDENTIAL_PROVIDER_USAGE_SCENARIO.CPUS_LOGON || cpus == CREDENTIAL_PROVIDER_USAGE_SCENARIO.CPUS_UNLOCK_WORKSTATION)
            {
                if (_credentials.Count == 0)
                {
                    _credentials.Add(new Mfs100Credential(this, cpus, ""));
                }
                return 0; // S_OK
            }
            return unchecked((int)0x80004001); // E_NOTIMPL
        }

        public int SetSerialization(ref CREDENTIAL_PROVIDER_CREDENTIAL_SERIALIZATION pcpcs)
        {
            return unchecked((int)0x80004001);
        }

        public int Advise(ICredentialProviderEvents pcpe, UIntPtr upAdviseContext)
        {
            _events = pcpe;
            _adviseContext = upAdviseContext;
            return 0;
        }

        public int Unadvise()
        {
            _events = null;
            return 0;
        }

        public int GetFieldDescriptorAt(uint dwIndex, out IntPtr ppcpfd)
        {
            ppcpfd = IntPtr.Zero;
            if (dwIndex >= Fields.Length) return unchecked((int)0x80070057);

            IntPtr ptr = Marshal.AllocHGlobal(Marshal.SizeOf(typeof(CREDENTIAL_PROVIDER_FIELD_DESCRIPTOR)));
            Marshal.StructureToPtr(Fields[dwIndex], ptr, false);
            ppcpfd = ptr;
            return 0;
        }

        public int GetFieldCategoryAt(uint dwIndex, out uint pcpfc)
        {
            pcpfc = 0;
            return 0;
        }

        public int GetCredentialCount(out uint pdwCount, out uint pdwDefault, out int pbAutoLogonWithDefault)
        {
            pdwCount = (uint)_credentials.Count;
            pdwDefault = 0;
            pbAutoLogonWithDefault = 0;
            return 0;
        }

        public int GetCredentialAt(uint dwIndex, out ICredentialProviderCredential ppcpc)
        {
            ppcpc = null;
            if (dwIndex < _credentials.Count)
            {
                ppcpc = _credentials[(int)dwIndex];
                return 0;
            }
            return unchecked((int)0x80070057);
        }

        // =========================================================================
        // ICredentialProviderSetUserArray (Windows 10 / 11 user integration)
        // =========================================================================
        public int SetUserArray(ICredentialProviderUserArray users)
        {
            _credentials.Clear();

            if (users != null)
            {
                uint count = 0;
                users.GetCount(out count);
                for (uint i = 0; i < count; i++)
                {
                    ICredentialProviderUser user;
                    if (users.GetAt(i, out user) == 0 && user != null)
                    {
                        string sid;
                        if (user.GetSid(out sid) == 0 && !string.IsNullOrEmpty(sid))
                        {
                            _credentials.Add(new Mfs100Credential(this, _scenario, sid));
                        }
                    }
                }
            }

            if (_credentials.Count == 0)
            {
                _credentials.Add(new Mfs100Credential(this, _scenario, ""));
            }

            return 0; // S_OK
        }
    }

    // =========================================================================
    // V2 Credential Tile Class
    // =========================================================================

    [ComVisible(true)]
    public class Mfs100Credential : ICredentialProviderCredential2
    {
        private readonly Mfs100Provider _provider;
        private readonly CREDENTIAL_PROVIDER_USAGE_SCENARIO _scenario;
        private readonly string _userSid;
        private ICredentialProviderCredentialEvents _events;

        private string _statusText = "Touch MFS100 scanner to unlock";
        private string _authDomain = "";
        private string _authUser = "";
        private string _authPassword = "";
        private bool _isAuthReady = false;

        public Mfs100Credential(Mfs100Provider provider, CREDENTIAL_PROVIDER_USAGE_SCENARIO scenario, string userSid)
        {
            _provider = provider;
            _scenario = scenario;
            _userSid = userSid ?? "";
        }

        public int GetUserSid(out string ppszSid)
        {
            ppszSid = _userSid;
            return 0; // S_OK
        }

        public int Advise(ICredentialProviderCredentialEvents pcpce)
        {
            _events = pcpce;
            return 0;
        }

        public int Unadvise()
        {
            _events = null;
            return 0;
        }

        public int SetSelected(out int pbAutoLogon)
        {
            pbAutoLogon = 0;
            ThreadPool.QueueUserWorkItem(ListenForFingerprint);
            return 0;
        }

        public int SetDeselected()
        {
            return 0;
        }

        public int GetFieldState(uint dwIndex, out CREDENTIAL_PROVIDER_FIELD_STATE pcpfs, out CREDENTIAL_PROVIDER_FIELD_INTERACTIVE_STATE pcpfis)
        {
            pcpfis = CREDENTIAL_PROVIDER_FIELD_INTERACTIVE_STATE.CPFIS_NONE;
            switch (dwIndex)
            {
                case 0: // Image
                case 1: // Title
                    pcpfs = CREDENTIAL_PROVIDER_FIELD_STATE.CPFS_DISPLAY_IN_BOTH;
                    break;
                case 2: // Status
                    pcpfs = CREDENTIAL_PROVIDER_FIELD_STATE.CPFS_DISPLAY_IN_SELECTED_TILE;
                    break;
                case 3: // Button
                    pcpfs = CREDENTIAL_PROVIDER_FIELD_STATE.CPFS_DISPLAY_IN_SELECTED_TILE;
                    pcpfis = CREDENTIAL_PROVIDER_FIELD_INTERACTIVE_STATE.CPFIS_FOCUSED;
                    break;
                default:
                    pcpfs = CREDENTIAL_PROVIDER_FIELD_STATE.CPFS_HIDDEN;
                    break;
            }
            return 0;
        }

        public int GetStringValue(uint dwIndex, out string ppsz)
        {
            ppsz = "";
            switch (dwIndex)
            {
                case 1: ppsz = "Mantra Fingerprint"; break;
                case 2: ppsz = _statusText; break;
                case 3: ppsz = "Scan Fingerprint"; break;
            }
            return 0;
        }

        public int GetBitmapValue(uint dwIndex, out IntPtr phbmp)
        {
            phbmp = IntPtr.Zero;
            if (dwIndex == 0)
            {
                try
                {
                    using (Bitmap bmp = new Bitmap(48, 48))
                    {
                        using (Graphics g = Graphics.FromImage(bmp))
                        {
                            g.Clear(Color.Transparent);
                            using (Pen pen = new Pen(Color.FromArgb(37, 99, 235), 3))
                            {
                                g.DrawArc(pen, 8, 6, 32, 36, 0, 360);
                                g.DrawArc(pen, 14, 12, 20, 24, 0, 360);
                                g.DrawArc(pen, 20, 18, 8, 12, 0, 360);
                            }
                        }
                        phbmp = bmp.GetHbitmap();
                    }
                }
                catch { }
            }
            return 0;
        }

        public int GetCheckboxValue(uint dwIndex, out int pbChecked, out string ppszLabel)
        {
            pbChecked = 0;
            ppszLabel = "";
            return 0;
        }

        public int SetStringValue(uint dwIndex, string psz) { return 0; }
        public int SetCheckboxValue(uint dwIndex, int bChecked) { return 0; }

        public int CommandLinkClicked(uint dwIndex)
        {
            if (dwIndex == 3)
            {
                ThreadPool.QueueUserWorkItem(ListenForFingerprint);
            }
            return 0;
        }

        private void ListenForFingerprint(object state)
        {
            try
            {
                UpdateStatus("Scanner active: Touch MFS100 now...");

                using (NamedPipeClientStream client = new NamedPipeClientStream(".", "MFS100AuthPipe", PipeDirection.InOut))
                {
                    client.Connect(5000); // Wait up to 5s for background auth service

                    string requestJson = "{\"command\":\"TRIGGER_UNLOCK\"}";
                    byte[] reqBytes = Encoding.UTF8.GetBytes(requestJson);
                    client.Write(reqBytes, 0, reqBytes.Length);
                    client.Flush();

                    byte[] respBuf = new byte[4096];
                    int read = client.Read(respBuf, 0, respBuf.Length);
                    string respJson = Encoding.UTF8.GetString(respBuf, 0, read);

                    if (respJson.Contains("\"SUCCESS\""))
                    {
                        _authUser = ExtractJsonVal(respJson, "user");
                        _authDomain = ExtractJsonVal(respJson, "domain");
                        _authPassword = ExtractJsonVal(respJson, "password");
                        _isAuthReady = true;

                        UpdateStatus("Finger verified! Unlocking...");
                        if (_events != null)
                        {
                            _events.OnCredentialsChanged(this);
                        }
                    }
                    else
                    {
                        UpdateStatus("Finger not recognized. Try again or use PIN.");
                    }
                }
            }
            catch
            {
                UpdateStatus("Auth service offline. Use PIN or Password.");
            }
        }

        private void UpdateStatus(string msg)
        {
            _statusText = msg;
            if (_events != null)
            {
                _events.SetFieldString(this, 2, _statusText);
            }
        }

        private string ExtractJsonVal(string json, string key)
        {
            string marker = "\"" + key + "\":\"";
            int idx = json.IndexOf(marker);
            if (idx == -1) return "";
            idx += marker.Length;
            int end = json.IndexOf("\"", idx);
            if (end == -1) return "";
            return json.Substring(idx, end - idx);
        }

        public int GetSerialization(
            out CREDENTIAL_PROVIDER_GET_SERIALIZATION_RESPONSE pcpgsr,
            out CREDENTIAL_PROVIDER_CREDENTIAL_SERIALIZATION pcpcs,
            out string ppszOptionalStatusText,
            out CREDENTIAL_PROVIDER_STATUS_ICON pcpsiOptionalStatusIcon)
        {
            pcpgsr = CREDENTIAL_PROVIDER_GET_SERIALIZATION_RESPONSE.CPGSR_NO_CREDENTIAL_NOT_FINISHED;
            pcpcs = new CREDENTIAL_PROVIDER_CREDENTIAL_SERIALIZATION();
            ppszOptionalStatusText = "";
            pcpsiOptionalStatusIcon = CREDENTIAL_PROVIDER_STATUS_ICON.CPSI_NONE;

            if (!_isAuthReady || string.IsNullOrEmpty(_authUser))
            {
                return 0;
            }

            IntPtr pAuthData;
            uint authDataLen;
            if (CreateKerbInteractiveLogon(_authDomain, _authUser, _authPassword, out pAuthData, out authDataLen))
            {
                pcpcs.clsipCredentialProvider = Mfs100Provider.ProviderGuid;
                pcpcs.ulAuthenticationPackage = GetNegotiateAuthPackage();
                pcpcs.cbSerialization = authDataLen;
                pcpcs.rgbSerialization = pAuthData;
                pcpgsr = CREDENTIAL_PROVIDER_GET_SERIALIZATION_RESPONSE.CPGSR_RETURN_CREDENTIAL_FINISHED;
            }

            return 0;
        }

        public int ReportResult(int status, int substatus, out string ppszOptionalStatusText, out CREDENTIAL_PROVIDER_STATUS_ICON pcpsiOptionalStatusIcon)
        {
            ppszOptionalStatusText = "";
            pcpsiOptionalStatusIcon = CREDENTIAL_PROVIDER_STATUS_ICON.CPSI_NONE;
            return 0;
        }

        // =========================================================================
        // Windows LSA Authentication Serialization
        // =========================================================================

        [DllImport("secur32.dll", SetLastError = false)]
        private static extern int LsaConnectUntrusted(out IntPtr LsaHandle);

        [DllImport("secur32.dll", SetLastError = false)]
        private static extern int LsaLookupAuthenticationPackage(IntPtr LsaHandle, ref LSA_STRING_IN PackageName, out uint AuthenticationPackage);

        [DllImport("secur32.dll", SetLastError = false)]
        private static extern int LsaDeregisterLogonProcess(IntPtr LsaHandle);

        [StructLayout(LayoutKind.Sequential)]
        private struct LSA_STRING_IN
        {
            public ushort Length;
            public ushort MaximumLength;
            public string Buffer;
        }

        private static uint GetNegotiateAuthPackage()
        {
            IntPtr lsa;
            if (LsaConnectUntrusted(out lsa) == 0)
            {
                LSA_STRING_IN pkg = new LSA_STRING_IN();
                pkg.Buffer = "Negotiate";
                pkg.Length = (ushort)pkg.Buffer.Length;
                pkg.MaximumLength = (ushort)(pkg.Buffer.Length + 1);

                uint pkgId;
                int res = LsaLookupAuthenticationPackage(lsa, ref pkg, out pkgId);
                LsaDeregisterLogonProcess(lsa);
                if (res == 0) return pkgId;
            }
            return 0;
        }

        private bool CreateKerbInteractiveLogon(string domain, string username, string password, out IntPtr pData, out uint dataLen)
        {
            byte[] domainBytes = Encoding.Unicode.GetBytes(domain);
            byte[] userBytes = Encoding.Unicode.GetBytes(username);
            byte[] passBytes = Encoding.Unicode.GetBytes(password);

            int headerSize = 4 + (3 * (IntPtr.Size == 8 ? 16 : 8));
            int totalSize = headerSize + domainBytes.Length + userBytes.Length + passBytes.Length;

            IntPtr buffer = Marshal.AllocHGlobal(totalSize);
            for (int i = 0; i < totalSize; i++) Marshal.WriteByte(buffer, i, 0);

            Marshal.WriteInt32(buffer, 0, 2); // KerbInteractiveLogon = 2

            int currentOffset = headerSize;
            Action<int, byte[]> writeString = (headerPos, bytes) =>
            {
                ushort len = (ushort)bytes.Length;
                Marshal.WriteInt16(buffer, headerPos, (short)len);
                Marshal.WriteInt16(buffer, headerPos + 2, (short)len);
                IntPtr strPtr = new IntPtr(buffer.ToInt64() + currentOffset);
                Marshal.WriteIntPtr(buffer, headerPos + (IntPtr.Size == 8 ? 8 : 4), strPtr);
                Marshal.Copy(bytes, 0, strPtr, bytes.Length);
                currentOffset += bytes.Length;
            };

            int stride = IntPtr.Size == 8 ? 16 : 8;
            writeString(4, domainBytes);
            writeString(4 + stride, userBytes);
            writeString(4 + (2 * stride), passBytes);

            pData = buffer;
            dataLen = (uint)totalSize;
            return true;
        }
    }
}
