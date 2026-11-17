package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBoltStorageUserCRUD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bolt-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := NewBoltStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to open bbolt: %v", err)
	}
	defer store.Close()

	userID := int64(998877)

	// Create user
	u, err := store.GetOrCreateUser(userID)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}
	if u.ID != userID {
		t.Errorf("expected ID %d, got %d", userID, u.ID)
	}

	// Update user settings
	u.WatermarkText = "TestWatermark"
	u.AsFile = true
	u.WatermarkColor = 2
	if err := store.SaveUser(u); err != nil {
		t.Fatalf("SaveUser failed: %v", err)
	}

	// Retrieve user again
	retrieved, err := store.GetUser(userID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if retrieved.WatermarkText != "TestWatermark" {
		t.Errorf("expected watermark 'TestWatermark', got '%s'", retrieved.WatermarkText)
	}
	if !retrieved.AsFile {
		t.Errorf("expected AsFile = true, got false")
	}
	if retrieved.WatermarkColor != 2 {
		t.Errorf("expected WatermarkColor = 2, got %d", retrieved.WatermarkColor)
	}

	count, err := store.TotalUsersCount()
	if err != nil {
		t.Fatalf("TotalUsersCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}
}
