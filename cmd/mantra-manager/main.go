package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"fp-manager/internal/matching"
	"fp-manager/internal/scanner"
	"fp-manager/internal/scanner/mantra"
	"fp-manager/internal/security"
	"fp-manager/internal/storage"
)

// CRITICAL: Lock main goroutine to OS thread for Win32 message pump stability
func init() {
	runtime.LockOSThread()
}

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")

	procGetModuleHandleW     = kernel32.NewProc("GetModuleHandleW")
	procRegisterClassExW     = user32.NewProc("RegisterClassExW")
	procCreateWindowExW      = user32.NewProc("CreateWindowExW")
	procDefWindowProcW       = user32.NewProc("DefWindowProcW")
	procShowWindow           = user32.NewProc("ShowWindow")
	procUpdateWindow         = user32.NewProc("UpdateWindow")
	procGetMessageW          = user32.NewProc("GetMessageW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessageW     = user32.NewProc("DispatchMessageW")
	procPostQuitMessage      = user32.NewProc("PostQuitMessage")
	procSendMessageW         = user32.NewProc("SendMessageW")
	procSetWindowTextW       = user32.NewProc("SetWindowTextW")
	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procEnableWindow         = user32.NewProc("EnableWindow")
	procDestroyWindow        = user32.NewProc("DestroyWindow")
	procInitCommonControlsEx = comctl32.NewProc("InitCommonControlsEx")
	procCreateFontW          = gdi32.NewProc("CreateFontW")
	procMessageBoxW          = user32.NewProc("MessageBoxW")
	procRtlMoveMemory        = kernel32.NewProc("RtlMoveMemory")
	procLoadIconW            = user32.NewProc("LoadIconW")
)

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_TABSTOP          = 0x00010000
	WS_BORDER           = 0x00800000
	WS_VSCROLL          = 0x00200000
	WS_CLIPSIBLINGS     = 0x04000000

	ES_AUTOHSCROLL = 0x0080
	ES_AUTOVSCROLL = 0x0040
	ES_MULTILINE   = 0x0004
	ES_READONLY    = 0x0800
	ES_PASSWORD    = 0x0020

	CBS_DROPDOWNLIST = 0x0003
	CBS_HASSTRINGS   = 0x0200
	LBS_NOTIFY       = 0x0001

	BS_PUSHBUTTON = 0x00000000
	SS_LEFT       = 0x00000000

	SW_HIDE = 0
	SW_SHOW = 5

	WM_DESTROY = 0x0002
	WM_SETFONT = 0x0030
	WM_COMMAND = 0x0111
	WM_NOTIFY  = 0x004E

	TCM_FIRST       = 0x1300
	TCM_INSERTITEMW = TCM_FIRST + 62
	TCM_GETCURSEL   = TCM_FIRST + 11
	TCIF_TEXT       = 0x0001
	TCN_FIRST       = -550
	TCN_SELCHANGE   = TCN_FIRST - 1

	CB_ADDSTRING    = 0x0143
	CB_RESETCONTENT = 0x014B
	CB_SETCURSEL    = 0x014E
	CB_GETCURSEL    = 0x0147
	CB_GETLBTEXT    = 0x0148

	LB_ADDSTRING    = 0x0180
	LB_RESETCONTENT = 0x0184
	LB_SETCURSEL    = 0x0186
	LB_GETCURSEL    = 0x0188
	LB_GETTEXT      = 0x0189

	ICC_TAB_CLASSES = 0x00000008

	MB_OK              = 0x00000000
	MB_ICONINFORMATION = 0x00000040
	MB_ICONWARNING     = 0x00000030
	MB_ICONERROR       = 0x00000010
)

const (
	ID_TABCTRL        = 100
	ID_BTN_REFRESHSYS = 101
	ID_BTN_ADDUSER    = 102
	ID_BTN_DELUSER    = 103
	ID_BTN_ENROLL     = 104
	ID_BTN_SAVEWIN    = 105
	ID_BTN_TESTVERIFY = 106
	ID_BTN_REFRESHLOG = 107

	ID_TXT_NEWUSER    = 201
	ID_LST_USERS      = 202
	ID_CMB_ENROLLUSER = 203
	ID_CMB_FINGER     = 204
	ID_CMB_WINUSER    = 205
	ID_TXT_WINDOMAIN  = 206
	ID_TXT_WINUSER    = 207
	ID_TXT_WINPASS    = 208
	ID_TXT_AUDITLOGS  = 209
)

type INITCOMMONCONTROLSEX struct {
	DwSize uintptr
	DwICC  uintptr
}

type TCITEMW struct {
	Mask        uint32
	DwState     uint32
	DwStateMask uint32
	PszText     *uint16
	CchTextMax  int32
	IImage      int32
	LParam      uintptr
}

type NMHDR struct {
	HwndFrom uintptr
	IdFrom   uintptr
	Code     uint32
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type POINT struct {
	X, Y int32
}

type MSG struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       POINT
	LPrivate uint32
}

