package verification

import "suda-forge/internal/orchestration"

func TaskVerified(run VerificationRun) bool {
	return run.Status == Passed && run.Summary.RequiredFailed == 0
}
func WorkflowVerified(runs []VerificationRun) bool {
	if len(runs) == 0 {
		return false
	}
	for _, run := range runs {
		if !TaskVerified(run) {
			return false
		}
	}
	return true
}
func ApplyTaskGate(task *orchestration.Task, run VerificationRun) bool {
	if task == nil || !TaskVerified(run) {
		return false
	}
	if task.Status != orchestration.TaskSucceeded && task.Status != orchestration.TaskRunning {
		return false
	}
	return true
}
func ApplyWorkflowGate(workflow *orchestration.Workflow, runs []VerificationRun) bool {
	if workflow == nil || !WorkflowVerified(runs) {
		return false
	}
	if workflow.Status != orchestration.WorkflowSucceeded && workflow.Status != orchestration.WorkflowRunning {
		return false
	}
	return true
}
