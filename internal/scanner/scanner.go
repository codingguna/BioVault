package scanner

import (
	"context"
	"errors"
)

// Common scanner errors.
var (
	ErrDeviceNotConnected = errors.New("MFS100 scanner is not connected")
	ErrCaptureTimeout     = errors.New("fingerprint capture timed out")
	ErrCaptureFailed      = errors.New("fingerprint capture failed")
	ErrPoorQuality        = errors.New("fingerprint quality is too low")
	ErrAlreadyInitialized = errors.New("scanner is already initialized")
	ErrNotInitialized     = errors.New("scanner is not initialized")
)

// DeviceInfo represents hardware details of the connected scanner.
type DeviceInfo struct {
	Serial string `json:"serial"`
	Make   string `json:"make"`
	Model  string `json:"model"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	SDKVer int    `json:"sdk_version"`
}

// CaptureResult represents the outcome of a fingerprint scan.
type CaptureResult struct {
	Quality      int    `json:"quality"`
	NFIQ         int    `json:"nfiq"`
	ISOTemplate  []byte `json:"iso_template"`
	ANSITemplate []byte `json:"ansi_template"`
	BitmapBase64 string `json:"bitmap_base64,omitempty"`
}

// Scanner defines the abstraction layer for fingerprint devices.
type Scanner interface {
	// Connect initializes the scanner and prepares it for scanning.
	Connect(ctx context.Context) error

	// DeviceInfo returns hardware and firmware properties.
	DeviceInfo() (*DeviceInfo, error)

	// IsConnected checks if the physical scanner is currently plugged in.
	IsConnected() bool

	// AutoCapture waits for finger placement, activates LED, checks quality,
	// and returns standard ISO & ANSI templates.
	AutoCapture(ctx context.Context, timeoutSec int, minQuality int) (*CaptureResult, error)

	// MatchISO compares a probe ISO 19794-2 template against an enrolled gallery template.
	// Returns a similarity score (0 - 10000) and whether it exceeds the match threshold.
	MatchISO(probe, gallery []byte) (score int, isMatch bool, err error)

	// MatchANSI compares two ANSI-378 templates.
	MatchANSI(probe, gallery []byte) (score int, isMatch bool, err error)

	// Close safely turns off the sensor LEDs and releases device handles.
	Close() error
}
