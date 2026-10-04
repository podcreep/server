
ALTER TABLE episodes
  ADD COLUMN duration_secs NUMBER;

UPDATE episodes SET duration_secs = -1;
