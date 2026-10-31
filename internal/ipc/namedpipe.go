package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

const (
	PipeName = `\\.\pipe\MFS100AuthPipe`

	pipeAccessDuplex = 0x00000003
	pipeTypeMessage  = 0x00000004
	pipeReadMessage  = 0x00000002
	pipeWait         = 0x00000000
	pipeUnlimited    = 255
	invalidHandle    = ^uintptr(0)
)

var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procCreateNamedPipeW   = kernel32.NewProc("CreateNamedPipeW")
	procConnectNamedPipe   = kernel32.NewProc("ConnectNamedPipe")
	procDisconnectNamedPipe = kernel32.NewProc("DisconnectNamedPipe")
	procReadFile           = kernel32.NewProc("ReadFile")
	procWriteFile          = kernel32.NewProc("WriteFile")
	procCloseHandle        = kernel32.NewProc("CloseHandle")
)

type Request struct {
	Command string `json:"command"` // "TRIGGER_UNLOCK", "STATUS", "CANCEL"
	User    string `json:"user,omitempty"`
}

type Response struct {
	Status   string `json:"status"` // "SUCCESS", "FAIL", "CANCELLED"
	Message  string `json:"message,omitempty"`
	User     string `json:"user,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Password string `json:"password,omitempty"`
	Score    int    `json:"score,omitempty"`
}

type PipeHandler func(req Request) Response

type Server struct {
	handler PipeHandler
	mu      sync.Mutex
	stop    chan struct{}
}

func NewServer(handler PipeHandler) *Server {
	return &Server{
		handler: handler,
		stop:    make(chan struct{}),
	}
}

// Start begins listening on \\.\pipe\MFS100AuthPipe in a background goroutine.
func (s *Server) Start(ctx context.Context) error {
	go func() {
		pipeNameW, _ := syscall.UTF16PtrFromString(PipeName)

		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stop:
				return
			default:
			}

			// Create instance of named pipe
			handle, _, err := procCreateNamedPipeW.Call(
				uintptr(unsafe.Pointer(pipeNameW)),
				pipeAccessDuplex,
				pipeTypeMessage|pipeReadMessage|pipeWait,
				pipeUnlimited,
				4096, // Out buffer
				4096, // In buffer
				5000, // Timeout ms
				0,    // Default security descriptor (accessible to LocalSystem / Winlogon / current user)
			)

			if handle == invalidHandle {
				fmt.Printf("[-] CreateNamedPipe failed: %v\n", err)
				return
			}

			// Wait for client (LogonUI / Credential Provider) to connect
			connected, _, _ := procConnectNamedPipe.Call(handle, 0)
			if connected == 0 {
				lastErr := syscall.GetLastError()
				if lastErr != syscall.Errno(535) { // ERROR_PIPE_CONNECTED = 535
					procCloseHandle.Call(handle)
					continue
				}
			}

			// Handle client connection
			s.handleClient(handle)
			procDisconnectNamedPipe.Call(handle)
			procCloseHandle.Call(handle)
		}
	}()

	return nil
}

func (s *Server) handleClient(handle uintptr) {
	buf := make([]byte, 4096)
	var bytesRead uint32

	r, _, _ := procReadFile.Call(
		handle,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&bytesRead)),
		0,
	)

	if r == 0 || bytesRead == 0 {
		return
	}

	var req Request
	if err := json.Unmarshal(buf[:bytesRead], &req); err != nil {
		req.Command = "STATUS"
	}

	resp := s.handler(req)
	respBytes, _ := json.Marshal(resp)

	var bytesWritten uint32
	procWriteFile.Call(
		handle,
		uintptr(unsafe.Pointer(&respBytes[0])),
		uintptr(len(respBytes)),
		uintptr(unsafe.Pointer(&bytesWritten)),
		0,
	)
}

// Stop shuts down the pipe listener.
func (s *Server) Stop() {
	close(s.stop)
}

// ClientSend sends a request to \\.\pipe\MFS100AuthPipe and reads the response.
func ClientSend(req Request) (*Response, error) {
	pipeNameW, _ := syscall.UTF16PtrFromString(PipeName)

	handle, err := syscall.CreateFile(
		pipeNameW,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0,
		nil,
		syscall.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open pipe: %w", err)
	}
	defer syscall.CloseHandle(handle)

	reqBytes, _ := json.Marshal(req)
	var written uint32
	err = syscall.WriteFile(handle, reqBytes, &written, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to write to pipe: %w", err)
	}

	buf := make([]byte, 4096)
	var bytesRead uint32
	err = syscall.ReadFile(handle, buf, &bytesRead, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read from pipe: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(buf[:bytesRead], &resp); err != nil {
		return nil, errors.New("invalid JSON response from pipe")
	}

	return &resp, nil
}
