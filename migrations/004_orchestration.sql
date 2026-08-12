CREATE TABLE IF NOT EXISTS workflows (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  goal TEXT NOT NULL,
  status TEXT NOT NULL,
  execution_strategy TEXT NOT NULL,
  failure_policy TEXT NOT NULL,
  max_parallel_tasks INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS orchestration_tasks (
  id TEXT PRIMARY KEY,
  workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  parent_task_id TEXT REFERENCES orchestration_tasks(id),
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  task_type TEXT NOT NULL,
  priority INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  assigned_agent TEXT,
  routing_request JSONB NOT NULL DEFAULT '{}',
  constraints JSONB NOT NULL DEFAULT '{}',
  retry_policy JSONB NOT NULL DEFAULT '{}',
  deadline TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS task_dependencies (
  task_id TEXT NOT NULL REFERENCES orchestration_tasks(id) ON DELETE CASCADE,
  depends_on_task_id TEXT NOT NULL REFERENCES orchestration_tasks(id) ON DELETE CASCADE,
  PRIMARY KEY(task_id, depends_on_task_id)
);
CREATE TABLE IF NOT EXISTS task_runs (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL REFERENCES orchestration_tasks(id) ON DELETE CASCADE,
  attempt INTEGER NOT NULL,
  status TEXT NOT NULL,
  agent_id TEXT,
  model_id TEXT,
  started_at TIMESTAMPTZ NOT NULL,
  completed_at TIMESTAMPTZ,
  error TEXT
);
CREATE TABLE IF NOT EXISTS task_artifacts (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  task_id TEXT NOT NULL REFERENCES orchestration_tasks(id) ON DELETE CASCADE,
  task_run_id TEXT NOT NULL REFERENCES task_runs(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  path TEXT,
  commit TEXT,
  metadata JSONB NOT NULL DEFAULT '{}'
);
CREATE TABLE IF NOT EXISTS task_approvals (
  id TEXT PRIMARY KEY,
  workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  task_id TEXT NOT NULL REFERENCES orchestration_tasks(id) ON DELETE CASCADE,
  permission TEXT NOT NULL,
  status TEXT NOT NULL,
  requested_at TIMESTAMPTZ NOT NULL,
  resolved_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS task_events (
  id TEXT PRIMARY KEY,
  workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  task_id TEXT REFERENCES orchestration_tasks(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  data JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS orchestration_tasks_workflow_idx ON orchestration_tasks(workflow_id, status);
CREATE INDEX IF NOT EXISTS task_events_workflow_idx ON task_events(workflow_id, created_at ASC);
