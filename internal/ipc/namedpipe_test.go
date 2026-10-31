package ipc

import (
	"context"
	"testing"
	"time"
)

func TestNamedPipeRoundTrip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := func(req Request) Response {
		if req.Command == "PING" {
			return Response{Status: "PONG", Message: "All good"}
		}
		return Response{Status: "FAIL"}
	}

	server := NewServer(handler)
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Server start failed: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	resp, err := ClientSend(Request{Command: "PING"})
	if err != nil {
		t.Fatalf("ClientSend failed: %v", err)
	}

	if resp.Status != "PONG" {
		t.Fatalf("Expected PONG, got %s", resp.Status)
	}
}
