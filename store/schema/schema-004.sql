
-- When selecting episodes, we tend to sort by pub_date.
CREATE INDEX IX_episode_pubdate ON episodes (podcast_id, pub_date);
