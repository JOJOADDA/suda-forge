package filesystem

import "testing"

func TestResolveConfinesWorkspace(t *testing.T) {
	got, err := Resolve("/workspace", "src/main.go")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/workspace/src/main.go" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveRejectsTraversal(t *testing.T) {
	if _, err := Resolve("/workspace", "../../etc/passwd"); err != ErrPathOutsideWorkspace {
		t.Fatalf("expected traversal rejection, got %v", err)
	}
}
