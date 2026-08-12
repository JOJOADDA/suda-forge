package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveExistingRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveExisting(root, "link/secret"); err != ErrPathOutsideWorkspace {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateParentRejectsSymlinkedParent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if err := ValidateParent(root, "link/new.txt"); err != ErrPathOutsideWorkspace {
		t.Fatalf("error = %v", err)
	}
}
