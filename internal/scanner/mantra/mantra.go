package mantra

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"fp-manager/internal/scanner"
)

const (
	defaultDllDir  = `C:\Program Files\Mantra\MFS100\Driver\MFS100Test`
	defaultDllName = "MFS100Dll.dll"
	matchThreshold = 1400 // Standard Mantra match threshold (score >= 1400 = 90%+ confidence)
)

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	procSetDllDir     = kernel32.NewProc("SetDllDirectoryW")

	// Global dummy callback to prevent MFS100Dll from jumping to null pointer on preview frames.
	defaultCaptureCallback = syscall.NewCallback(func(rawFrame uintptr) uintptr {
		return 0
	})
)

// silenceCRuntime suppresses verbose internal printf statements (e.g. "MFS100AutoCapture Quality = 68")
// emitted directly by the vendor C runtime to stdout/stderr.
func silenceCRuntime(dllDir string) {
	dirW, _ := syscall.UTF16PtrFromString(dllDir)
	procSetDllDir.Call(uintptr(unsafe.Pointer(dirW)))

	hNul, err := syscall.CreateFile(
		syscall.StringToUTF16Ptr("NUL"),
		syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_WRITE,
		nil,
		syscall.OPEN_EXISTING,
		0,
		0,
	)
	if err == nil {
		setStdHandle := kernel32.NewProc("SetStdHandle")
		setStdHandle.Call(uintptr(0xFFFFFFF5), uintptr(hNul)) // STD_OUTPUT_HANDLE = -11
		setStdHandle.Call(uintptr(0xFFFFFFF4), uintptr(hNul)) // STD_ERROR_HANDLE = -12
	}

	for _, crtName := range []string{"msvcr90.dll", "msvcrt.dll"} {
		crt := syscall.NewLazyDLL(crtName)
		freopen := crt.NewProc("freopen")
		iobFunc := crt.NewProc("__iob_func")
		if freopen.Find() == nil && iobFunc.Find() == nil {
			mode, _ := syscall.BytePtrFromString("w")
			nulStr, _ := syscall.BytePtrFromString("NUL")
			iob, _, _ := iobFunc.Call()
			stdoutPtr := iob + 32 // FILE* stdout (index 1)
			stderrPtr := iob + 64 // FILE* stderr (index 2)
			freopen.Call(uintptr(unsafe.Pointer(nulStr)), uintptr(unsafe.Pointer(mode)), stdoutPtr)
			freopen.Call(uintptr(unsafe.Pointer(nulStr)), uintptr(unsafe.Pointer(mode)), stderrPtr)
		}
	}
}

type Scanner struct {
	mu          sync.Mutex
	dllPath     string
	mod         *syscall.LazyDLL
	initialized bool
	info        *scanner.DeviceInfo

	// Procedures
	procInit            *syscall.LazyProc
	procInitWithKey     *syscall.LazyProc
	procUninit          *syscall.LazyProc
	procDeviceConnected *syscall.LazyProc
	procGetSDKVersion   *syscall.LazyProc
	procGetInfo         *syscall.LazyProc
	procGetWidth        *syscall.LazyProc
	procGetHeight       *syscall.LazyProc
	procAutoCapture     *syscall.LazyProc
	procMatchISO        *syscall.LazyProc
	procMatchANSI       *syscall.LazyProc
}

