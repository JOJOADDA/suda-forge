CREATE TABLE IF NOT EXISTS agent_definitions (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  display_name TEXT NOT NULL,
  adapter TEXT NOT NULL,
  version TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  capabilities JSONB NOT NULL DEFAULT '[]',
  runtime_requirements JSONB NOT NULL DEFAULT '{}',
  authentication_method TEXT NOT NULL DEFAULT '',
  configuration JSONB NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS providers (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  base_url TEXT NOT NULL DEFAULT '',
  authentication_type TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  configuration JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS models (
  id TEXT PRIMARY KEY,
  provider_id TEXT NOT NULL REFERENCES providers(id),
  model_id TEXT NOT NULL,
  display_name TEXT NOT NULL,
  context_window INTEGER NOT NULL DEFAULT 0,
  reasoning BOOLEAN NOT NULL DEFAULT false,
  coding BOOLEAN NOT NULL DEFAULT false,
  vision BOOLEAN NOT NULL DEFAULT false,
  tool_use BOOLEAN NOT NULL DEFAULT false,
  structured_output BOOLEAN NOT NULL DEFAULT false,
  local BOOLEAN NOT NULL DEFAULT false,
  remote BOOLEAN NOT NULL DEFAULT true,
  input_cost NUMERIC NOT NULL DEFAULT 0,
  output_cost NUMERIC NOT NULL DEFAULT 0,
  latency_class TEXT NOT NULL DEFAULT '',
  availability TEXT NOT NULL DEFAULT 'UNKNOWN',
  metadata JSONB NOT NULL DEFAULT '{}',
  UNIQUE(provider_id, model_id)
);
CREATE TABLE IF NOT EXISTS model_capabilities (
  model_id TEXT NOT NULL REFERENCES models(id) ON DELETE CASCADE,
  capability TEXT NOT NULL,
  PRIMARY KEY(model_id, capability)
);
CREATE TABLE IF NOT EXISTS agent_configurations (
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL REFERENCES agent_definitions(id),
  name TEXT NOT NULL,
  models JSONB NOT NULL DEFAULT '[]',
  default_model JSONB,
  credential_reference_id TEXT,
  permissions JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS credential_references (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  provider_id TEXT NOT NULL REFERENCES providers(id),
  kind TEXT NOT NULL,
  secret_name TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS agent_adapters (
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL REFERENCES agent_definitions(id) ON DELETE CASCADE,
  version TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  metadata JSONB NOT NULL DEFAULT '{}'
);
CREATE TABLE IF NOT EXISTS agent_sessions (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  agent_id TEXT NOT NULL REFERENCES agent_definitions(id),
  configuration_id TEXT REFERENCES agent_configurations(id),
  provider_id TEXT,
  model_id TEXT,
  runtime_id TEXT NOT NULL,
  working_directory TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS agent_sessions_project_idx ON agent_sessions(project_id, created_at DESC);
CREATE TABLE IF NOT EXISTS agent_session_events (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL REFERENCES agent_sessions(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  normalized JSONB NOT NULL DEFAULT '{}',
  raw JSONB,
  usage JSONB,
  requires_approval BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS agent_session_events_session_idx ON agent_session_events(session_id, created_at ASC);

INSERT INTO agent_definitions (id,name,display_name,adapter,status) VALUES
  ('codex','codex','Codex','codex','AVAILABLE'),
  ('claude-code','claude-code','Claude Code','claude-code','AVAILABLE'),
  ('kimi','kimi','Kimi','kimi','AVAILABLE')
ON CONFLICT (id) DO NOTHING;
INSERT INTO providers (id,name,type,status) VALUES ('custom','Custom','custom','AVAILABLE') ON CONFLICT (id) DO NOTHING;
