package projectintelligence

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ DB *pgxpool.Pool }

func (s Store) SaveAnalysis(ctx context.Context, a Analysis) error {
	if s.DB == nil {
		return errors.New("intelligence database unavailable")
	}
	platforms, _ := json.Marshal(a.Intent.Platforms)
	constraints, _ := json.Marshal(a.Intent.Constraints)
	preferences, _ := json.Marshal(a.Intent.Preferences)
	budget, _ := json.Marshal(a.Intent.Budget)
	if _, err := s.DB.Exec(ctx, `INSERT INTO project_intents(id,project_id,user_prompt,target_audience,platforms,constraints_json,preferences,budget) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO UPDATE SET user_prompt=EXCLUDED.user_prompt`, a.Decision.ID+"/intent", a.Intent.ProjectID, a.Intent.UserPrompt, a.Intent.TargetAudience, platforms, constraints, preferences, budget); err != nil {
		return err
	}
	for _, r := range a.Requirements {
		if _, err := s.DB.Exec(ctx, `INSERT INTO project_requirements(id,project_id,category,description,priority,required,source,confidence) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO UPDATE SET description=EXCLUDED.description`, r.ID, r.ProjectID, r.Category, r.Description, r.Priority, r.Required, r.Source, r.Confidence); err != nil {
			return err
		}
	}
	stack := a.Decision.Selected
	tf, _ := json.Marshal(stack.TestFramework)
	ef, _ := json.Marshal(stack.E2EFramework)
	infra, _ := json.Marshal(stack.Infrastructure)
	if _, err := s.DB.Exec(ctx, `INSERT INTO technology_stacks(id,project_id,language,framework,runtime,package_manager,build_system,test_framework,e2e_framework,database_name,infrastructure) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT(id) DO UPDATE SET framework=EXCLUDED.framework`, a.Decision.ID+"/stack", a.Decision.ProjectID, stack.Language, stack.Framework, stack.Runtime, stack.PackageManager, stack.BuildSystem, tf, ef, stack.Database, infra); err != nil {
		return err
	}
	reasons, _ := json.Marshal(a.Decision.Reasons)
	rejected, _ := json.Marshal(a.Decision.Rejected)
	raw, _ := json.Marshal(a)
	_, err := s.DB.Exec(ctx, `INSERT INTO architecture_decisions(id,project_id,status,selected_id,selected_stack_id,override_value,reasons,rejected,raw_evidence,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status,selected_id=EXCLUDED.selected_id,reasons=EXCLUDED.reasons,rejected=EXCLUDED.rejected,raw_evidence=EXCLUDED.raw_evidence`, a.Decision.ID, a.Decision.ProjectID, a.Decision.Status, a.Decision.SelectedID, a.Decision.ID+"/stack", a.Decision.Override, reasons, rejected, raw, a.Decision.CreatedAt)
	return err
}