// New creates a new Mantra MFS100 scanner instance.
func New(customDllPath string) (*Scanner, error) {
	dllPath := customDllPath
	if dllPath == "" {
		dllPath = os.Getenv("MFS100_DLL_PATH")
	}
	if dllPath == "" {
		dllPath = filepath.Join(defaultDllDir, defaultDllName)
	}

	if _, err := os.Stat(dllPath); err != nil {
		return nil, fmt.Errorf("MFS100 DLL not found at %s: %w", dllPath, err)
	}

	dllDir := filepath.Dir(dllPath)
	silenceCRuntime(dllDir)

	mod := syscall.NewLazyDLL(dllPath)

	s := &Scanner{
		dllPath:             dllPath,
		mod:                 mod,
		procInit:            mod.NewProc("MFS100Init"),
		procInitWithKey:     mod.NewProc("MFS100InitWithKey"),
		procUninit:          mod.NewProc("MFS100Uninit"),
		procDeviceConnected: mod.NewProc("MFS100DeviceConnected"),
		procGetSDKVersion:   mod.NewProc("MFS100GetSDKVersion"),
		procGetInfo:         mod.NewProc("MFS100GetInfo"),
		procGetWidth:        mod.NewProc("MFS100GetWidth"),
		procGetHeight:       mod.NewProc("MFS100GetHeight"),
		procAutoCapture:     mod.NewProc("MFS100AutoCapture"),
		procMatchISO:        mod.NewProc("MFS100MatchISO"),
		procMatchANSI:       mod.NewProc("MFS100MatchANSI"),
	}

	return s, nil
}

// IsConnected checks whether the MFS100 device is connected over USB.
func (s *Scanner) IsConnected() bool {
	r, _, _ := s.procDeviceConnected.Call()
	return int32(r) == 0
}

// Connect initializes the scanner and caches device information.
func (s *Scanner) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	if !s.IsConnected() {
		return scanner.ErrDeviceNotConnected
	}

	serialBuf := make([]byte, 128)
	rInit, _, err := s.procInit.Call(uintptr(unsafe.Pointer(&serialBuf[0])))
	if int32(rInit) != 0 {
		return fmt.Errorf("MFS100Init failed with error code %d: %w", int32(rInit), err)
	}
	s.initialized = true

	sdkVer, _, _ := s.procGetSDKVersion.Call()

	makeBuf := make([]byte, 64)
	modelBuf := make([]byte, 64)
	s.procGetInfo.Call(uintptr(unsafe.Pointer(&makeBuf[0])), uintptr(unsafe.Pointer(&modelBuf[0])))

	w, _, _ := s.procGetWidth.Call()
	h, _, _ := s.procGetHeight.Call()

	serial := string(bytes.TrimRight(serialBuf, "\x00"))
	makeStr := string(bytes.TrimRight(makeBuf, "\x00"))
	modelStr := string(bytes.TrimRight(modelBuf, "\x00"))

	s.info = &scanner.DeviceInfo{
		Serial: serial,
		Make:   makeStr,
		Model:  modelStr,
		Width:  int(w),
		Height: int(h),
		SDKVer: int(sdkVer),
	}

	return nil
}

// DeviceInfo returns details of the connected device.
func (s *Scanner) DeviceInfo() (*scanner.DeviceInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized || s.info == nil {
		return nil, scanner.ErrNotInitialized
	}
	return s.info, nil
}

