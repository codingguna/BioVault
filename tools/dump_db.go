package main

import (
	"context"
	"fmt"
	"fp-manager/internal/storage"
)

func main() {
	dbPath := `d:\software\fp\mfs100.db`
	fmt.Printf("[+] DB Path: %s\n", dbPath)

	db, err := storage.Open(dbPath)
	if err != nil {
		fmt.Printf("[-] Failed to open DB: %v\n", err)
		return
	}
	defer db.Close()

	ctx := context.Background()

	users, err := db.ListUsers(ctx)
	fmt.Printf("[+] Users (%d, err=%v):\n", len(users), err)
	for _, u := range users {
		hasCred := db.HasWindowsCredential(ctx, u.ID)
		cred, cErr := db.GetWindowsCredential(ctx, u.ID)
		winInfo := "NONE"
		if cErr == nil && cred != nil {
			winInfo = fmt.Sprintf("%s\\%s (pass len=%d, direct=%v)", cred.WindowsDomain, cred.WindowsUser, len(cred.PlainPassword), cred.UserID == u.ID)
		}
		fmt.Printf("    - ID: %s | Name: '%s' | Fingers: %d | HasCred: %v | WinCred: %s\n",
			u.ID, u.DisplayName, u.FingerCount, hasCred, winInfo)
	}

	tmpls, err := db.GetAllTemplates(ctx)
	fmt.Printf("\n[+] Total Templates in DB: %d (err=%v)\n", len(tmpls), err)
	for i, t := range tmpls {
		fmt.Printf("    [%d] ID: %s | UserID: %s | Finger: %s | Quality: %d | DecryptedLen: %d\n",
			i+1, t.ID, t.UserID, t.FingerLabel, t.Quality, len(t.DecryptedBytes))
	}

	anyCred, err := db.GetAnyWindowsCredential(ctx)
	fmt.Printf("\n[+] GetAnyWindowsCredential: err=%v, cred=%+v\n", err, anyCred)
}
