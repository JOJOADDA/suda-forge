CREATE TABLE IF NOT EXISTS global_cache_blobs (
    artifact_id TEXT PRIMARY KEY REFERENCES shared_artifacts(id) ON DELETE CASCADE,
    checksum TEXT NOT NULL,
    data BYTEA NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS global_cache_blobs_updated_idx ON global_cache_blobs(updated_at);
