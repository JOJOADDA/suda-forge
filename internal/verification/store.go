package verification

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ DB *pgxpool.Pool }

func (s Store) Save(ctx context.Context, run VerificationRun) error {
	if s.DB == nil {
		return errors.New("verification database unavailable")
	}
	summary, _ := json.Marshal(run.Summary)
	profile, _ := json.Marshal(run.Profile)
	state, _ := json.Marshal(run.State)
	if _, err := s.DB.Exec(ctx, `INSERT INTO verification_runs(id,project_id,workflow_id,task_id,task_run_id,status,started_at,completed_at,summary,profile,state) VALUES($1,$2,NULLIF($3,''),NULLIF($4,''),NULLIF($5,''),$6,$7,$8,$9,$10,$11) ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status,completed_at=EXCLUDED.completed_at,summary=EXCLUDED.summary,profile=EXCLUDED.profile,state=EXCLUDED.state`, run.ID, run.ProjectID, run.WorkflowID, run.TaskID, run.TaskRunID, run.Status, run.StartedAt, run.CompletedAt, summary, profile, state); err != nil {
		return err
	}
	for _, result := range run.Results {
		evidence, _ := json.Marshal(result.Evidence)
		var failure any
		if result.Failure != nil {
			failure, _ = json.Marshal(result.Failure)
		}
		check := findCheck(run.Checks, result.CheckID)
		config, _ := json.Marshal(check.Configuration)
		if _, err := s.DB.Exec(ctx, `INSERT INTO verification_checks(id,verification_run_id,type,name,required,status,configuration,evidence,failure,started_at,completed_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT(verification_run_id,id) DO UPDATE SET status=EXCLUDED.status,evidence=EXCLUDED.evidence,failure=EXCLUDED.failure,completed_at=EXCLUDED.completed_at`, result.CheckID, run.ID, check.Type, check.Name, check.Required, result.Status, config, evidence, failure, result.StartedAt, result.CompletedAt); err != nil {
			return err
		}
	}
	for _, failure := range run.Failures {
		raw, _ := json.Marshal(failure)
		if _, err := s.DB.Exec(ctx, `INSERT INTO verification_failures(id,verification_run_id,check_id,failure_type,report) VALUES($1,$2,$3,$4,$5) ON CONFLICT(id) DO UPDATE SET report=EXCLUDED.report`, failure.ID, run.ID, failure.CheckID, failure.FailureType, raw); err != nil {
			return err
		}
	}
	for _, artifact := range run.Artifacts {
		metadata, _ := json.Marshal(artifact.Metadata)
		if _, err := s.DB.Exec(ctx, `INSERT INTO verification_artifacts(id,project_id,task_id,verification_run_id,check_id,kind,path,metadata) VALUES($1,$2,NULLIF($3,''),$4,NULLIF($5,''),$6,$7,$8) ON CONFLICT(id) DO UPDATE SET path=EXCLUDED.path,metadata=EXCLUDED.metadata`, artifact.ID, artifact.ProjectID, artifact.TaskID, artifact.RunID, artifact.CheckID, artifact.Kind, artifact.Path, metadata); err != nil {
			return err
		}
	}
	for _, repair := range run.Repairs {
		plan, _ := json.Marshal(repair.Plan)
		if _, err := s.DB.Exec(ctx, `INSERT INTO verification_repairs(id,verification_run_id,attempt,status,plan,error,started_at,completed_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status,error=EXCLUDED.error,completed_at=EXCLUDED.completed_at`, repair.ID, run.ID, repair.Attempt, repair.Status, plan, repair.Error, repair.StartedAt, repair.CompletedAt); err != nil {
			return err
		}
	}
	return nil
}
func (s Store) Get(ctx context.Context, id ID) (VerificationRun, error) {
	if s.DB == nil {
		return VerificationRun{}, errors.New("verification database unavailable")
	}
	var run VerificationRun
	var status string
	var summary, profile, state []byte
	err := s.DB.QueryRow(ctx, `SELECT id,project_id,COALESCE(workflow_id,''),COALESCE(task_id,''),COALESCE(task_run_id,''),status,started_at,completed_at,summary,profile,state FROM verification_runs WHERE id=$1`, id).Scan(&run.ID, &run.ProjectID, &run.WorkflowID, &run.TaskID, &run.TaskRunID, &status, &run.StartedAt, &run.CompletedAt, &summary, &profile, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return VerificationRun{}, errors.New("verification run not found")
	}
	if err != nil {
		return VerificationRun{}, err
	}
	run.Status = Status(status)
	_ = json.Unmarshal(summary, &run.Summary)
	_ = json.Unmarshal(profile, &run.Profile)
	_ = json.Unmarshal(state, &run.State)
	rows, err := s.DB.Query(ctx, `SELECT id,type,name,required,status,configuration,evidence,failure,started_at,completed_at FROM verification_checks WHERE verification_run_id=$1 ORDER BY id`, id)
	if err != nil {
		return VerificationRun{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var checkID ID
		var typ, name, status string
		var required bool
		var config, evidence, failure []byte
		var result VerificationResult
		if err := rows.Scan(&checkID, &typ, &name, &required, &status, &config, &evidence, &failure, &result.StartedAt, &result.CompletedAt); err != nil {
			return VerificationRun{}, err
		}
		check := VerificationCheck{ID: checkID, Type: CheckType(typ), Name: name, Required: required, Configuration: map[string]any{}}
		_ = json.Unmarshal(config, &check.Configuration)
		_ = json.Unmarshal(evidence, &result.Evidence)
		result.CheckID = checkID
		result.Status = Status(status)
		if len(failure) > 0 {
			result.Failure = &FailureReport{}
			_ = json.Unmarshal(failure, result.Failure)
			run.Failures = append(run.Failures, *result.Failure)
		}
		run.Checks = append(run.Checks, check)
		run.Results = append(run.Results, result)
	}
	if err := rows.Err(); err != nil {
		return VerificationRun{}, err
	}
	repairRows, err := s.DB.Query(ctx, `SELECT id,attempt,status,plan,COALESCE(error,''),started_at,completed_at FROM verification_repairs WHERE verification_run_id=$1 ORDER BY attempt`, id)
	if err != nil {
		return VerificationRun{}, err
	}
	defer repairRows.Close()
	for repairRows.Next() {
		var repair RepairAttempt
		var status string
		var plan []byte
		if err := repairRows.Scan(&repair.ID, &repair.Attempt, &status, &plan, &repair.Error, &repair.StartedAt, &repair.CompletedAt); err != nil {
			return VerificationRun{}, err
		}
		repair.RunID = id
		repair.Status = Status(status)
		_ = json.Unmarshal(plan, &repair.Plan)
		run.Repairs = append(run.Repairs, repair)
	}
	return run, repairRows.Err()
}
func (s Store) ListForTask(ctx context.Context, taskID string) ([]VerificationRun, error) {
	if s.DB == nil {
		return nil, errors.New("verification database unavailable")
	}
	rows, err := s.DB.Query(ctx, `SELECT id FROM verification_runs WHERE task_id=$1 ORDER BY started_at DESC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []VerificationRun{}
	for rows.Next() {
		var id ID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		run, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, rows.Err()
}
func (s Store) Artifacts(ctx context.Context, runID ID) ([]VerificationArtifact, error) {
	if s.DB == nil {
		return nil, errors.New("verification database unavailable")
	}
	rows, err := s.DB.Query(ctx, `SELECT id,project_id,COALESCE(task_id,''),verification_run_id,COALESCE(check_id,''),kind,path,metadata FROM verification_artifacts WHERE verification_run_id=$1 ORDER BY id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []VerificationArtifact{}
	for rows.Next() {
		var a VerificationArtifact
		var metadata []byte
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.TaskID, &a.RunID, &a.CheckID, &a.Kind, &a.Path, &metadata); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metadata, &a.Metadata)
		out = append(out, a)
	}
	return out, rows.Err()
}
