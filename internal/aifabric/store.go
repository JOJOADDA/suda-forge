package aifabric

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ DB *pgxpool.Pool }

func (s Store) SaveRuntime(ctx context.Context, runtime AIRuntime, health RuntimeHealth) error {
	if s.DB == nil {
		return errors.New("ai fabric database unavailable")
	}
	spec := runtime.Spec()
	caps, _ := json.Marshal(runtime.Capabilities())
	cfg, _ := json.Marshal(spec.Configuration)
	_, err := s.DB.Exec(ctx, `INSERT INTO ai_runtimes(id,kind,endpoint,local,auto_start,capabilities,configuration,status,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,now()) ON CONFLICT(id) DO UPDATE SET kind=EXCLUDED.kind,endpoint=EXCLUDED.endpoint,local=EXCLUDED.local,auto_start=EXCLUDED.auto_start,capabilities=EXCLUDED.capabilities,configuration=EXCLUDED.configuration,status=EXCLUDED.status,updated_at=now()`, spec.ID, spec.Kind, spec.Endpoint, spec.Local, spec.AutoStart, caps, cfg, health.Status)
	if err != nil {
		return err
	}
	snapshot, _ := json.Marshal(health)
	_, err = s.DB.Exec(ctx, `INSERT INTO ai_runtime_health(runtime_id,status,snapshot,checked_at) VALUES($1,$2,$3,$4) ON CONFLICT(runtime_id) DO UPDATE SET status=EXCLUDED.status,snapshot=EXCLUDED.snapshot,checked_at=EXCLUDED.checked_at`, spec.ID, health.Status, snapshot, health.LastChecked)
	return err
}

func (s Store) SaveHardware(ctx context.Context, resources HostResources) error {
	if s.DB == nil {
		return errors.New("ai fabric database unavailable")
	}
	raw, _ := json.Marshal(resources)
	id := "hardware_" + resources.DetectedAt.UTC().Format("20060102150405")
	if _, err := s.DB.Exec(ctx, `INSERT INTO ai_hardware_resources(id,snapshot,detected_at) VALUES($1,$2,$3)`, id, raw, resources.DetectedAt); err != nil {
		return err
	}
	for _, gpu := range resources.GPUs {
		g, _ := json.Marshal(gpu)
		if _, err := s.DB.Exec(ctx, `INSERT INTO ai_gpu_resources(id,hardware_id,snapshot) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET snapshot=EXCLUDED.snapshot`, gpu.ID, id, g); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) SaveUsage(ctx context.Context, req InferenceRequest, response InferenceResponse, status string) error {
	if s.DB == nil {
		return errors.New("ai fabric database unavailable")
	}
	_, err := s.DB.Exec(ctx, `INSERT INTO ai_inference_requests(request_id,project_id,agent_id,task_id,model_id,runtime_id,status,started_at,completed_at) VALUES($1,$2,NULLIF($3,''),NULLIF($4,''),$5,$6,$7,now(),now()) ON CONFLICT(request_id) DO UPDATE SET status=EXCLUDED.status,completed_at=now()`, req.RequestID, req.ProjectID, req.AgentID, req.TaskID, req.ModelID, req.RuntimeID, status)
	if err != nil {
		return err
	}
	u := response.Usage
	_, err = s.DB.Exec(ctx, `INSERT INTO ai_inference_usage(request_id,input_tokens,output_tokens,total_tokens,estimated_cost,cpu_millis,gpu_millis,tokens_per_second,duration_ms,local) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,(SELECT local FROM ai_runtimes WHERE id=$10)) ON CONFLICT(request_id) DO UPDATE SET input_tokens=EXCLUDED.input_tokens,output_tokens=EXCLUDED.output_tokens,total_tokens=EXCLUDED.total_tokens,estimated_cost=EXCLUDED.estimated_cost,duration_ms=EXCLUDED.duration_ms`, req.RequestID, u.InputTokens, u.OutputTokens, u.TotalTokens, u.EstimatedCost, u.CPUMillis, u.GPUMillis, u.TokensPerSecond, response.Latency.Milliseconds(), req.RuntimeID)
	return err
}

func (s Store) SaveProjectSettings(ctx context.Context, settings ProjectAISettings) error {
	if s.DB == nil {
		return errors.New("ai fabric database unavailable")
	}
	providers, _ := json.Marshal(settings.AllowedProviders)
	runtimes, _ := json.Marshal(settings.AllowedRuntimes)
	models, _ := json.Marshal(settings.AllowedModels)
	_, err := s.DB.Exec(ctx, `INSERT INTO project_ai_settings(project_id,preferred_agent,preferred_model,routing_policy,privacy_policy,local_only,budget,allowed_providers,allowed_runtimes,allowed_models,updated_at) VALUES($1,NULLIF($2,''),NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10,now()) ON CONFLICT(project_id) DO UPDATE SET preferred_agent=EXCLUDED.preferred_agent,preferred_model=EXCLUDED.preferred_model,routing_policy=EXCLUDED.routing_policy,privacy_policy=EXCLUDED.privacy_policy,local_only=EXCLUDED.local_only,budget=EXCLUDED.budget,allowed_providers=EXCLUDED.allowed_providers,allowed_runtimes=EXCLUDED.allowed_runtimes,allowed_models=EXCLUDED.allowed_models,updated_at=now()`, settings.ProjectID, settings.PreferredAgent, settings.PreferredModel, settings.RoutingPolicy, settings.PrivacyPolicy, settings.LocalOnly, settings.Budget, providers, runtimes, models)
	return err
}
func (s Store) ProjectSettings(ctx context.Context, projectID string) (ProjectAISettings, error) {
	if s.DB == nil {
		return ProjectAISettings{}, errors.New("ai fabric database unavailable")
	}
	var settings ProjectAISettings
	var providers, runtimes, models []byte
	err := s.DB.QueryRow(ctx, `SELECT project_id,COALESCE(preferred_agent,''),COALESCE(preferred_model,''),routing_policy,privacy_policy,local_only,budget,allowed_providers,allowed_runtimes,allowed_models FROM project_ai_settings WHERE project_id=$1`, projectID).Scan(&settings.ProjectID, &settings.PreferredAgent, &settings.PreferredModel, &settings.RoutingPolicy, &settings.PrivacyPolicy, &settings.LocalOnly, &settings.Budget, &providers, &runtimes, &models)
	if err != nil {
		return settings, err
	}
	_ = json.Unmarshal(providers, &settings.AllowedProviders)
	_ = json.Unmarshal(runtimes, &settings.AllowedRuntimes)
	_ = json.Unmarshal(models, &settings.AllowedModels)
	return settings, nil
}
