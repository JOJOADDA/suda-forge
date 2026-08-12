package verification

import "errors"

var ErrCheckCycle = errors.New("verification check graph contains a cycle")
var ErrCheckDependency = errors.New("verification check graph references a missing dependency")
var ErrDuplicateCheck = errors.New("verification profile contains duplicate check id")

func ValidateChecks(checks []VerificationCheck) error {
	byID := map[ID]VerificationCheck{}
	for _, check := range checks {
		if check.ID == "" {
			return ErrDuplicateCheck
		}
		if _, ok := byID[check.ID]; ok {
			return ErrDuplicateCheck
		}
		byID[check.ID] = check
	}
	state := map[ID]uint8{}
	var visit func(ID) error
	visit = func(id ID) error {
		if state[id] == 1 {
			return ErrCheckCycle
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		for _, dep := range byID[id].Dependencies {
			if _, ok := byID[dep]; !ok {
				return ErrCheckDependency
			}
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for id := range byID {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}
