package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"fp-manager/internal/scanner/mantra"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("     MFS100 Hardware Diagnostic & Probe Tool     ")
	fmt.Println("==================================================")

	sc, err := mantra.New("")
	if err != nil {
		fmt.Printf("[-] Failed to load Mantra scanner DLL: %v\n", err)
		os.Exit(1)
	}

	if !sc.IsConnected() {
		fmt.Println("[-] MFS100 is NOT detected. Please plug in the USB scanner.")
		os.Exit(1)
	}
	fmt.Println("[+] Physical device detected on USB!")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sc.Connect(ctx); err != nil {
		fmt.Printf("[-] Failed to initialize device: %v\n", err)
		os.Exit(1)
	}
	defer sc.Close()

	info, err := sc.DeviceInfo()
	if err != nil {
		fmt.Printf("[-] Failed to get device info: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[+] Manufacturer: %s\n", info.Make)
	fmt.Printf("[+] Model:        %s\n", info.Model)
	fmt.Printf("[+] Serial No:    %s\n", info.Serial)
	fmt.Printf("[+] Resolution:   %d x %d pixels\n", info.Width, info.Height)
	fmt.Printf("[+] SDK Version:  %d\n", info.SDKVer)

	fmt.Println("\n[i] Testing live capture (scanner will remain active for 5 seconds)...")
	fmt.Println("[i] Place and hold your finger firmly on the optical sensor...")

	captureCtx, captureCancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer captureCancel()

	start := time.Now()
	res, err := sc.AutoCapture(captureCtx, 5, 40)
	if err != nil {
		fmt.Printf("[-] Capture finished or cancelled: %v\n", err)
	} else {
		fmt.Printf("[+] Capture SUCCESS in %v!\n", time.Since(start).Round(time.Millisecond))
		fmt.Printf("    - Quality:      %d%%\n", res.Quality)
		fmt.Printf("    - NFIQ:         %d\n", res.NFIQ)
		fmt.Printf("    - ISO Template: %d bytes\n", len(res.ISOTemplate))
		fmt.Printf("    - ANSI Template:%d bytes\n", len(res.ANSITemplate))
	}

	fmt.Println("\n[+] Device cleanup completed. Exit OK.")
}
