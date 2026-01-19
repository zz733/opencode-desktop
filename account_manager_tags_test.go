package main

import (
	"os"
	"testing"
)

func setupTestAccountManagerForTags(t *testing.T) (*AccountManager, string, func()) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "kiro-test-tags")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cryptoService := NewCryptoService("test-key")
	storageService := NewStorageService(tmpDir, cryptoService)
	accountMgr := NewAccountManager(storageService, cryptoService)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return accountMgr, tmpDir, cleanup
}

func TestTagManagement(t *testing.T) {
	am, _, cleanup := setupTestAccountManagerForTags(t)
	defer cleanup()

	// 1. Test AddTag
	tag1 := Tag{
		Name:        "Dev",
		Color:       "#FF0000",
		Description: "Development accounts",
	}

	err := am.AddTag(tag1)
	if err != nil {
		t.Fatalf("Failed to add tag: %v", err)
	}

	// Verify tag added
	tags := am.GetTags()
	if len(tags) != 1 {
		t.Errorf("Expected 1 tag, got %d", len(tags))
	}
	if tags[0].Name != tag1.Name {
		t.Errorf("Expected tag name %s, got %s", tag1.Name, tags[0].Name)
	}

	// 2. Test Duplicate Tag
	err = am.AddTag(tag1)
	if err == nil {
		t.Error("Expected error when adding duplicate tag")
	}

	// 3. Test Add Another Tag
	tag2 := Tag{
		Name:  "Prod",
		Color: "#00FF00",
	}
	err = am.AddTag(tag2)
	if err != nil {
		t.Fatalf("Failed to add second tag: %v", err)
	}

	tags = am.GetTags()
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}

	// 4. Test Delete Tag
	err = am.DeleteTag("Dev")
	if err != nil {
		t.Fatalf("Failed to delete tag: %v", err)
	}

	tags = am.GetTags()
	if len(tags) != 1 {
		t.Errorf("Expected 1 tag after deletion, got %d", len(tags))
	}
	if tags[0].Name != "Prod" {
		t.Errorf("Expected remaining tag to be Prod, got %s", tags[0].Name)
	}

	// 5. Test Delete Non-existent Tag
	err = am.DeleteTag("NonExistent")
	if err == nil {
		t.Error("Expected error when deleting non-existent tag")
	}
}

func TestTagPersistence(t *testing.T) {
	am, tmpDir, cleanup := setupTestAccountManagerForTags(t)
	defer cleanup()

	// Add a tag
	tag := Tag{Name: "PersistTest", Color: "#0000FF"}
	am.AddTag(tag)

	// Create new manager instance pointing to same storage
	cryptoService := NewCryptoService("test-key")
	storageService := NewStorageService(tmpDir, cryptoService)
	newAm := NewAccountManager(storageService, cryptoService)

	// Verify tag loaded
	tags := newAm.GetTags()
	if len(tags) != 1 {
		t.Errorf("Expected 1 tag loaded, got %d", len(tags))
	}
	if len(tags) > 0 && tags[0].Name != "PersistTest" {
		t.Errorf("Expected tag PersistTest, got %s", tags[0].Name)
	}
}
