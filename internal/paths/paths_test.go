package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureWithinAllowed(t *testing.T) {
	tmpDir := t.TempDir()

	// Path inside allowed root: should pass
	if err := EnsureWithinAllowed(filepath.Join(tmpDir, "sub"), []string{tmpDir}); err != nil {
		t.Errorf("expected allowed path, got error: %v", err)
	}

	// Path outside allowed root: should fail
	if err := EnsureWithinAllowed("/etc/passwd", []string{tmpDir}); err == nil {
		t.Error("expected error for path outside allowed roots")
	}
}

func TestResolveOutputIncrement(t *testing.T) {
	tmpDir := t.TempDir()

	// First call: no collision
	p1, err := ResolveOutput("test.txt", tmpDir, "md", "increment")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p1) != "test.md" {
		t.Errorf("expected test.md, got %s", filepath.Base(p1))
	}

	// Create the file to force collision
	os.WriteFile(p1, []byte("x"), 0o644)

	// Second call: should increment
	p2, err := ResolveOutput("test.txt", tmpDir, "md", "increment")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p2) != "test_001.md" {
		t.Errorf("expected test_001.md, got %s", filepath.Base(p2))
	}
}
