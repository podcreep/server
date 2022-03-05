CREATE TABLE cron (
  id BIGSERIAL NOT NULL PRIMARY KEY,
  job_name TEXT NOT NULL,
  enabled BOOLEAN NOT NULL,
  schedule TEXT NOT NULL,
  next_run TIMESTAMP WITH TIME ZONE
);