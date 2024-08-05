
ALTER TABLE podcasts
  ADD COLUMN discover_id TEXT;

UPDATE podcasts SET discover_id = '';
