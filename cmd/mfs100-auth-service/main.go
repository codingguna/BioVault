package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"fp-manager/internal/ipc"
	"fp-manager/internal/matching"
	"fp-manager/internal/scanner"
	"fp-manager/internal/scanner/mantra"
	"fp-manager/internal/scanner/mock"
	"fp-manager/internal/storage"
)

func main() {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	defaultDbPath := filepath.Join(exeDir, "mfs100.db")

	dbPath := flag.String("db", defaultDbPath, "SQLite database file path")
	useMock := flag.Bool("mock", false, "Use mock scanner")
	testUnlock := flag.Bool("test-unlock", false, "Client test: send TRIGGER_UNLOCK to running service")
	flag.Parse()

	// If running as client test
	if *testUnlock {
		fmt.Println("[*] Sending TRIGGER_UNLOCK request to running auth service...")
		resp, err := ipc.ClientSend(ipc.Request{Command: "TRIGGER_UNLOCK"})
		if err != nil {
			log.Fatalf("[-] Failed to communicate with auth service pipe: %v\n", err)
		}
		fmt.Printf("[+] Response Status: %s\n", resp.Status)
		if resp.Status == "SUCCESS" {
			fmt.Printf("[+] Verified User: %s\\%s (Score: %d)\n", resp.Domain, resp.User, resp.Score)
			fmt.Println("[+] Password successfully decrypted from biometric vault!")
		} else {
			fmt.Printf("[-] Unlock failed: %s\n", resp.Message)
		}
		return
	}

	// Setup logging to C:\ProgramData\Mfs100\service.log
	logDir := filepath.Join(os.Getenv("ProgramData"), "Mfs100")
	_ = os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "service.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	}

	log.Println("==================================================")
	log.Println("       MFS100 Windows Background Auth Service     ")
	log.Println("==================================================")
	log.Printf("[+] Service executable: %s\n", exePath)
	log.Printf("[+] Database path: %s\n", *dbPath)

	// 1. Storage
	absDbPath, _ := filepath.Abs(*dbPath)
	db, err := storage.Open(absDbPath)
	if err != nil {
		log.Fatalf("[-] Failed to open database: %v\n", err)
	}
	defer db.Close()

	// 2. Scanner
	var sc scanner.Scanner
	if *useMock {
		sc = mock.New(true)
		_ = sc.Connect(context.Background())
	} else {
		mSc, err := mantra.New("")
		if err != nil {
			log.Fatalf("[-] Failed to initialize Mantra driver: %v\n", err)
		}
		sc = mSc
		if err := sc.Connect(context.Background()); err != nil {
			log.Printf("[!] Warning: Connect scanner error: %v\n", err)
		} else {
			info, _ := sc.DeviceInfo()
			if info != nil {
				log.Printf("[+] Scanner online: %s %s (Serial: %s)\n", info.Make, info.Model, info.Serial)
			}
		}
	}
	defer sc.Close()

	matcher := matching.NewService(sc, db)

	// 3. Named Pipe Handler
	pipeHandler := func(req ipc.Request) ipc.Response {
		switch req.Command {
		case "STATUS":
			if sc.IsConnected() {
				return ipc.Response{Status: "READY"}
			}
			return ipc.Response{Status: "DISCONNECTED"}

		case "TRIGGER_UNLOCK":
			log.Println("[+] Received unlock request from LogonUI/pipe. Waiting for finger...")
			captureCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			capRes, err := sc.AutoCapture(captureCtx, 10, 45)
			if err != nil {
				log.Printf("[-] Scan failed or timed out: %v\n", err)
				return ipc.Response{Status: "FAIL", Message: fmt.Sprintf("Scan failed: %v", err)}
			}

			// Perform 1:N Identification
			idRes, err := matcher.IdentifyUser(context.Background(), capRes.ISOTemplate)
			if err != nil || !idRes.Identified || idRes.User == nil {
				log.Println("[-] Finger not recognized in database.")
				return ipc.Response{Status: "FAIL", Message: "Fingerprint not recognized"}
			}

			log.Printf("[+] Finger matched profile: %s (Score: %d)\n", idRes.User.DisplayName, idRes.Score)

			// Retrieve bound Windows credential
			cred, err := db.GetWindowsCredential(context.Background(), idRes.User.ID)
			if err != nil {
				log.Printf("[-] User %s has no Windows account bound: %v\n", idRes.User.DisplayName, err)
				return ipc.Response{Status: "FAIL", Message: "No Windows account bound to this fingerprint"}
			}

			log.Printf("[+] Biometric authorization approved for %s\\%s!\n", cred.WindowsDomain, cred.WindowsUser)
			_ = db.LogAudit(context.Background(), "WINDOWS_UNLOCK_SUCCESS", idRes.User.ID, idRes.User.DisplayName,
				"SUCCESS", idRes.Score, fmt.Sprintf("Unlocked account %s\\%s", cred.WindowsDomain, cred.WindowsUser))

			return ipc.Response{
				Status:   "SUCCESS",
				User:     cred.WindowsUser,
				Domain:   cred.WindowsDomain,
				Password: cred.PlainPassword,
				Score:    idRes.Score,
			}

		default:
			return ipc.Response{Status: "UNKNOWN_COMMAND"}
		}
	}

	server := ipc.NewServer(pipeHandler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Start(ctx); err != nil {
		log.Fatalf("[-] Failed to start Named Pipe server: %v\n", err)
	}

	log.Printf("[+] Named Pipe listening on: %s\n", ipc.PipeName)
	log.Println("[+] Auth Service running. Ready to receive unlock requests.")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("\n[+] Shutting down auth service...")
	server.Stop()
	log.Println("[+] Stopped.")
}