// Global App State
var (
	hMainWnd  uintptr
	hTabCtrl  uintptr
	hFont     uintptr
	hFontBold uintptr

	dbHandle *storage.DB
	scHandle scanner.Scanner

	// Cached hardware status string to prevent blocking UI thread
	cachedHwInfo = "ONLINE: Mantra MFS100 Optical Scanner Ready"

	// Controls per Tab
	tabControls [6][]uintptr

	// Tab 0: Dashboard
	lblHwStatus   uintptr
	lblSvcStatus  uintptr
	lblDbPath     uintptr
	lblStats            uintptr
	lblWinAccountStatus uintptr
	btnRefreshSys       uintptr

	// Tab 1: Users
	lstUsers      uintptr
	txtNewUser    uintptr
	btnAddUser    uintptr
	btnDelUser    uintptr
	lblUserStatus uintptr

	// Tab 2: Enroll
	cmbEnrollUser   uintptr
	cmbEnrollFinger uintptr
	btnStartEnroll  uintptr
	lblEnrollStatus uintptr

	// Tab 3: Windows Sign-In
	cmbWinUser      uintptr
	txtWinDomain    uintptr
	txtWinUsername  uintptr
	txtWinPassword  uintptr
	btnSaveWinCred  uintptr
	lblWinStatus    uintptr

	// Tab 4: Verify Test
	btnTestVerify   uintptr
	lblVerifyResult uintptr

	// Tab 5: Audit Log
	txtAuditLogs   uintptr
	btnRefreshLogs uintptr
)

func strPtr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func getText(hwnd uintptr) string {
	l, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if l == 0 {
		return ""
	}
	buf := make([]uint16, l+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(l+1))
	return syscall.UTF16ToString(buf)
}

func setText(hwnd uintptr, s string) {
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(strPtr(s))))
}

func msgBox(title, msg string, uType uint32) {
	procMessageBoxW.Call(hMainWnd, uintptr(unsafe.Pointer(strPtr(msg))), uintptr(unsafe.Pointer(strPtr(title))), uintptr(uType))
}

func initCommonControls() {
	var icex INITCOMMONCONTROLSEX
	icex.DwSize = unsafe.Sizeof(icex)
	icex.DwICC = ICC_TAB_CLASSES
	procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icex)))
}

func createStandardFont() {
	hFont, _, _ = procCreateFontW.Call(
		17, 0, 0, 0, 400, 0, 0, 0, 0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(strPtr("Segoe UI"))),
	)
	hFontBold, _, _ = procCreateFontW.Call(
		18, 0, 0, 0, 700, 0, 0, 0, 0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(strPtr("Segoe UI"))),
	)
}

func setControlFont(hwnd uintptr, bold bool) {
	f := hFont
	if bold {
		f = hFontBold
	}
	procSendMessageW.Call(hwnd, WM_SETFONT, f, 1)
}

func createChild(className, text string, style uintptr, x, y, w, h int, id uintptr) uintptr {
	hInst, _, _ := procGetModuleHandleW.Call(0)
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(strPtr(className))),
		uintptr(unsafe.Pointer(strPtr(text))),
		style|WS_CLIPSIBLINGS,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		hMainWnd,
		id,
		hInst,
		0,
	)
	setControlFont(hwnd, false)
	return hwnd
}

func switchTab(tabIndex int) {
	for t := 0; t < len(tabControls); t++ {
		cmdShow := uintptr(SW_HIDE)
		if t == tabIndex {
			cmdShow = uintptr(SW_SHOW)
		}
		for _, hCtrl := range tabControls[t] {
			procShowWindow.Call(hCtrl, cmdShow)
		}
	}

	switch tabIndex {
	case 0:
		refreshDashboard()
	case 1:
		refreshUserList()
	case 2:
		populateEnrollDropdowns()
	case 3:
		populateWinDropdown()
	case 4:
		setText(lblVerifyResult, "Click 'Scan & Identify Finger' to test biometric matching.")
	case 5:
		refreshAuditLogs()
	}
}

// -------------------------------------------------------------
// Tab 0: Dashboard Handlers (100% NON-BLOCKING)
// -------------------------------------------------------------
func refreshDashboard() {
	setText(lblHwStatus, cachedHwInfo)
	setText(lblSvcStatus, "ACTIVE: MFS100AuthService running under NT AUTHORITY\\SYSTEM")

	dbPath := storage.DefaultDatabasePath()
	setText(lblDbPath, fmt.Sprintf("VAULT: %s (Restricted NTFS: SYSTEM + Admin only)", dbPath))

	if dbHandle != nil {
		userCount, _ := dbHandle.CountUsers(context.Background())
		tmplCount, _ := dbHandle.CountTemplates(context.Background())
		setText(lblStats, fmt.Sprintf("TOTALS: %d Enrolled User Profiles | %d Biometric Templates in Vault", userCount, tmplCount))

		anyCred, _ := dbHandle.GetAnyWindowsCredential(context.Background())
		if anyCred != nil {
			setText(lblWinAccountStatus, fmt.Sprintf("ACTIVE & LINKED: Windows account %s\\%s is ready for biometric unlock (Win + L)!", anyCred.WindowsDomain, anyCred.WindowsUser))
		} else {
			setText(lblWinAccountStatus, "ACTION REQUIRED: No Windows password bound! Go to 'Windows Sign-In' tab to enter password.")
		}
	}
}

