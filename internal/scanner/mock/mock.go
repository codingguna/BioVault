package mock

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"fp-manager/internal/scanner"
)

type Scanner struct {
	mu          sync.Mutex
	initialized bool
	connected   bool
}

func New(connected bool) *Scanner {
	return &Scanner{connected: connected}
}

func (m *Scanner) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.connected {
		return scanner.ErrDeviceNotConnected
	}
	m.initialized = true
	return nil
}

func (m *Scanner) DeviceInfo() (*scanner.DeviceInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.initialized {
		return nil, scanner.ErrNotInitialized
	}
	return &scanner.DeviceInfo{
		Serial: "MOCK-MFS100-9999",
		Make:   "MOCK-MANTRA",
		Model:  "MFS100-SIMULATOR",
		Width:  316,
		Height: 354,
		SDKVer: 9025,
	}, nil
}

func (m *Scanner) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *Scanner) AutoCapture(ctx context.Context, timeoutSec int, minQuality int) (*scanner.CaptureResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.initialized {
		return nil, scanner.ErrNotInitialized
	}

	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	iso := make([]byte, 270)
	_, _ = rand.Read(iso)
	ansi := make([]byte, 272)
	_, _ = rand.Read(ansi)

	return &scanner.CaptureResult{
		Quality:      85,
		NFIQ:         1,
		ISOTemplate:  iso,
		ANSITemplate: ansi,
		BitmapBase64: base64.StdEncoding.EncodeToString([]byte("mock-bmp-image-data")),
	}, nil
}

func (m *Scanner) MatchISO(probe, gallery []byte) (int, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.initialized {
		return 0, false, scanner.ErrNotInitialized
	}
	if len(probe) == 0 || len(gallery) == 0 {
		return 0, false, errors.New("empty templates")
	}

	if bytes.Equal(probe, gallery) {
		return 10000, true, nil
	}
	return 250, false, nil
}

func (m *Scanner) MatchANSI(probe, gallery []byte) (int, bool, error) {
	return m.MatchISO(probe, gallery)
}

func (m *Scanner) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initialized = false
	return nil
}
