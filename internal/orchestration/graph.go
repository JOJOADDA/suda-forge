package orchestration

import "sort"

func ValidateGraph(graph TaskGraph) error {
	for id, task := range graph.Tasks {
		if task.ID != id {
			return ErrInvalidParent
		}
		seen := map[ID]bool{}
		for _, dep := range graph.Dependencies[id] {
			if seen[dep] {
				return ErrDuplicateDependency
			}
			seen[dep] = true
			if _, ok := graph.Tasks[dep]; !ok {
				return ErrMissingDependency
			}
		}
		if task.ParentTaskID != "" {
			parent, ok := graph.Tasks[task.ParentTaskID]
			if !ok || (parent.WorkflowID != task.WorkflowID) {
				return ErrInvalidParent
			}
		}
	}
	state := map[ID]uint8{}
	var visit func(ID) error
	visit = func(id ID) error {
		if state[id] == 1 {
			return ErrCycle
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		deps := append([]ID(nil), graph.Dependencies[id]...)
		sort.Slice(deps, func(i, j int) bool { return deps[i] < deps[j] })
		for _, dep := range deps {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	ids := make([]ID, 0, len(graph.Tasks))
	for id := range graph.Tasks {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}
func ReadyTasks(graph TaskGraph) []ID {
	out := []ID{}
	ids := make([]ID, 0, len(graph.Tasks))
	for id := range graph.Tasks {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		task := graph.Tasks[id]
		if task.Status != TaskPending && task.Status != TaskReady {
			continue
		}
		ready := true
		for _, dep := range graph.Dependencies[id] {
			if graph.Tasks[dep].Status != TaskSucceeded {
				ready = false
				break
			}
		}
		if ready {
			out = append(out, id)
		}
	}
	return out
}
func PropagateFailures(graph *TaskGraph) {
	changed := true
	for changed {
		changed = false
		for id, task := range graph.Tasks {
			if task.Status != TaskPending && task.Status != TaskReady {
				continue
			}
			for _, dep := range graph.Dependencies[id] {
				if graph.Tasks[dep].Status == TaskFailed || graph.Tasks[dep].Status == TaskTimedOut || graph.Tasks[dep].Status == TaskCancelled {
					task.Status = TaskBlocked
					graph.Tasks[id] = task
					changed = true
					break
				}
			}
		}
	}
}