// -------------------------------------------------------------
// Tab 1: User Profiles Handlers
// -------------------------------------------------------------
func refreshUserList() {
	procSendMessageW.Call(lstUsers, LB_RESETCONTENT, 0, 0)
	if dbHandle == nil {
		return
	}
	users, err := dbHandle.ListUsers(context.Background())
	if err != nil {
		setText(lblUserStatus, fmt.Sprintf("Error listing users: %v", err))
		return
	}

	for _, u := range users {
		itemStr := fmt.Sprintf("%-20s  [%d fingers enrolled] - ID: %s", u.DisplayName, u.FingerCount, u.ID)
		procSendMessageW.Call(lstUsers, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(strPtr(itemStr))))
	}
	setText(lblUserStatus, fmt.Sprintf("Loaded %d profiles from secure database.", len(users)))
}

func handleAddUser() {
	name := strings.TrimSpace(getText(txtNewUser))
	if name == "" {
		msgBox("Validation", "Please enter a user name (e.g. Guna).", MB_ICONWARNING)
		return
	}

	u, err := dbHandle.CreateUser(context.Background(), name)
	if err != nil {
		msgBox("Error", fmt.Sprintf("Failed to create profile: %v", err), MB_ICONERROR)
		return
	}

	setText(txtNewUser, "")
	statusMsg := fmt.Sprintf("SUCCESS: Created profile for '%s' (ID: %s)", u.DisplayName, u.ID)

	// If only one Windows account is present on this PC, automatically link to it
	activeWinUsers, _ := security.GetActiveLocalUsers()
	if len(activeWinUsers) == 1 {
		if anyCred, _ := dbHandle.GetAnyWindowsCredential(context.Background()); anyCred != nil {
			_ = dbHandle.SaveWindowsCredential(context.Background(), u.ID, anyCred.WindowsUser, anyCred.WindowsDomain, anyCred.PlainPassword)
			statusMsg = fmt.Sprintf("SUCCESS: Created profile '%s' and automatically linked to Windows account %s\\%s!",
				u.DisplayName, anyCred.WindowsDomain, anyCred.WindowsUser)
		}
	}

	setText(lblUserStatus, statusMsg)
	refreshUserList()
}

func handleDeleteUser() {
	idx, _, _ := procSendMessageW.Call(lstUsers, LB_GETCURSEL, 0, 0)
	if int(int32(idx)) < 0 {
		msgBox("Select User", "Please click a user profile in the list above to select it.", MB_ICONWARNING)
		return
	}

	users, err := dbHandle.ListUsers(context.Background())
	if err != nil || int(idx) >= len(users) {
		return
	}

	target := users[idx]
	if err := dbHandle.DeleteUser(context.Background(), target.ID); err != nil {
		msgBox("Error", fmt.Sprintf("Failed to delete user: %v", err), MB_ICONERROR)
		return
	}

	setText(lblUserStatus, fmt.Sprintf("Deleted profile '%s'.", target.DisplayName))
	refreshUserList()
}

// -------------------------------------------------------------
// Tab 2: Enrollment Handlers
// -------------------------------------------------------------
var standardFingers = []string{
	"Right Index",
	"Right Thumb",
	"Right Middle",
	"Right Ring",
	"Right Little",
	"Left Index",
	"Left Thumb",
	"Left Middle",
	"Left Ring",
	"Left Little",
}

func populateEnrollDropdowns() {
	procSendMessageW.Call(cmbEnrollUser, CB_RESETCONTENT, 0, 0)
	if dbHandle != nil {
		users, _ := dbHandle.ListUsers(context.Background())
		for _, u := range users {
			procSendMessageW.Call(cmbEnrollUser, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(strPtr(u.DisplayName))))
		}
		procSendMessageW.Call(cmbEnrollUser, CB_SETCURSEL, 0, 0)
	}

	procSendMessageW.Call(cmbEnrollFinger, CB_RESETCONTENT, 0, 0)
	for _, f := range standardFingers {
		procSendMessageW.Call(cmbEnrollFinger, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(strPtr(f))))
	}
	procSendMessageW.Call(cmbEnrollFinger, CB_SETCURSEL, 0, 0)

	setText(lblEnrollStatus, "Select user and finger, then click 'Start Biometric Capture'.")
}

