package agent

import (
	"errors"
	"slices"
)

var ErrPermissionDenied = errors.New("agent permission denied")

func (p PermissionPolicy) Allows(permission Permission) bool {
	return slices.Contains(p.Allowed, permission)
}
func (p PermissionPolicy) NeedsApproval(permission Permission) bool {
	return slices.Contains(p.ApprovalRequired, permission)
}
func Authorize(policy PermissionPolicy, projectID string, permission Permission) error {
	if policy.ProjectID != projectID || !policy.Allows(permission) {
		return ErrPermissionDenied
	}
	return nil
}
func ValidateCredentialReference(ref CredentialReference) error {
	if ref.ID == "" || ref.ProjectID == "" || ref.ProviderID == "" || ref.Kind == "" || ref.SecretName == "" {
		return errors.New("credential reference is incomplete")
	}
	return nil
}