// AutoCapture executes a capture session with automatic finger detection.
// It will actively scan and retry for the full timeout duration (at least 5s)
// allowing the user to place and adjust their finger comfortably on the sensor.
func (s *Scanner) AutoCapture(ctx context.Context, timeoutSec int, minQuality int) (*scanner.CaptureResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return nil, scanner.ErrNotInitialized
	}

	if timeoutSec < 5 {
		timeoutSec = 5
	}
	if minQuality <= 0 {
		minQuality = 40
	}

	w := s.info.Width
	h := s.info.Height
	if w <= 0 || h <= 0 {
		w, h = 316, 354
	}

	rawSize := w * h
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		remaining := time.Until(deadline)
		sliceMs := int(remaining.Milliseconds())
		if sliceMs > 2500 {
			sliceMs = 2500
		}
		if sliceMs < 500 {
			sliceMs = 500
		}

		rawFrame := make([]byte, rawSize)
		bmpData := make([]byte, rawSize+2048)
		isoTemplate := make([]byte, 2048)
		ansiTemplate := make([]byte, 2048)

		var (
			isoLen  int32
			ansiLen int32
			quality int32
			nfiq    int32
		)

		ret, _, _ := s.procAutoCapture.Call(
			defaultCaptureCallback,
			uintptr(sliceMs),
			uintptr(unsafe.Pointer(&rawFrame[0])),
			uintptr(unsafe.Pointer(&bmpData[0])),
			uintptr(unsafe.Pointer(&isoTemplate[0])),
			uintptr(unsafe.Pointer(&isoLen)),
			uintptr(unsafe.Pointer(&ansiTemplate[0])),
			uintptr(unsafe.Pointer(&ansiLen)),
			uintptr(unsafe.Pointer(&quality)),
			uintptr(unsafe.Pointer(&nfiq)),
			0, // Detect_Finger = 0: continuously capture without premature abort on light touch
		)

		retCode := int32(ret)
		if retCode == 0 && int(quality) >= minQuality && isoLen > 0 {
			finalISO := make([]byte, isoLen)
			copy(finalISO, isoTemplate[:isoLen])

			finalANSI := make([]byte, ansiLen)
			copy(finalANSI, ansiTemplate[:ansiLen])

			var bmpBase64 string
			if len(bmpData) > 54 && bmpData[0] == 'B' && bmpData[1] == 'M' {
				bmpFileSize := int(bmpData[2]) | int(bmpData[3])<<8 | int(bmpData[4])<<16 | int(bmpData[5])<<24
				if bmpFileSize > 54 && bmpFileSize <= len(bmpData) {
					bmpBase64 = base64.StdEncoding.EncodeToString(bmpData[:bmpFileSize])
				}
			}

			return &scanner.CaptureResult{
				Quality:      int(quality),
				NFIQ:         int(nfiq),
				ISOTemplate:  finalISO,
				ANSITemplate: finalANSI,
				BitmapBase64: bmpBase64,
			}, nil
		}

		// Brief pause between slices while user places/adjusts finger
		time.Sleep(80 * time.Millisecond)
	}

	return nil, scanner.ErrCaptureTimeout
}

// MatchISO matches two ISO 19794-2 templates.
func (s *Scanner) MatchISO(probe, gallery []byte) (int, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return 0, false, scanner.ErrNotInitialized
	}
	if len(probe) == 0 || len(gallery) == 0 {
		return 0, false, errors.New("templates cannot be empty")
	}

	var score int32
	ret, _, err := s.procMatchISO.Call(
		uintptr(unsafe.Pointer(&probe[0])),
		uintptr(unsafe.Pointer(&gallery[0])),
		180, // Max rotation angle
		uintptr(unsafe.Pointer(&score)),
	)

	retCode := int32(ret)
	if retCode != 0 {
		return 0, false, fmt.Errorf("MatchISO failed with code %d: %w", retCode, err)
	}

	isMatch := int(score) >= matchThreshold
	return int(score), isMatch, nil
}

// MatchANSI matches two ANSI-378 templates.
func (s *Scanner) MatchANSI(probe, gallery []byte) (int, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return 0, false, scanner.ErrNotInitialized
	}
	if len(probe) == 0 || len(gallery) == 0 {
		return 0, false, errors.New("templates cannot be empty")
	}

	var score int32
	ret, _, err := s.procMatchANSI.Call(
		uintptr(unsafe.Pointer(&probe[0])),
		uintptr(unsafe.Pointer(&gallery[0])),
		180,
		uintptr(unsafe.Pointer(&score)),
	)

	retCode := int32(ret)
	if retCode != 0 {
		return 0, false, fmt.Errorf("MatchANSI failed with code %d: %w", retCode, err)
	}

	isMatch := int(score) >= matchThreshold
	return int(score), isMatch, nil
}

// Close releases the scanner and unloads the driver connection.
func (s *Scanner) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return nil
	}

	r, _, _ := s.procUninit.Call()
	s.initialized = false
	s.info = nil

	if int32(r) != 0 {
		return fmt.Errorf("MFS100Uninit returned code %d", int32(r))
	}
	return nil
}
