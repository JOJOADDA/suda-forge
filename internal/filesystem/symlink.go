package filesystem

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolveExisting validates an existing path after symlink resolution. It is
// intended for read, delete, and metadata operations inside the runtime.
func ResolveExisting(workspace, requested string) (string, error) {
	candidate, err := Resolve(workspace, requested)
	if err != nil {
		return "", err
	}
	rootReal, err := filepath.EvalSymlinks(filepath.Clean(workspace))
	if err != nil {
		return "", err
	}
	candidateReal, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", err
	}
	if candidateReal != rootReal && !strings.HasPrefix(candidateReal, rootReal+string(filepath.Separator)) {
		return "", ErrPathOutsideWorkspace
	}
	return candidateReal, nil
}

// ValidateParent prevents a new file from escaping through a symlinked parent.
func ValidateParent(workspace, requested string) error {
	candidate, err := Resolve(workspace, requested)
	if err != nil {
		return err
	}
	parentReal, err := filepath.EvalSymlinks(filepath.Dir(candidate))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	rootReal, err := filepath.EvalSymlinks(filepath.Clean(workspace))
	if err != nil {
		return err
	}
	if parentReal != rootReal && !strings.HasPrefix(parentReal, rootReal+string(filepath.Separator)) {
		return ErrPathOutsideWorkspace
	}
	return nil
}
