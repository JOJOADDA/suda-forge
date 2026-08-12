package agent

import "testing"

func TestAuthorizeIsProjectScoped(t *testing.T) {
	policy := PermissionPolicy{ProjectID: "p1", Allowed: []Permission{PermissionGitCommit}}
	if err := Authorize(policy, "p2", PermissionGitCommit); err != ErrPermissionDenied {
		t.Fatalf("error = %v", err)
	}
	if err := Authorize(policy, "p1", PermissionGitPush); err != ErrPermissionDenied {
		t.Fatalf("error = %v", err)
	}
	if err := Authorize(policy, "p1", PermissionGitCommit); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialReferenceContainsNoRawSecret(t *testing.T) {
	if err := ValidateCredentialReference(CredentialReference{ID: "c1", ProjectID: "p1", ProviderID: "openai", Kind: "api-key", SecretName: "OPENAI_API_KEY"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCredentialReference(CredentialReference{ID: "c1", ProjectID: "p1", ProviderID: "openai", Kind: "api-key"}); err == nil {
		t.Fatal("incomplete reference must be rejected")
	}
}
