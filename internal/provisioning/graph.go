package provisioning

import "fmt"

func DefaultSteps() []Step {
	return []Step{
		{ID: "runtime", Name: "Create isolated runtime", Status: StepPending, Required: true},
		{ID: "configure", Name: "Configure isolation and filesystem", Status: StepPending, Dependencies: []string{"runtime"}, Required: true},
		{ID: "system", Name: "Install system packages", Status: StepPending, Dependencies: []string{"configure"}, Required: true},
		{ID: "language", Name: "Install language runtime", Status: StepPending, Dependencies: []string{"system"}, Required: true},
		{ID: "framework", Name: "Install framework", Status: StepPending, Dependencies: []string{"language"}, Required: true},
		{ID: "tools", Name: "Install package/build/test tools", Status: StepPending, Dependencies: []string{"language"}, Required: true},
		{ID: "agents", Name: "Install agent CLIs", Status: StepPending, Dependencies: []string{"tools"}, Required: true},
		{ID: "browser", Name: "Install browser automation", Status: StepPending, Dependencies: []string{"tools"}, Required: false},
		{ID: "project", Name: "Install project dependencies and initialize Git", Status: StepPending, Dependencies: []string{"framework", "tools"}, Required: true},
		{ID: "verify", Name: "Verify environment capabilities", Status: StepPending, Dependencies: []string{"project", "agents", "browser"}, Required: true},
	}
}
func ValidateGraph(steps []Step) error {
	byID := map[string]bool{}
	for _, s := range steps {
		if s.ID == "" {
			return fmt.Errorf("step id is required")
		}
		if byID[s.ID] {
			return fmt.Errorf("duplicate step %s", s.ID)
		}
		byID[s.ID] = true
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("provisioning dependency cycle at %s", id)
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		var step Step
		for _, s := range steps {
			if s.ID == id {
				step = s
				break
			}
		}
		for _, dep := range step.Dependencies {
			if !byID[dep] {
				return fmt.Errorf("step %s depends on missing step %s", id, dep)
			}
			if err := visit(dep); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	for _, s := range steps {
		if err := visit(s.ID); err != nil {
			return err
		}
	}
	return nil
}
func ReadySteps(steps []Step) []Step {
	out := []Step{}
	for _, s := range steps {
		if s.Status != StepPending {
			continue
		}
		ready := true
		for _, dep := range s.Dependencies {
			for _, candidate := range steps {
				if candidate.ID == dep && candidate.Status != StepPassed {
					ready = false
				}
			}
		}
		if ready {
			out = append(out, s)
		}
	}
	return out
}
