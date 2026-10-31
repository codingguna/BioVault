package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStorageLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_fp.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// 1. Create user
	user, err := db.CreateUser(ctx, "Alice Test")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.DisplayName != "Alice Test" {
		t.Fatalf("Unexpected user: %+v", user)
	}

	// 2. Save template
	sampleISO := []byte("ISO_TEMPLATE_DATA_ALICE_INDEX")
	tmpl, err := db.SaveTemplate(ctx, user.ID, "Right Index", "ISO_19794_2", 9025, sampleISO, 85, 1)
	if err != nil {
		t.Fatalf("SaveTemplate failed: %v", err)
	}

	// 3. Retrieve user templates
	templates, err := db.GetUserTemplates(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserTemplates failed: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("Expected 1 template, got %d", len(templates))
	}
	if string(templates[0].DecryptedBytes) != string(sampleISO) {
		t.Fatalf("Decrypted template mismatch: got %s, want %s", string(templates[0].DecryptedBytes), string(sampleISO))
	}

	// 4. GetAllTemplates for 1:N
	allTemplates, err := db.GetAllTemplates(ctx)
	if err != nil {
		t.Fatalf("GetAllTemplates failed: %v", err)
	}
	if len(allTemplates) != 1 {
		t.Fatalf("Expected 1 total template, got %d", len(allTemplates))
	}

	// 5. Audit log check
	events, err := db.ListAuditEvents(ctx, 10)
	if err != nil {
		t.Fatalf("ListAuditEvents failed: %v", err)
	}
	if len(events) < 2 { // USER_CREATED + TEMPLATE_ENROLLED
		t.Fatalf("Expected at least 2 audit events, got %d", len(events))
	}

	_ = tmpl
	_ = os.Remove(dbPath)
}