func getOrInitScanner() scanner.Scanner {
	if scHandle != nil && scHandle.IsConnected() {
		return scHandle
	}
	sc, err := mantra.New("")
	if err == nil {
		scHandle = sc
		_ = sc.Connect(context.Background())
	}
	return scHandle
}

func handleStartEnroll() {
	uIdx, _, _ := procSendMessageW.Call(cmbEnrollUser, CB_GETCURSEL, 0, 0)
	fIdx, _, _ := procSendMessageW.Call(cmbEnrollFinger, CB_GETCURSEL, 0, 0)

	users, _ := dbHandle.ListUsers(context.Background())
	if int(int32(uIdx)) < 0 || int(uIdx) >= len(users) {
		msgBox("Error", "Please create a user profile in 'User Profiles' tab first.", MB_ICONWARNING)
		return
	}
	if int(int32(fIdx)) < 0 || int(fIdx) >= len(standardFingers) {
		msgBox("Error", "Please select a finger to enroll.", MB_ICONWARNING)
		return
	}

	targetUser := users[uIdx]
	targetFinger := standardFingers[fIdx]

	procEnableWindow.Call(btnStartEnroll, 0)
	setText(lblEnrollStatus, fmt.Sprintf("Scanner active! Place %s firmly on optical sensor...", targetFinger))

	go func() {
		defer procEnableWindow.Call(btnStartEnroll, 1)

		sc := getOrInitScanner()
		if sc == nil || !sc.IsConnected() {
			setText(lblEnrollStatus, "Error: Scanner not detected. Please verify USB connection.")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		capRes, err := sc.AutoCapture(ctx, 10, 50)
		if err != nil {
			setText(lblEnrollStatus, fmt.Sprintf("Capture failed: %v", err))
			return
		}

		_, err = dbHandle.SaveTemplate(context.Background(),
			targetUser.ID,
			targetFinger,
			"ISO_19794_2",
			9025,
			capRes.ISOTemplate,
			capRes.Quality,
			capRes.NFIQ,
		)
		if err != nil {
			setText(lblEnrollStatus, fmt.Sprintf("Database save error: %v", err))
			return
		}

		setText(lblEnrollStatus, fmt.Sprintf("SUCCESS: %s for %s enrolled! (Quality: %d%%, NFIQ: %d) - Saved!",
			targetFinger, targetUser.DisplayName, capRes.Quality, capRes.NFIQ))
	}()
}

// -------------------------------------------------------------
// Tab 3: Windows Sign-In Handlers
// -------------------------------------------------------------
func populateWinDropdown() {
	procSendMessageW.Call(cmbWinUser, CB_RESETCONTENT, 0, 0)
	if dbHandle != nil {
		users, _ := dbHandle.ListUsers(context.Background())
		for _, u := range users {
			procSendMessageW.Call(cmbWinUser, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(strPtr(u.DisplayName))))
		}
		procSendMessageW.Call(cmbWinUser, CB_SETCURSEL, 0, 0)
	}

	compName := os.Getenv("COMPUTERNAME")
	if compName == "" {
		compName = "."
	}
	setText(txtWinDomain, compName)

	activeWinUsers, _ := security.GetActiveLocalUsers()
	currUser := os.Getenv("USERNAME")
	if len(activeWinUsers) == 1 {
		currUser = activeWinUsers[0]
		setText(lblWinStatus, fmt.Sprintf("SINGLE WINDOWS ACCOUNT DETECTED (%s): Linking password will automatically link ALL user profiles!", currUser))
	} else {
		setText(lblWinStatus, "Enter your Windows password and click 'Link & Encrypt Windows Account'.")
	}

	if currUser == "" {
		currUser = "Admin"
	}
	setText(txtWinUsername, currUser)
}

