package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"fp-manager/internal/scanner"
	"fp-manager/internal/scanner/mantra"
	"fp-manager/internal/scanner/mock"
	"fp-manager/internal/storage"
	"fp-manager/internal/web"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	dbPath := flag.String("db", "mfs100.db", "SQLite database file path")
	useMock := flag.Bool("mock", false, "Use mock scanner instead of physical hardware")
	noBrowser := flag.Bool("no-browser", false, "Do not auto-open default web browser")
	flag.Parse()

	log.Println("==================================================")
	log.Println("     MFS100 Fingerprint Manager Desktop App      ")
	log.Println("==================================================")

	// 1. Initialize SQLite Database
	absDbPath, _ := filepath.Abs(*dbPath)
	log.Printf("[+] Initializing database at: %s\n", absDbPath)
	db, err := storage.Open(absDbPath)
	if err != nil {
		log.Fatalf("[-] Failed to open database: %v\n", err)
	}
	defer db.Close()

	// 2. Initialize Scanner Adapter
	var sc scanner.Scanner
	if *useMock {
		log.Println("[!] Running with MOCK scanner simulator.")
		mockSc := mock.New(true)
		_ = mockSc.Connect(context.Background())
		sc = mockSc
	} else {
		log.Println("[+] Loading Mantra MFS100 native driver...")
		mSc, err := mantra.New("")
		if err != nil {
			log.Printf("[!] Warning: Could not initialize Mantra driver DLL: %v\n", err)
			log.Println("[!] Falling back to mock scanner for testing.")
			mockSc := mock.New(false)
			sc = mockSc
		} else {
			sc = mSc
			if sc.IsConnected() {
				log.Println("[+] Physical MFS100 scanner detected! Connecting...")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := sc.Connect(ctx); err != nil {
					log.Printf("[!] Connect error: %v\n", err)
				} else {
					info, _ := sc.DeviceInfo()
					if info != nil {
						log.Printf("[+] Connected to %s %s (Serial: %s, SDK: v%d)\n", info.Make, info.Model, info.Serial, info.SDKVer)
					}
				}
				cancel()
			} else {
				log.Println("[!] MFS100 scanner is not currently plugged in. You can plug it in anytime.")
			}
		}
	}
	defer func() {
		log.Println("[+] Closing scanner...")
		_ = sc.Close()
	}()

	// 3. Initialize Web Server
	srvCore := web.NewServer(sc, db)
	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("[-] Failed to listen on %s: %v\n", addr, err)
	}

	appURL := fmt.Sprintf("http://127.0.0.1:%d", listener.Addr().(*net.TCPAddr).Port)
	log.Printf("[+] Web UI is running at: %s\n", appURL)

	httpServer := &http.Server{
		Handler: srvCore.Handler(),
	}

	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[-] HTTP server error: %v\n", err)
		}
	}()

	// 4. Auto-launch default browser
	if !*noBrowser {
		go func() {
			time.Sleep(300 * time.Millisecond)
			_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", appURL).Start()
		}()
	}

	log.Println("[+] Application ready. Press Ctrl+C to terminate.")

	// 5. Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("\n[+] Shutting down gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()

	_ = httpServer.Shutdown(shutdownCtx)
	log.Println("[+] Server stopped. Goodbye!")
}
