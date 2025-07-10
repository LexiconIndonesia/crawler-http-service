CREATE TABLE IF NOT EXISTS "jobs" (
  "id" varchar(64) NOT NULL PRIMARY KEY,
  "status" varchar(20) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL
);

-- Add useful indices
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