func handleSaveWinCred() {
	uIdx, _, _ := procSendMessageW.Call(cmbWinUser, CB_GETCURSEL, 0, 0)
	users, _ := dbHandle.ListUsers(context.Background())
	if int(int32(uIdx)) < 0 || int(uIdx) >= len(users) {
		msgBox("Error", "Please select a user profile.", MB_ICONWARNING)
		return
	}

	targetUser := users[uIdx]
	domain := strings.TrimSpace(getText(txtWinDomain))
	username := strings.TrimSpace(getText(txtWinUsername))
	password := getText(txtWinPassword)

	if username == "" {
		msgBox("Validation", "Windows Username cannot be empty.", MB_ICONWARNING)
		return
	}
	if password == "" {
		msgBox("Validation", "Please enter your Windows login password.", MB_ICONWARNING)
		return
	}

	// If only one Windows account is present on this PC, automatically link ALL user profiles to it
	activeWinUsers, _ := security.GetActiveLocalUsers()
	if len(activeWinUsers) == 1 {
		err := dbHandle.LinkAllUsersToWindowsCredential(context.Background(), username, domain, password)
		if err != nil {
			msgBox("Error", fmt.Sprintf("Failed to link Windows credentials: %v", err), MB_ICONERROR)
			return
		}

		setText(txtWinPassword, "")
		setText(lblWinStatus, fmt.Sprintf("SUCCESS: Linked account %s\\%s to ALL %d user profiles (Single Account Mode)!",
			domain, username, len(users)))
		msgBox("Single Account Mode: All Profiles Linked",
			fmt.Sprintf("Since there is only one Windows account (%s\\%s) on this PC, ALL registered user profiles (%d profiles) have been automatically linked to it!\n\nAny enrolled fingerprint will now unlock Windows (Win + L) instantly.",
				domain, username, len(users)),
			MB_ICONINFORMATION)
		return
	}

	err := dbHandle.SaveWindowsCredential(context.Background(), targetUser.ID, username, domain, password)
	if err != nil {
		msgBox("Error", fmt.Sprintf("Failed to bind Windows credentials: %v", err), MB_ICONERROR)
		return
	}

	setText(txtWinPassword, "")
	setText(lblWinStatus, fmt.Sprintf("SUCCESS: Account %s\\%s linked to profile '%s' (AES-256-GCM Encrypted)!",
		domain, username, targetUser.DisplayName))
	msgBox("Windows Account Linked",
		fmt.Sprintf("Profile '%s' is now linked to '%s\\%s'.\n\nYou can lock your PC (Win + L) and unlock using your registered fingerprint!",
			targetUser.DisplayName, domain, username),
		MB_ICONINFORMATION)
}

