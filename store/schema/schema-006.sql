
-- Add the last_updated column, set the default value to current_timestamp and then make it non-null.
ALTER TABLE episode_progress
  ADD COLUMN last_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

