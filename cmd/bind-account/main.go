package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"fp-manager/internal/storage"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("   MFS100 Windows Account Biometric Password Binding")
	fmt.Println("==================================================")

	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	dbPath := filepath.Join(exeDir, "mfs100.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		log.Fatalf("[-] Failed to open database: %v\n", err)
	}
	defer db.Close()

	users, err := db.ListUsers(context.Background())
	if err != nil || len(users) == 0 {
		log.Fatalf("[-] No enrolled users found in database. Please enroll fingerprints first.\n")
	}

	user := users[0] // Select first enrolled user (e.g. Guna)
	fmt.Printf("[+] Enrolled Fingerprint Profile: %s (ID: %s)\n", user.DisplayName, user.ID)

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("\nEnter your Windows Username [default: Admin]: ")
	winUser, _ := reader.ReadString('\n')
	winUser = strings.TrimSpace(winUser)
	if winUser == "" {
		winUser = "Admin"
	}

	fmt.Print("Enter your Windows Domain or Computer Name [default: .]: ")
	winDomain, _ := reader.ReadString('\n')
	winDomain = strings.TrimSpace(winDomain)
	if winDomain == "" {
		winDomain = "."
	}

	fmt.Printf("Enter the Windows Password for %s: ", winUser)
	pass, _ := reader.ReadString('\n')
	pass = strings.TrimSpace(pass)
	if pass == "" {
		log.Fatalln("[-] Password cannot be empty.")
	}

	err = db.SaveWindowsCredential(context.Background(), user.ID, winUser, winDomain, pass)
	if err != nil {
		log.Fatalf("[-] Failed to save encrypted credential: %v\n", err)
	}

	fmt.Println("\n==================================================")
	fmt.Printf("[+] SUCCESS! Biometric vault updated for %s\\%s.\n", winDomain, winUser)
	fmt.Println("[+] Your password is encrypted with Windows DPAPI.")
	fmt.Println("[+] You can now unlock your PC with your fingerprint!")
	fmt.Println("==================================================")
}
