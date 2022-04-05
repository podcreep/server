
ALTER TABLE episode_progress ALTER episode_complete TYPE BOOL USING CASE WHEN episode_complete=0 THEN FALSE ELSE TRUE END;
