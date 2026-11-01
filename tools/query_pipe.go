package main

import (
	"fmt"
	"fp-manager/internal/ipc"
)

func main() {
	resp, err := ipc.ClientSend(ipc.Request{Command: "DEBUG"})
	fmt.Printf("DEBUG response: err=%v, resp=%+v\n", err, resp)
}
