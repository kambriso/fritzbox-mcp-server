package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

)

// TestFetchAllXML tests downloading all XML descriptors from a FRITZ!Box
// This test requires a FRITZ!Box to be accessible at the configured address
// Set FRITZ_HOST in .env file or environment variable to run this test
func TestFetchAllXML(t *testing.T) {
	// Try to load config (which loads .env file)
	_, _ = load()

	// Check if FRITZ_HOST is set (from .env or environment)
	host := os.Getenv("FRITZ_HOST")
	if host == "" {
		t.Skip("Skipping integration test: FRITZ_HOST not set in .env or environment")
	}

	// Get port (default 49000)
	port := os.Getenv("FRITZ_PORT")
	if port == "" {
		port = "49000"
	}

	// Construct baseURL
	baseURL := "http://" + host + ":" + port

	// Create temporary directory for XML files
	tempDir := t.TempDir()
	xmlDir := filepath.Join(tempDir, "xml")

	t.Logf("Fetching XML from %s to %s", baseURL, xmlDir)

	// Fetch all XML files
	ctx := context.Background()
	if err := fetchAllXML(ctx, baseURL, xmlDir); err != nil {
		t.Fatalf("fetchAllXML failed: %v", err)
	}

	// Verify tr64desc.xml exists
	descPath := filepath.Join(xmlDir, "tr64desc.xml")
	if _, err := os.Stat(descPath); os.IsNotExist(err) {
		t.Errorf("tr64desc.xml not found at %s", descPath)
	}

	// Count downloaded files
	entries, err := os.ReadDir(xmlDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	if len(entries) < 2 {
		t.Errorf("Expected at least 2 XML files (tr64desc.xml + SCPDs), got %d", len(entries))
	}

	t.Logf("Successfully downloaded %d XML files:", len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		t.Logf("  - %s (%d bytes)", entry.Name(), info.Size())
	}
}

// TestFetchAllXMLInvalidURL tests error handling for invalid URLs
func TestFetchAllXMLInvalidURL(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	err := fetchAllXML(ctx, "http://invalid-host-that-does-not-exist:49000", tempDir)
	if err == nil {
		t.Error("Expected error for invalid host, got nil")
	}
}
