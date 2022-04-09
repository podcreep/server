
-- Add the last_updated column, set the default value to NOW() and then make it non-null.
ALTER TABLE episode_progress
  ADD COLUMN last_updated TIMESTAMP WITH TIME ZONE;

UPDATE episode_progress SET last_updated = NOW();

ALTER TABLE episode_progress
  ALTER COLUMN last_updated SET NOT NULL;