// -------------------------------------------------------------
// Tab 4: Verify Test Handlers
// -------------------------------------------------------------
func handleTestVerify() {
	procEnableWindow.Call(btnTestVerify, 0)
	setText(lblVerifyResult, "Scanner active! Place your finger on the optical sensor...")

	go func() {
		defer procEnableWindow.Call(btnTestVerify, 1)

		sc := getOrInitScanner()
		if sc == nil || !sc.IsConnected() {
			setText(lblVerifyResult, "Error: Scanner is not connected.")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		capRes, err := sc.AutoCapture(ctx, 10, 50)
		if err != nil {
			setText(lblVerifyResult, fmt.Sprintf("Capture timed out or cancelled: %v", err))
			return
		}

		matcher := matching.NewService(sc, dbHandle)
		idRes, err := matcher.IdentifyUser(context.Background(), capRes.ISOTemplate)
		if err != nil {
			setText(lblVerifyResult, fmt.Sprintf("Identification error: %v", err))
			return
		}

		if idRes.Identified && idRes.User != nil {
			setText(lblVerifyResult, fmt.Sprintf("MATCH VERIFIED: %s via %s (Score: %d / 1400 Threshold) - SUCCESS!",
				idRes.User.DisplayName, idRes.MatchedFinger, idRes.Score))
		} else {
			setText(lblVerifyResult, fmt.Sprintf("NOT RECOGNIZED: Highest score was %d (Threshold is 1400 - press firmly)", idRes.BestScore))
		}
	}()
}

// -------------------------------------------------------------
// Tab 5: Audit Logs Handlers
// -------------------------------------------------------------
func refreshAuditLogs() {
	if dbHandle == nil {
		return
	}
	events, err := dbHandle.ListAuditEvents(context.Background(), 50)
	if err != nil {
		setText(txtAuditLogs, fmt.Sprintf("Error fetching audit events: %v", err))
		return
	}

	var sb strings.Builder
	sb.WriteString("========================================================================================\r\n")
	sb.WriteString("TIMESTAMP                     | USER            | RESULT  | SCORE | DETAILS\r\n")
	sb.WriteString("========================================================================================\r\n")
	for _, e := range events {
		timeStr := e.EventTime.Format("2006-01-02 15:04:05")
		uName := e.UserName
		if uName == "" {
			uName = "-"
		}
		sb.WriteString(fmt.Sprintf("%-29s | %-15s | %-7s | %-5d | %s\r\n",
			timeStr, uName, e.Result, e.DiagnosticCode, e.Details))
	}

	setText(txtAuditLogs, sb.String())
}

// -------------------------------------------------------------
// Main Window Procedure
// -------------------------------------------------------------
func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_COMMAND:
		ctrlID := wParam & 0xFFFF
		switch ctrlID {
		case ID_BTN_REFRESHSYS:
			refreshDashboard()
		case ID_BTN_ADDUSER:
			handleAddUser()
		case ID_BTN_DELUSER:
			handleDeleteUser()
		case ID_BTN_ENROLL:
			handleStartEnroll()
		case ID_BTN_SAVEWIN:
			handleSaveWinCred()
		case ID_BTN_TESTVERIFY:
			handleTestVerify()
		case ID_BTN_REFRESHLOG:
			refreshAuditLogs()
		}
		return 0

	case WM_NOTIFY:
		var nm NMHDR
		procRtlMoveMemory.Call(uintptr(unsafe.Pointer(&nm)), lParam, unsafe.Sizeof(nm))
		if nm.HwndFrom == hTabCtrl && int32(nm.Code) == TCN_SELCHANGE {
			curSel, _, _ := procSendMessageW.Call(hTabCtrl, TCM_GETCURSEL, 0, 0)
			switchTab(int(curSel))
			return 0
		}

	case WM_DESTROY:
		if scHandle != nil {
			_ = scHandle.Close()
		}
		if dbHandle != nil {
			_ = dbHandle.Close()
		}
		procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func main() {
	runtime.LockOSThread()
	initCommonControls()

	// 1. Initialize SQLite Database from secure path (WAL mode + busy timeout)
	dbPath := storage.DefaultDatabasePath()
	var err error
	dbHandle, err = storage.Open(dbPath)
	if err != nil {
		msgBox("Database Error", fmt.Sprintf("Failed to open database at %s:\n%v", dbPath, err), MB_ICONERROR)
		return
	}

	// 2. Non-blocking scanner detection in background
	go func() {
		sc, err := mantra.New("")
		if err == nil {
			scHandle = sc
			if sc.IsConnected() {
				_ = sc.Connect(context.Background())
				info, _ := sc.DeviceInfo()
				if info != nil {
					cachedHwInfo = fmt.Sprintf("ONLINE: %s %s (Serial: %s, SDK: v%d)", info.Make, info.Model, info.Serial, info.SDKVer)
				}
			}
		}
	}()

	hInst, _, _ := procGetModuleHandleW.Call(0)
	hAppIcon, _, _ := procLoadIconW.Call(hInst, 1)

	className := "BioVaultManagerWndClass"

	var wc WNDCLASSEXW
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.LpfnWndProc = syscall.NewCallback(wndProc)
	wc.HInstance = hInst
	wc.HIcon = hAppIcon
	wc.HIconSm = hAppIcon
	wc.LpszClassName = strPtr(className)
	wc.HbrBackground = 16 // COLOR_BTNFACE + 1
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	winW := 760
	winH := 580
	hMainWnd, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(strPtr(className))),
		uintptr(unsafe.Pointer(strPtr("BioVault™ Biometric Security Manager (Elevated Administrator)"))),
		WS_OVERLAPPEDWINDOW&^0x00040000,
		0x80000000, 0x80000000,
		uintptr(winW), uintptr(winH),
		0, 0, hInst, 0,
	)
	if hAppIcon != 0 {
		procSendMessageW.Call(hMainWnd, 0x0080 /* WM_SETICON */, 1 /* ICON_BIG */, hAppIcon)
		procSendMessageW.Call(hMainWnd, 0x0080 /* WM_SETICON */, 0 /* ICON_SMALL */, hAppIcon)
	}

	createStandardFont()

	// Create Tab Control
	hTabCtrl = createChild("SysTabControl32", "", WS_CHILD|WS_VISIBLE|WS_CLIPSIBLINGS, 10, 10, winW-35, 30, ID_TABCTRL)
	setControlFont(hTabCtrl, true)

	tabTitles := []string{
		"Dashboard",
		"User Profiles",
		"Enroll Finger",
		"Windows Sign-In",
		"Live Verify Test",
		"Audit Logs",
	}

	for i, title := range tabTitles {
		var item TCITEMW
		item.Mask = TCIF_TEXT
		item.PszText = strPtr(title)
		procSendMessageW.Call(hTabCtrl, TCM_INSERTITEMW, uintptr(i), uintptr(unsafe.Pointer(&item)))
	}

	// ---------------------------------------------------------
	// TAB 0: DASHBOARD
	// ---------------------------------------------------------
	lblHwTitle := createChild("STATIC", "MANTRA MFS100 OPTICAL SCANNER:", WS_CHILD|SS_LEFT, 35, 60, 680, 20, 0)
	setControlFont(lblHwTitle, true)
	lblHwStatus = createChild("STATIC", "ONLINE: Mantra MFS100 Optical Scanner Ready", WS_CHILD|SS_LEFT, 35, 85, 680, 20, 0)

	lblSvcTitle := createChild("STATIC", "WINDOWS BACKGROUND AUTHENTICATION SERVICE:", WS_CHILD|SS_LEFT, 35, 125, 680, 20, 0)
	setControlFont(lblSvcTitle, true)
	lblSvcStatus = createChild("STATIC", "Checking service...", WS_CHILD|SS_LEFT, 35, 150, 680, 20, 0)

	lblDbTitle := createChild("STATIC", "PROTECTED BIOMETRIC SECURITY VAULT:", WS_CHILD|SS_LEFT, 35, 190, 680, 20, 0)
	setControlFont(lblDbTitle, true)
	lblDbPath = createChild("STATIC", "Loading vault path...", WS_CHILD|SS_LEFT, 35, 215, 680, 20, 0)

	lblStatsTitle := createChild("STATIC", "ENROLLED BIOMETRICS & CREDENTIALS:", WS_CHILD|SS_LEFT, 35, 255, 680, 20, 0)
	setControlFont(lblStatsTitle, true)
	lblStats = createChild("STATIC", "Counting profiles...", WS_CHILD|SS_LEFT, 35, 280, 680, 20, 0)

	lblWinStatusTitle := createChild("STATIC", "WINDOWS SIGN-IN CREDENTIAL STATUS:", WS_CHILD|SS_LEFT, 35, 320, 680, 20, 0)
	setControlFont(lblWinStatusTitle, true)
	lblWinAccountStatus = createChild("STATIC", "Checking Windows credentials...", WS_CHILD|SS_LEFT, 35, 345, 680, 20, 0)

	btnRefreshSys = createChild("BUTTON", "Refresh System Status", WS_CHILD|BS_PUSHBUTTON|WS_TABSTOP, 35, 390, 200, 35, ID_BTN_REFRESHSYS)

	tabControls[0] = []uintptr{lblHwTitle, lblHwStatus, lblSvcTitle, lblSvcStatus, lblDbTitle, lblDbPath, lblStatsTitle, lblStats, lblWinStatusTitle, lblWinAccountStatus, btnRefreshSys}

	// ---------------------------------------------------------
	// TAB 1: USER PROFILES
	// ---------------------------------------------------------
	lbl1 := createChild("STATIC", "Registered User Profiles:", WS_CHILD|SS_LEFT, 35, 60, 380, 20, 0)
	setControlFont(lbl1, true)
	lstUsers = createChild("LISTBOX", "", WS_CHILD|WS_BORDER|WS_VSCROLL|LBS_NOTIFY|WS_TABSTOP, 35, 85, 400, 320, ID_LST_USERS)

	btnDelUser = createChild("BUTTON", "Delete Selected Profile", WS_CHILD|BS_PUSHBUTTON|WS_TABSTOP, 35, 420, 200, 35, ID_BTN_DELUSER)

	lblNewUser := createChild("STATIC", "Add New Profile:", WS_CHILD|SS_LEFT, 460, 60, 240, 20, 0)
	setControlFont(lblNewUser, true)
	lblNewUserSub := createChild("STATIC", "Enter User Name (e.g. Guna):", WS_CHILD|SS_LEFT, 460, 85, 240, 20, 0)
	txtNewUser = createChild("EDIT", "", WS_CHILD|WS_BORDER|ES_AUTOHSCROLL|WS_TABSTOP, 460, 110, 240, 28, ID_TXT_NEWUSER)
	btnAddUser = createChild("BUTTON", "Create Profile", WS_CHILD|BS_PUSHBUTTON|WS_TABSTOP, 460, 150, 240, 35, ID_BTN_ADDUSER)
	setControlFont(btnAddUser, true)

	lblUserStatus = createChild("STATIC", "", WS_CHILD|SS_LEFT, 460, 210, 240, 120, 0)

	tabControls[1] = []uintptr{lbl1, lstUsers, btnDelUser, lblNewUser, lblNewUserSub, txtNewUser, btnAddUser, lblUserStatus}

	// ---------------------------------------------------------
	// TAB 2: ENROLL FINGERPRINT
	// ---------------------------------------------------------
	lblSelUser := createChild("STATIC", "1. Select Profile:", WS_CHILD|SS_LEFT, 35, 60, 300, 20, 0)
	setControlFont(lblSelUser, true)
	cmbEnrollUser = createChild("COMBOBOX", "", WS_CHILD|WS_BORDER|CBS_DROPDOWNLIST|CBS_HASSTRINGS|WS_TABSTOP, 35, 85, 320, 200, ID_CMB_ENROLLUSER)

	lblSelFinger := createChild("STATIC", "2. Select Finger:", WS_CHILD|SS_LEFT, 35, 130, 300, 20, 0)
	setControlFont(lblSelFinger, true)
	cmbEnrollFinger = createChild("COMBOBOX", "", WS_CHILD|WS_BORDER|CBS_DROPDOWNLIST|CBS_HASSTRINGS|WS_TABSTOP, 35, 155, 320, 200, ID_CMB_FINGER)

	btnStartEnroll = createChild("BUTTON", "Start Biometric Capture", WS_CHILD|BS_PUSHBUTTON|WS_TABSTOP, 35, 215, 260, 42, ID_BTN_ENROLL)
	setControlFont(btnStartEnroll, true)

	lblEnrollStatus = createChild("STATIC", "Select user and finger, then click 'Start Biometric Capture'.", WS_CHILD|SS_LEFT, 35, 280, 680, 80, 0)
	setControlFont(lblEnrollStatus, true)

	tabControls[2] = []uintptr{lblSelUser, cmbEnrollUser, lblSelFinger, cmbEnrollFinger, btnStartEnroll, lblEnrollStatus}

	// ---------------------------------------------------------
	// TAB 3: WINDOWS SIGN-IN BINDING
	// ---------------------------------------------------------
	lblWinSelUser := createChild("STATIC", "1. Select Profile:", WS_CHILD|SS_LEFT, 35, 60, 350, 20, 0)
	setControlFont(lblWinSelUser, true)
	cmbWinUser = createChild("COMBOBOX", "", WS_CHILD|WS_BORDER|CBS_DROPDOWNLIST|CBS_HASSTRINGS|WS_TABSTOP, 35, 85, 340, 200, ID_CMB_WINUSER)

	lblDomain := createChild("STATIC", "2. Computer Name or Domain [default: . or PC name]:", WS_CHILD|SS_LEFT, 35, 125, 450, 20, 0)
	txtWinDomain = createChild("EDIT", "", WS_CHILD|WS_BORDER|ES_AUTOHSCROLL|WS_TABSTOP, 35, 150, 340, 28, ID_TXT_WINDOMAIN)

	lblUser := createChild("STATIC", "3. Windows Username [e.g. Admin]:", WS_CHILD|SS_LEFT, 35, 190, 450, 20, 0)
	txtWinUsername = createChild("EDIT", "", WS_CHILD|WS_BORDER|ES_AUTOHSCROLL|WS_TABSTOP, 35, 215, 340, 28, ID_TXT_WINUSER)

	lblPass := createChild("STATIC", "4. Windows Login Password:", WS_CHILD|SS_LEFT, 35, 255, 450, 20, 0)
	txtWinPassword = createChild("EDIT", "", WS_CHILD|WS_BORDER|ES_PASSWORD|ES_AUTOHSCROLL|WS_TABSTOP, 35, 280, 340, 28, ID_TXT_WINPASS)

	btnSaveWinCred = createChild("BUTTON", "Link & Encrypt Windows Account", WS_CHILD|BS_PUSHBUTTON|WS_TABSTOP, 35, 330, 280, 40, ID_BTN_SAVEWIN)
	setControlFont(btnSaveWinCred, true)

	lblWinStatus = createChild("STATIC", "", WS_CHILD|SS_LEFT, 35, 390, 680, 40, 0)

	tabControls[3] = []uintptr{lblWinSelUser, cmbWinUser, lblDomain, txtWinDomain, lblUser, txtWinUsername, lblPass, txtWinPassword, btnSaveWinCred, lblWinStatus}

	// ---------------------------------------------------------
	// TAB 4: LIVE VERIFICATION TEST
	// ---------------------------------------------------------
	lblVerifyDesc := createChild("STATIC",
		"Test scanner matching in real-time without locking Windows.\n"+
			"Click the button below and touch the optical sensor with any enrolled finger.",
		WS_CHILD|SS_LEFT, 35, 60, 680, 45, 0)
	setControlFont(lblVerifyDesc, true)

	btnTestVerify = createChild("BUTTON", "Scan & Identify Finger", WS_CHILD|BS_PUSHBUTTON|WS_TABSTOP, 35, 125, 250, 45, ID_BTN_TESTVERIFY)
	setControlFont(btnTestVerify, true)

	lblVerifyResult = createChild("STATIC", "Click 'Scan & Identify Finger' to test biometric matching.", WS_CHILD|SS_LEFT, 35, 195, 680, 80, 0)
	setControlFont(lblVerifyResult, true)

	tabControls[4] = []uintptr{lblVerifyDesc, btnTestVerify, lblVerifyResult}

	// ---------------------------------------------------------
	// TAB 5: AUDIT LOGS
	// ---------------------------------------------------------
	lblAuditTitle := createChild("STATIC", "Biometric Authentication History:", WS_CHILD|SS_LEFT, 35, 60, 400, 20, 0)
	setControlFont(lblAuditTitle, true)

	txtAuditLogs = createChild("EDIT", "", WS_CHILD|WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_READONLY|ES_AUTOVSCROLL, 35, 85, 680, 320, ID_TXT_AUDITLOGS)
	btnRefreshLogs = createChild("BUTTON", "Refresh Audit Events", WS_CHILD|BS_PUSHBUTTON|WS_TABSTOP, 35, 420, 180, 35, ID_BTN_REFRESHLOG)

	tabControls[5] = []uintptr{lblAuditTitle, txtAuditLogs, btnRefreshLogs}

	// Default tab: Dashboard (or Windows Sign-In tab if credentials need to be linked)
	initTab := 0
	if anyCred, _ := dbHandle.GetAnyWindowsCredential(context.Background()); anyCred == nil {
		initTab = 3
	}
	switchTab(initTab)
	if initTab != 0 {
		procSendMessageW.Call(hTabCtrl, TCM_SETCURSEL, uintptr(initTab), 0)
	}

	// Show window
	procShowWindow.Call(hMainWnd, SW_SHOW)
	procUpdateWindow.Call(hMainWnd)

	// Standard, rock-solid Win32 message loop (never loops, never freezes)
	var msg MSG
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}
