package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
)

// Podcast is the parent entity for a podcast.
type Podcast struct {
	// A unique ID for this podcast.
	ID int64 `json:"id"`

	// If this podcast was discovered using the discovery API, this will be the ID of that podcast in the discovery
	// API. This is a string because the discoverIDs are opaque.
	DiscoverID string `json:"discoverId"`

	// The title of the podcast.
	Title string `json:"title"`

	// The description of the podcast.
	Description string `json:"description"`

	// The URL of the title image for the podcast.
	ImageURL string `json:"imageUrl"`

	// If true, the image is external and we should link to it directly rather than as a blob.
	IsImageExternal bool `json:"isImageExternal"`

	// The path on disk to the file where we have the image for this podcast saved. This will be
	// null before we've fetched the image.
	ImagePath *string `json:"-"`

	// The URL of the podcast's RSS feed.
	FeedURL string `json:"feedUrl"`

	// The time that this podcast was last fetched When fetching the RSS feed again, we'll tell the
	// server to only give us new data if it has been changed since this time.
	LastFetchTime time.Time `json:"lastFetchTime"`

	// Episodes is the list of episodes that belong to this podcast.
	Episodes []*Episode `json:"episodes"`
}

// Episode is a single episode in a podcast.
type Episode struct {
	ID               int64     `json:"id"`
	PodcastID        int64     `json:"podcastID"`
	GUID             string    `json:"-"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	DescriptionHTML  bool      `json:"descriptionHtml"`
	ShortDescription string    `json:"shortDescription"`
	PubDate          time.Time `json:"pubDate"`
	MediaURL         string    `json:"mediaUrl"`

	// Position is the offset, in seconds, that the user is at for the episode. This will be null for
	// episodes that don't have any progress (either the user is not subscribed, or they haven't
	// started watching yet).
	Position *int32 `json:"position"`

	// IsComplete will be true if the user has fully listened to this episode.
	IsComplete *bool `json:"isComplete"`

	// LastListenTime is the last time you listened to this episode. Null if you haven't listened yet.
	LastListenTime *time.Time `json:"lastListenTime"`
}

// EpisodeProgress is the state of a single episode of a podcast for a given account.
type EpisodeProgress struct {
	// PodcastID is the ID of the podcast this episode belongs to.
	AccountID int64

	// EpisodeID is the ID of the episode.
	EpisodeID int64

	// Position is the position, in seconds, that playback is up to. Negative means you've completely
	// finished the episode and we mark it "done".
	PositionSecs int32

	// EpisodeComplete is true when the user has marked this episode complete.
	EpisodeComplete bool

	// LastUpdated is the date/time this playback state was actually saved.
	LastUpdated time.Time
}

// SavePodcast saves the given podcast to the store.
func SavePodcast(ctx context.Context, p *Podcast) (int64, error) {
	if p.ID == 0 {
		sql := "INSERT INTO podcasts (discover_id, title, description, image_url, image_path, feed_url, last_fetch_time) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id"
		row := pool.QueryRow(ctx, sql, p.DiscoverID, p.Title, p.Description, p.ImageURL, p.ImagePath, p.FeedURL, time.Time{})
		err := row.Scan(&p.ID)
		return p.ID, err
	} else {
		sql := "UPDATE podcasts SET discover_id=$1, title=$2, description=$3, image_url=$4, image_path=$5, feed_url=$6, last_fetch_time=$7 WHERE id=$8"
		_, err := pool.Exec(ctx, sql, p.DiscoverID, p.Title, p.Description, p.ImageURL, p.ImagePath, p.FeedURL, p.LastFetchTime, p.ID)
		return p.ID, err
	}
}

// SaveEpisode saves the given episode to the data store.
func SaveEpisode(ctx context.Context, p *Podcast, ep *Episode) error {
	var sql = `INSERT INTO episodes
		       (guid, podcast_id, title, description, description_html, short_description, pub_date, media_url)
					 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
					 ON CONFLICT (podcast_id, guid) DO UPDATE SET
					   title=$3, description=$4, description_html=$5, short_description=$6, pub_date=$7, media_url=$8
					 RETURNING id`
	row := pool.QueryRow(ctx, sql, ep.GUID, p.ID, ep.Title, ep.Description, ep.DescriptionHTML, ep.ShortDescription, ep.PubDate, ep.MediaURL)
	var id int64
	if err := row.Scan(&id); err != nil {
		return err
	}

	if ep.ID != 0 && ep.ID != id {
		// TODO: delete the episode, something has gone wrong
		return fmt.Errorf("found existing episode with same GUID but different ID")
	}

	return nil
}

// LoadPodcast returns the podcast with the given ID.
func LoadPodcast(ctx context.Context, podcastID int64) (*Podcast, error) {
	podcast := &Podcast{}
	sql := "SELECT id, discover_id, title, description, image_url, image_path, feed_url, last_fetch_time FROM podcasts WHERE id=$1"
	row := pool.QueryRow(ctx, sql, podcastID)
	if err := row.Scan(&podcast.ID, &podcast.DiscoverID, &podcast.Title, &podcast.Description, &podcast.ImageURL, &podcast.ImagePath, &podcast.FeedURL, &podcast.LastFetchTime); err != nil {
		return nil, fmt.Errorf("error scanning row: %w", err)
	}
	return podcast, nil
}

// LoadPodcastByDiscoverId attempts to load a podcast with the given discover ID.
func LoadPodcastByDiscoverId(ctx context.Context, discoverID string) (*Podcast, error) {
	podcast := &Podcast{}
	stmt := "SELECT id, discover_id, title, description, image_url, image_path, feed_url, last_fetch_time FROM podcasts WHERE discover_id=$1"
	row := pool.QueryRow(ctx, stmt, discoverID)
	if err := row.Scan(&podcast.ID, &podcast.DiscoverID, &podcast.Title, &podcast.Description, &podcast.ImageURL, &podcast.ImagePath, &podcast.FeedURL, &podcast.LastFetchTime); err != nil {
		return nil, fmt.Errorf("error scanning row: %w", err)
	}
	return podcast, nil
}

// LoadEpisode gets the episode with the given ID for the given podcast.
func LoadEpisode(ctx context.Context, p *Podcast, episodeID int64) (*Episode, error) {
	sql := `SELECT
			id, podcast_id, guid, title, description, description_html, short_description, pub_date, media_url
		FROM episodes
		WHERE id = $1`
	row := pool.QueryRow(ctx, sql, episodeID)
	var ep Episode
	if err := row.Scan(&ep.ID, &ep.PodcastID, &ep.GUID, &ep.Title, &ep.Description, &ep.DescriptionHTML, &ep.ShortDescription, &ep.PubDate, &ep.MediaURL); err != nil {
		return nil, fmt.Errorf("error scanning row: %w", err)
	}

	return &ep, nil
}

func populateEpisode(currRow pgx.Row) (*Episode, error) {
	var ep Episode
	err := currRow.Scan(&ep.ID, &ep.PodcastID, &ep.GUID, &ep.Title, &ep.Description, &ep.DescriptionHTML, &ep.ShortDescription, &ep.PubDate, &ep.MediaURL, &ep.Position, &ep.IsComplete, &ep.LastListenTime)
	return &ep, err
}

func populateEpisodes(rows pgx.Rows) ([]*Episode, error) {
	var episodes []*Episode
	for rows.Next() {
		ep, err := populateEpisode(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		episodes = append(episodes, ep)
	}

	return episodes, nil
}

// LoadEpisodes loads all episodes for the given podcast, up to the given limit. If limit is < 0
// then loads all episodes.
func LoadEpisodes(ctx context.Context, podcastID int64, limit int) ([]*Episode, error) {
	sql := `SELECT
	    id, podcast_id, guid, title, description, description_html, short_description, pub_date, media_url, NULL, NULL, NULL
		FROM episodes
		WHERE podcast_id = $1
		ORDER BY pub_date DESC`
	if limit > 0 {
		sql += " LIMIT $2"
	}
	rows, _ := pool.Query(ctx, sql, podcastID, limit)
	defer rows.Close()

	return populateEpisodes(rows)
}

// LoadEpisodesForSubscription gets the episodes to display for the given subscribed account. We'll
// return all episodes that the account has not finished listening to.
func LoadEpisodesForSubscription(ctx context.Context, acct *Account, p *Podcast) ([]*Episode, error) {
	sql := `SELECT
			id, podcast_id, guid, title, description, description_html, short_description, pub_date, media_url, position_secs, episode_complete, episode_progress.last_updated
		FROM episodes
		LEFT OUTER JOIN episode_progress ON episodes.id = episode_progress.episode_id
		WHERE podcast_id = $1
		ORDER BY pub_date DESC`
	rows, _ := pool.Query(ctx, sql, p.ID)
	defer rows.Close()

	return populateEpisodes(rows)
}

// LoadEpisodesNewAndInProgress gets the new and in-progress episodes for the given account. In this
// case, new episodes are ones that don't have any progress at all (and only from the last numDays
// days). And of course, in-progress ones are ones that have progress but are not yet
// marked done. For in-progress episode, we don't just limit them to the last numDays days, we will
// return them all.
func LoadEpisodesNewAndInProgress(ctx context.Context, acct *Account, numDays int) (newEpisodes []*Episode, inProgress []*Episode, err error) {
	sql := `
		SELECT e.id, e.podcast_id, guid, title, description, description_html, short_description,
		       pub_date, media_url, position_secs, episode_complete, ep.last_updated
		FROM episodes e
		INNER JOIN subscriptions s ON s.podcast_id = e.podcast_id
		LEFT JOIN episode_progress ep ON ep.episode_id = e.id AND ep.account_id = s.account_id
		WHERE (pub_date > $1 OR ep.position_secs IS NOT NULL)
		  AND s.account_id = $2
		ORDER BY pub_date DESC`
	rows, _ := pool.Query(ctx, sql, time.Now().Add(-time.Hour*24*time.Duration(numDays)), acct.ID)
	defer rows.Close()

	var episodes []*Episode
	for rows.Next() {
		ep, err := populateEpisode(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("error scanning row: %w", err)
		}

		if ep.Position == nil {
			episodes = append(episodes, ep)
		} else {
			inProgress = append(inProgress, ep)
		}
	}

	return episodes, inProgress, nil
}

func populatePodcasts(rows pgx.Rows) ([]*Podcast, error) {
	var podcasts []*Podcast
	for rows.Next() {
		var podcast Podcast
		if err := rows.Scan(&podcast.ID, &podcast.DiscoverID, &podcast.Title, &podcast.Description, &podcast.ImageURL, &podcast.ImagePath, &podcast.FeedURL, &podcast.LastFetchTime); err != nil {
			return nil, fmt.Errorf("error scanning podcast2: %w", err)
		}

		podcasts = append(podcasts, &podcast)
	}

	return podcasts, nil
}

// LoadPodcasts loads all podcasts from the data store.
// TODO: support paging, filtering, sorting(?), etc.
func LoadPodcasts(ctx context.Context) ([]*Podcast, error) {
	sql := "SELECT id, discover_id, title, description, image_url, image_path, feed_url, last_fetch_time FROM podcasts"
	rows, _ := pool.Query(ctx, sql)
	defer rows.Close()

	return populatePodcasts(rows)
}

// DeletePodcast deletes the podcast with the given ID. This should remove the podcast as well as
// all episodes, subscriptions and so on.
func DeletePodcast(ctx context.Context, podcast *Podcast) error {
	sql := "DELETE FROM podcasts WHERE id=$1"
	_, err := pool.Exec(ctx, sql, podcast.ID)
	return err
}

// SaveEpisodeProgress saves the given EpisodeProgress to the database.
func SaveEpisodeProgress(ctx context.Context, progress *EpisodeProgress) error {
	now := time.Now()
	if progress.LastUpdated.After(now) {
		progress.LastUpdated = time.Now()
	}
	sql := `INSERT INTO episode_progress
		(account_id, episode_id, position_secs, episode_complete, last_updated)
		VALUES ($1, $2, $3, FALSE, $4)
		ON CONFLICT (account_id, episode_id) DO UPDATE SET
		position_secs=$3,
		last_updated=$4`
	_, err := pool.Exec(ctx, sql, progress.AccountID, progress.EpisodeID, progress.PositionSecs, progress.LastUpdated)
	return err
}

func GetMostRecentPlaybackState(ctx context.Context, acct *Account) (*Episode, error) {
	sql := `
		SELECT e.id, e.podcast_id, guid, title, description, description_html, short_description,
		       pub_date, media_url, position_secs, episode_complete, ep.last_updated
		FROM episodes e
		INNER JOIN subscriptions s ON s.podcast_id = e.podcast_id
		INNER JOIN episode_progress ep ON ep.episode_id = e.id AND ep.account_id = s.account_id
		WHERE s.account_id = $1
		ORDER BY ep.last_updated DESC
		LIMIT 1`
	row := pool.QueryRow(ctx, sql, acct.ID)
	return populateEpisode(row)
}
