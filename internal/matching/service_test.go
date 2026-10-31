package matching

import (
	"context"
	"path/filepath"
	"testing"

	"fp-manager/internal/scanner/mock"
	"fp-manager/internal/storage"
)

func TestMatchingService(t *testing.T) {
	tempDir := t.TempDir()
	db, err := storage.Open(filepath.Join(tempDir, "test_match.db"))
	if err != nil {
		t.Fatalf("Storage open failed: %v", err)
	}
	defer db.Close()

	sc := mock.New(true)
	if err := sc.Connect(context.Background()); err != nil {
		t.Fatalf("Mock scanner connect failed: %v", err)
	}
	defer sc.Close()

	ctx := context.Background()
	svc := NewService(sc, db)

	// Create 2 test users
	user1, err := db.CreateUser(ctx, "Bob Smith")
	if err != nil {
		t.Fatalf("Create user1 failed: %v", err)
	}
	user2, err := db.CreateUser(ctx, "Carol Danvers")
	if err != nil {
		t.Fatalf("Create user2 failed: %v", err)
	}

	// Enrol finger for Bob
	bobISO := []byte("BOB_THUMB_ISO_DATA")
	_, err = db.SaveTemplate(ctx, user1.ID, "Right Thumb", "ISO_19794_2", 9025, bobISO, 90, 1)
	if err != nil {
		t.Fatalf("Save bob template failed: %v", err)
	}

	// 1. Verify Bob with matching template
	verifyRes, err := svc.VerifyUser(ctx, user1.ID, bobISO)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !verifyRes.Matched {
		t.Fatalf("Expected Bob to match, got false (score %d)", verifyRes.Score)
	}

	// 2. Verify Bob with non-matching template
	strangerISO := []byte("UNKNOWN_PERSON_ISO")
	verifyFail, err := svc.VerifyUser(ctx, user1.ID, strangerISO)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if verifyFail.Matched {
		t.Fatalf("Expected stranger to fail Bob's verify")
	}

	// 3. Identify Bob (1:N)
	idRes, err := svc.IdentifyUser(ctx, bobISO)
	if err != nil {
		t.Fatalf("Identify failed: %v", err)
	}
	if !idRes.Identified || idRes.User.ID != user1.ID {
		t.Fatalf("Expected to identify Bob, got %+v", idRes)
	}

	// 4. Identify stranger (1:N)
	idStranger, err := svc.IdentifyUser(ctx, strangerISO)
	if err != nil {
		t.Fatalf("Identify stranger failed: %v", err)
	}
	if idStranger.Identified {
		t.Fatalf("Expected stranger not to be identified")
	}

	_ = user2
}
