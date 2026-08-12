CREATE TABLE IF NOT EXISTS ai_runtimes (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  endpoint TEXT NOT NULL,
  local BOOLEAN NOT NULL DEFAULT TRUE,
  auto_start BOOLEAN NOT NULL DEFAULT FALSE,
  capabilities JSONB NOT NULL DEFAULT '{}'::jsonb,
  configuration JSONB NOT NULL DEFAULT '{}'::jsonb,
  status TEXT NOT NULL DEFAULT 'OFFLINE',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS ai_runtime_health (
  runtime_id TEXT PRIMARY KEY REFERENCES ai_runtimes(id) ON DELETE CASCADE,
  status TEXT NOT NULL,
  snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  checked_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS ai_hardware_resources (
  id TEXT PRIMARY KEY,
  snapshot JSONB NOT NULL,
  detected_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS ai_gpu_resources (
  id TEXT PRIMARY KEY,
  hardware_id TEXT NOT NULL,
  snapshot JSONB NOT NULL
);
CREATE TABLE IF NOT EXISTS ai_model_installations (
  runtime_id TEXT NOT NULL REFERENCES ai_runtimes(id) ON DELETE CASCADE,
  model_id TEXT NOT NULL,
  status TEXT NOT NULL,
  source TEXT,
  quantization TEXT,
  revision TEXT,
  resource_requirements JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY(runtime_id, model_id)
);
CREATE TABLE IF NOT EXISTS ai_model_health (
  runtime_id TEXT NOT NULL REFERENCES ai_runtimes(id) ON DELETE CASCADE,
  model_id TEXT NOT NULL,
  status TEXT NOT NULL,
  snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  checked_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY(runtime_id, model_id)
);
CREATE TABLE IF NOT EXISTS ai_inference_requests (
  request_id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  agent_id TEXT,
  task_id TEXT,
  provider_id TEXT,
  model_id TEXT NOT NULL,
  runtime_id TEXT NOT NULL,
  status TEXT NOT NULL,
  started_at TIMESTAMPTZ NOT NULL,
  completed_at TIMESTAMPTZ,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE TABLE IF NOT EXISTS ai_inference_usage (
  request_id TEXT PRIMARY KEY REFERENCES ai_inference_requests(request_id) ON DELETE CASCADE,
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  total_tokens INTEGER NOT NULL DEFAULT 0,
  estimated_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
  cpu_millis BIGINT NOT NULL DEFAULT 0,
  gpu_millis BIGINT NOT NULL DEFAULT 0,
  tokens_per_second DOUBLE PRECISION NOT NULL DEFAULT 0,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  local BOOLEAN NOT NULL DEFAULT TRUE
);
CREATE TABLE IF NOT EXISTS project_ai_settings (
  project_id TEXT PRIMARY KEY,
  preferred_agent TEXT,
  preferred_model TEXT,
  routing_policy TEXT NOT NULL DEFAULT 'BALANCED',
  privacy_policy TEXT NOT NULL DEFAULT 'PUBLIC',
  local_only BOOLEAN NOT NULL DEFAULT FALSE,
  budget DOUBLE PRECISION NOT NULL DEFAULT 0,
  allowed_providers JSONB NOT NULL DEFAULT '[]'::jsonb,
  allowed_runtimes JSONB NOT NULL DEFAULT '[]'::jsonb,
  allowed_models JSONB NOT NULL DEFAULT '[]'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS ai_model_capability_checks (
  id TEXT PRIMARY KEY,
  runtime_id TEXT NOT NULL,
  model_id TEXT NOT NULL,
  capability TEXT NOT NULL,
  status TEXT NOT NULL,
  evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
  checked_at TIMESTAMPTZ NOT NULL
);
