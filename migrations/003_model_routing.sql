CREATE TABLE IF NOT EXISTS model_pricing (
  id BIGSERIAL PRIMARY KEY,
  model_id TEXT NOT NULL REFERENCES models(id) ON DELETE CASCADE,
  input_cost NUMERIC NOT NULL,
  output_cost NUMERIC NOT NULL,
  currency TEXT NOT NULL,
  pricing_unit TEXT NOT NULL,
  effective_date TIMESTAMPTZ NOT NULL,
  UNIQUE(model_id, effective_date)
);
CREATE TABLE IF NOT EXISTS model_health (
  provider_id TEXT PRIMARY KEY REFERENCES providers(id) ON DELETE CASCADE,
  status TEXT NOT NULL,
  latency_ms BIGINT NOT NULL DEFAULT 0,
  rate_limit_remaining INTEGER NOT NULL DEFAULT 0,
  authentication_ok BOOLEAN NOT NULL DEFAULT false,
  checked_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS routing_policies (
  id TEXT PRIMARY KEY,
  scope TEXT NOT NULL,
  scope_id TEXT NOT NULL DEFAULT '',
  policy TEXT NOT NULL,
  local_policy TEXT NOT NULL DEFAULT 'REMOTE_ALLOWED',
  budget_limit NUMERIC NOT NULL DEFAULT 0,
  configuration JSONB NOT NULL DEFAULT '{}',
  UNIQUE(scope, scope_id)
);
CREATE TABLE IF NOT EXISTS routing_decisions (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  agent_id TEXT NOT NULL,
  task_profile JSONB NOT NULL,
  request JSONB NOT NULL,
  selected_provider_id TEXT,
  selected_model_id TEXT,
  decision JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS routing_decisions_project_idx ON routing_decisions(project_id, created_at DESC);
CREATE TABLE IF NOT EXISTS model_usage_events (
  id TEXT PRIMARY KEY,
  session_id TEXT,
  provider_id TEXT NOT NULL,
  model_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  estimated_cost NUMERIC NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL
);
