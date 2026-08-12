package filesystem

import (
	"errors"
	"path/filepath"
	"strings"
)

var ErrPathOutsideWorkspace = errors.New("path escapes project workspace")

// Resolve confines every project path to the runtime workspace. The runtime
// adapter remains responsible for executing the resulting operation in LXC.
func Resolve(workspace, requested string) (string, error) {
	if workspace == "" {
		return "", errors.New("workspace is required")
	}
	requested = strings.TrimSpace(requested)
	for _, part := range strings.FieldsFunc(requested, func(r rune) bool { return r == '/' || r == filepath.Separator }) {
		if part == ".." {
			return "", ErrPathOutsideWorkspace
		}
	}
	if requested == "" {
		requested = "."
	}
	clean := filepath.Clean("/" + requested)
	candidate := filepath.Join(workspace, clean)
	root := filepath.Clean(workspace)
	if candidate != root && !strings.HasPrefix(candidate, root+string(filepath.Separator)) {
		return "", ErrPathOutsideWorkspace
	}
	return candidate, nil
}
