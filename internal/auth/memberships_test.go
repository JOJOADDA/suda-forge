package auth

import "testing"

func TestProjectPermissionAllowed(t *testing.T) {
	tests := []struct {
		role       string
		permission ProjectPermission
		allowed    bool
	}{
		{"viewer", PermissionRead, true},
		{"viewer", PermissionRun, false},
		{"runner", PermissionRead, true},
		{"runner", PermissionRun, true},
		{"runner", PermissionEdit, false},
		{"editor", PermissionEdit, true},
		{"editor", PermissionRun, true},
		{"editor", PermissionDeploy, false},
		{"owner", PermissionDeploy, true},
		{"unknown", PermissionRead, false},
	}
	for _, test := range tests {
		t.Run(test.role+":"+string(test.permission), func(t *testing.T) {
			if got := projectPermissionAllowed(test.role, test.permission); got != test.allowed {
				t.Fatalf("projectPermissionAllowed(%q, %q) = %v, want %v", test.role, test.permission, got, test.allowed)
			}
		})
	}
}
