ALTER TABLE autonomous_loop_plans
    ADD COLUMN IF NOT EXISTS goal TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS current_stage TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS results JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS error TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS worker_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS lease_until TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS autonomous_loop_plans_runnable_idx
    ON autonomous_loop_plans(status, updated_at)
    WHERE status IN ('RUNNING', 'BLOCKED', 'FAILED');
