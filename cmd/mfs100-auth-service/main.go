package main

import (
	"context"
	"flag"
	"fmt"
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

type syncWriter struct {
	f *os.File
}

func (s *syncWriter) Write(p []byte) (n int, err error) {
	n, err = s.f.Write(p)
	_ = s.f.Sync()
	return n, err
}

func main() {
	exePath, _ := os.Executable()
	defaultDbPath := storage.DefaultDatabasePath()

	dbPath := flag.String("db", defaultDbPath, "SQLite database file path")
	useMock := flag.Bool("mock", false, "Use mock scanner")
	flag.Parse()

	// Setup immediate-flush logging to C:\ProgramData\MantraUnlock\Logs\service.log (with fallback to C:\ProgramData\Mfs100)
	logDir := `C:\ProgramData\MantraUnlock\Logs`
	_ = os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "service.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		legacyLogDir := `C:\ProgramData\Mfs100`
		_ = os.MkdirAll(legacyLogDir, 0755)
		logFile, err = os.OpenFile(filepath.Join(legacyLogDir, "service.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	}
	if err == nil {
		defer logFile.Close()
		sw := &syncWriter{f: logFile}
		log.SetOutput(sw)
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

		case "DEBUG":
			tmpls, tErr := db.GetAllTemplates(context.Background())
			users, uErr := db.ListUsers(context.Background())
			return ipc.Response{
				Status:  "DEBUG",
				Message: fmt.Sprintf("db=%s users=%d(err:%v) tmpls=%d(err:%v)", absDbPath, len(users), uErr, len(tmpls), tErr),
			}

		case "TRIGGER_UNLOCK":
			log.Println("[+] Received unlock request from LogonUI/pipe. Waiting for finger...")
			captureCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			capRes, err := sc.AutoCapture(captureCtx, 10, 50)
			if err != nil {
				log.Printf("[-] Scan failed or timed out: %v\n", err)
				return ipc.Response{Status: "FAIL", Message: fmt.Sprintf("Scan failed: %v", err)}
			}

			log.Printf("[+] Finger captured (Quality: %d, NFIQ: %d, ISOLen: %d). Performing 1:N Identification...\n",
				capRes.Quality, capRes.NFIQ, len(capRes.ISOTemplate))

			// Perform 1:N Identification
			idRes, err := matcher.IdentifyUser(context.Background(), capRes.ISOTemplate)
			if err != nil {
				log.Printf("[-] Identify error: %v\n", err)
				return ipc.Response{Status: "FAIL", Message: fmt.Sprintf("Identify error: %v", err)}
			}
			if !idRes.Identified || idRes.User == nil {
				log.Printf("[-] Finger not recognized. Best score: %d (needed 1400), Best finger: %s\n",
					idRes.BestScore, idRes.BestFinger)

				var msg string
				if idRes.BestScore > 0 {
					msg = fmt.Sprintf("Fingerprint not recognized (score %d, needed 1400 - press firmly)", idRes.BestScore)
				} else {
					allT, _ := db.GetAllTemplates(context.Background())
					msg = fmt.Sprintf("Fingerprint not recognized (highest score 0 on %d templates)", len(allT))
				}
				return ipc.Response{Status: "FAIL", Message: msg}
			}

			log.Printf("[+] Finger matched profile: %s via %s (Score: %d)\n",
				idRes.User.DisplayName, idRes.MatchedFinger, idRes.Score)

			// Retrieve bound Windows credential
			cred, err := db.GetWindowsCredential(context.Background(), idRes.User.ID)
			if err != nil {
				log.Printf("[-] User %s has no Windows account bound: %v\n", idRes.User.DisplayName, err)
				return ipc.Response{Status: "FAIL", Message: "No Windows account bound to this fingerprint"}
			}
			if cred.UserID != idRes.User.ID {
				log.Printf("[+] Single account mode: Inheriting Windows account %s\\%s for user %s\n",
					cred.WindowsDomain, cred.WindowsUser, idRes.User.DisplayName)
			}

			log.Printf("[+] Biometric authorization approved for %s\\%s!\n", cred.WindowsDomain, cred.WindowsUser)
			_ = db.LogAudit(context.Background(), "WINDOWS_UNLOCK_SUCCESS", idRes.User.ID, idRes.User.DisplayName,
				"SUCCESS", idRes.Score, fmt.Sprintf("Unlocked account %s\\%s via %s", cred.WindowsDomain, cred.WindowsUser, idRes.MatchedFinger))

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
