
CREATE TABLE schema_version (
  version BIGINT NOT NULL
);
INSERT INTO schema_version (version) VALUES (0);


CREATE TABLE accounts (
  id INTEGER NOT NULL PRIMARY KEY,
  cookie TEXT NOT NULL,
  username TEXT NOT NULL,
  password_hash BYTEA NOT NULL
);


CREATE TABLE podcasts (
  id INTEGER NOT NULL PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  image_url TEXT NOT NULL,
  feed_url TEXT NOT NULL,
  last_fetch_time TIMESTAMP NOT NULL
);


CREATE TABLE episodes (
  id INTEGER NOT NULL PRIMARY KEY,
  podcast_id INTEGER NOT NULL,
  guid TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  description_html BOOLEAN NOT NULL,
  short_description TEXT NOT NULL,
  pub_date TIMESTAMP NOT NULL,
  media_url TEXT NOT NULL,

  CONSTRAINT FK_episode_podcast
    FOREIGN KEY (podcast_id) 
    REFERENCES podcasts (id)
    ON DELETE CASCADE
);

CREATE UNIQUE INDEX UIX_episode_guid ON episodes (podcast_id, guid);


CREATE TABLE subscriptions (
  podcast_id INTEGER NOT NULL,
  account_id INTEGER NOT NULL,

  CONSTRAINT FK_subscription_podcast
    FOREIGN KEY (podcast_id) 
    REFERENCES podcasts (id)
    ON DELETE CASCADE,
  CONSTRAINT FK_subscription_account
    FOREIGN KEY (account_id)
    REFERENCES accounts (id)
    ON DELETE CASCADE
);

CREATE UNIQUE INDEX UIX_subscription ON subscriptions (podcast_id, account_id);


CREATE TABLE episode_progress (
  account_id INTEGER NOT NULL,
  episode_id INTEGER NOT NULL,
  position_secs INT NOT NULL,
  episode_complete INT NOT NULL,

  CONSTRAINT FK_episode_progress_account
    FOREIGN KEY (account_id)
    REFERENCES accounts (id)
    ON DELETE CASCADE,
  CONSTRAINT FK_episode_progress_episode
    FOREIGN KEY (episode_id)
    REFERENCES episodes (id)
    ON DELETE CASCADE
);

CREATE UNIQUE INDEX UIX_episode_progress ON episode_progress (account_id, episode_id);
