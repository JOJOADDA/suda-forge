CREATE TABLE IF NOT EXISTS verification_runs (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  workflow_id TEXT,
  task_id TEXT,
  task_run_id TEXT,
  status TEXT NOT NULL,
  started_at TIMESTAMPTZ NOT NULL,
  completed_at TIMESTAMPTZ,
  summary JSONB NOT NULL DEFAULT '{}'::jsonb,
  profile JSONB NOT NULL DEFAULT '{}'::jsonb,
  state JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS idx_verification_runs_project ON verification_runs(project_id, started_at DESC);
CREATE TABLE IF NOT EXISTS verification_checks (
  id TEXT NOT NULL,
  verification_run_id TEXT NOT NULL REFERENCES verification_runs(id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  name TEXT NOT NULL,
  required BOOLEAN NOT NULL,
  status TEXT NOT NULL,
  configuration JSONB NOT NULL DEFAULT '{}'::jsonb,
  evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
  failure JSONB,
  started_at TIMESTAMPTZ NOT NULL,
  completed_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (verification_run_id, id)
);
CREATE TABLE IF NOT EXISTS verification_failures (
  id TEXT PRIMARY KEY,
  verification_run_id TEXT NOT NULL REFERENCES verification_runs(id) ON DELETE CASCADE,
  check_id TEXT NOT NULL,
  failure_type TEXT NOT NULL,
  report JSONB NOT NULL
);
CREATE TABLE IF NOT EXISTS verification_repairs (
  id TEXT PRIMARY KEY,
  verification_run_id TEXT NOT NULL REFERENCES verification_runs(id) ON DELETE CASCADE,
  attempt INTEGER NOT NULL,
  status TEXT NOT NULL,
  plan JSONB NOT NULL,
  error TEXT,
  started_at TIMESTAMPTZ NOT NULL,
  completed_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS verification_artifacts (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  task_id TEXT,
  verification_run_id TEXT NOT NULL REFERENCES verification_runs(id) ON DELETE CASCADE,
  check_id TEXT,
  kind TEXT NOT NULL,
  path TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);
