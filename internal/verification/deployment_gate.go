package verification

import "errors"

var ErrVerificationGateFailed = errors.New("deployment gate requires a passed verification run")

type DeploymentGate interface{ Allow(VerificationRun) error }
type RequiredVerificationGate struct{}

func (RequiredVerificationGate) Allow(run VerificationRun) error {
	if !TaskVerified(run) {
		return ErrVerificationGateFailed
	}
	return nil
}
