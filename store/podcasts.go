package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/microcosm-cc/bluemonday"
)

var (
	// The policy we use to sanitize the Description's HTML.
	htmlDescriptionPolicy = bluemonday.UGCPolicy()
)

// Podcast is the parent entity for a podcast.
type Podcast struct {
	// A unique ID for this podcast.
	ID int64 `json:"id"`

	// The title of the podcast.
	Title string `json:"title"`

	// The description of the podcast.
	Description string `json:"description"`

	// The URL of the title image for the podcast.
	ImageURL string `json:"imageUrl"`

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
}

// InProgressEpisode is an Episode that contains addition "progress", for episodes that you are
// currently partially-through.
type InProgressEpisode struct {
	Episode

	// Position is the offset, in seconds, that the user is at for the episode.
	Position int32
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
}

// SavePodcast saves the given podcast to the store.
func SavePodcast(ctx context.Context, p *Podcast) (int64, error) {
	sql := "INSERT INTO podcasts (title, description, image_url, feed_url, last_fetch_time) VALUES ($1, $2, $3, $4, $5) RETURNING id"
	row := conn.QueryRow(ctx, sql, p.Title, p.Description, p.ImageURL, p.FeedURL, time.Time{})
	err := row.Scan(&p.ID)
	return p.ID, err
}

// SaveEpisode saves the given episode to the data store.
func SaveEpisode(ctx context.Context, p *Podcast, ep *Episode) error {
	var sql = `INSERT INTO episodes
		       (guid, podcast_id, title, description, description_html, short_description, pub_date, media_url)
					 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
					 ON CONFLICT (podcast_id, guid) DO UPDATE SET
					   title=$3, description=$4, description_html=$5, short_description=$6, pub_date=$7, media_url=$8
					 RETURNING id`
	row := conn.QueryRow(ctx, sql, ep.GUID, p.ID, ep.Title, ep.Description, ep.DescriptionHTML, ep.ShortDescription, ep.PubDate, ep.MediaURL)
	var id int64
	if err := row.Scan(&id); err != nil {
		return err
	}

	if ep.ID != 0 && ep.ID != id {
		// TODO: delete the episode, something has gone wrong
		return fmt.Errorf("Found existing episode with same GUID but different ID")
	}

	return nil
}

// GetPodcast returns the podcast with the given ID.
func GetPodcast(ctx context.Context, podcastID int64) (*Podcast, error) {
	podcast := &Podcast{}
	sql := "SELECT id, title, description, image_url, feed_url, last_fetch_time FROM podcasts WHERE id=$1"
	row := conn.QueryRow(ctx, sql, podcastID)
	if err := row.Scan(&podcast.ID, &podcast.Title, &podcast.Description, &podcast.ImageURL, &podcast.FeedURL, &podcast.LastFetchTime); err != nil {
		return nil, fmt.Errorf("Error scanning row: %w", err)
	}
	return podcast, nil
}

// GetEpisode gets the episode with the given ID for the given podcast.
func GetEpisode(ctx context.Context, p *Podcast, episodeID int64) (*Episode, error) {
	sql := `SELECT
			id, podcast_id, guid, title, description, description_html, short_description, pub_date, media_url
		FROM episodes
		WHERE id = $1`
	row := conn.QueryRow(ctx, sql, episodeID)
	var ep Episode
	if err := row.Scan(&ep.ID, &ep.PodcastID, &ep.GUID, &ep.Title, &ep.Description, &ep.DescriptionHTML, &ep.ShortDescription, &ep.PubDate, &ep.MediaURL); err != nil {
		return nil, fmt.Errorf("Error scanning row: %w", err)
	}

	return &ep, nil
}

func populateEpisode(currRow pgx.Rows) (*Episode, error) {
	var ep Episode
	err := currRow.Scan(&ep.ID, &ep.PodcastID, &ep.GUID, &ep.Title, &ep.Description, &ep.DescriptionHTML, &ep.ShortDescription, &ep.PubDate, &ep.MediaURL)
	return &ep, err
}

func populateInProgressEpisode(currRow pgx.Rows) (*InProgressEpisode, error) {
	var ep InProgressEpisode
	err := currRow.Scan(&ep.ID, &ep.PodcastID, &ep.GUID, &ep.Title, &ep.Description, &ep.DescriptionHTML, &ep.ShortDescription, &ep.PubDate, &ep.MediaURL, &ep.Position)
	return &ep, err
}

func populateEpisodes(rows pgx.Rows) ([]*Episode, error) {
	var episodes []*Episode
	for rows.Next() {
		ep, err := populateEpisode(rows)
		if err != nil {
			return nil, fmt.Errorf("Error scanning row: %w", err)
		}

		episodes = append(episodes, ep)
	}

	return episodes, nil
}

// LoadEpisodes loads all episodes for the given podcast, up to the given limit. If limit is < 0
// then loads all episodes.
// TODO: rename this GetEpisodes
func LoadEpisodes(ctx context.Context, podcastID int64, limit int) ([]*Episode, error) {
	sql := `SELECT
	    id, podcast_id, guid, title, description, description_html, short_description, pub_date, media_url
		FROM episodes
		WHERE podcast_id = $1
		ORDER BY pub_date DESC`
	if limit > 0 {
		sql += " LIMIT $2"
	}
	rows, err := conn.Query(ctx, sql, podcastID, limit)
	if err != nil {
		return nil, fmt.Errorf("Error querying rows: %w", err)
	}
	defer rows.Close()

	return populateEpisodes(rows)
}

// GetEpisodesForSubscription gets the episodes to display for the given subscribed account. We'll
// return all episodes that the account has not finished listening to.
func GetEpisodesForSubscription(ctx context.Context, acct *Account, p *Podcast) ([]*Episode, error) {

	// TODO: check episode_progress

	sql := `SELECT
			id, podcast_id, guid, title, description, description_html, short_description, pub_date, media_url
		FROM episodes
		WHERE podcast_id = $1
		ORDER BY pub_date DESC`
	rows, err := conn.Query(ctx, sql, p.ID)
	if err != nil {
		return nil, fmt.Errorf("Error querying rows: %w", err)
	}
	defer rows.Close()

	return populateEpisodes(rows)
}

// GetEpisodesNewAndInProgress gets the new and in-progress episodes for the given account. In this
// case, new episodes are ones that don't have any progress at all (and only from the last numDays
// days). And of course, in-progress ones are ones that have progress but are not yet
// marked done. For in-progress episode, we don't just limit them to the last numDays days, we will
// return them all.
func GetEpisodesNewAndInProgress(ctx context.Context, acct *Account, numDays int) (newEpisodes []*Episode, inProgress []*InProgressEpisode, err error) {
	sql := `
		SELECT e.id, e.podcast_id, guid, title, description, description_html, short_description,
		       pub_date, media_url, (CASE WHEN position_secs IS NULL THEN 0 ELSE position_secs END)
		FROM episodes e
		INNER JOIN subscriptions s ON s.podcast_id = e.podcast_id
		LEFT JOIN episode_progress ep ON ep.episode_id = e.id AND ep.account_id = s.account_id
		WHERE (pub_date > $1 OR ep.position_secs IS NOT NULL)
		  AND s.account_id = $2
		ORDER BY pub_date DESC`
	rows, err := conn.Query(ctx, sql, time.Now().Add(-time.Hour*24*time.Duration(numDays)), acct.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("Error querying rows: %w", err)
	}
	defer rows.Close()

	var episodes []*Episode
	for rows.Next() {
		ep, err := populateInProgressEpisode(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("Error scanning row: %w", err)
		}

		if ep.Position == 0 {
			episodes = append(episodes, &ep.Episode)
		} else {
			inProgress = append(inProgress, ep)
		}
	}

	return episodes, inProgress, nil
}

// GetEpisodesBetween gets all episodes between the two given dates.
func GetEpisodesBetween(ctx context.Context, p *Podcast, start, end time.Time) ([]*Episode, error) {
	var episodes []*Episode

	//key := datastore.IDKey("podcast", p.ID, nil)
	//q := datastore.NewQuery("episode").Ancestor(key).Filter("PubDate >=", start).Filter("PubDate <=", end).Order("-PubDate")
	//for t := ds.Run(ctx, q); ; {
	//	var ep Episode
	//	key, err := t.Next(&ep)
	//	if err == iterator.Done {
	//		break
	//	} else if err != nil {
	//		return nil, err
	//	}
	//	ep.ID = key.ID
	//	episodes = append(episodes, &ep)
	//}

	return episodes, nil
}

// ClearEpisodes removes all episodes for the given podcast.
func ClearEpisodes(ctx context.Context, podcastID int64) error {
	//for {
	//	// Fetch in batches of 1000
	//	key := datastore.IDKey("podcast", podcastID, nil)
	//	q := datastore.NewQuery("episode").Ancestor(key).KeysOnly().Limit(1000)
	//	keys, err := ds.GetAll(ctx, q, nil)
	//	if err != nil {
	//		return err
	//	}
	//	if len(keys) == 0 {
	//		return nil
	//	}
	//	log.Printf("Got %d episodes to delete (first one: %s)", len(keys), keys[0])

	//	if len(keys) < 100 {
	//		err = ds.DeleteMulti(ctx, keys)
	//		if err != nil {
	//			return err
	//		}
	//		return nil
	//	}

	//	// And delete in batches of 100.
	//	for i := 0; i < len(keys); i += 100 {
	//		err = ds.DeleteMulti(ctx, keys[i:i+100])
	//		if err != nil {
	//			return err
	//		}
	//		log.Printf("Deleted 100 episodes")
	//	}
	//}
	return nil
}

func populatePodcasts(rows pgx.Rows) ([]*Podcast, error) {
	var podcasts []*Podcast
	for rows.Next() {
		var podcast Podcast
		if err := rows.Scan(&podcast.ID, &podcast.Title, &podcast.Description, &podcast.ImageURL, &podcast.FeedURL, &podcast.LastFetchTime); err != nil {
			return nil, fmt.Errorf("Error scanning podcast: %w", err)
		}

		podcasts = append(podcasts, &podcast)
	}

	return podcasts, nil
}

// LoadPodcasts loads all podcasts from the data store.
// TODO: support paging, filtering, sorting(?), etc.
func LoadPodcasts(ctx context.Context) ([]*Podcast, error) {
	sql := "SELECT id, title, description, image_url, feed_url, last_fetch_time FROM podcasts"
	rows, err := conn.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("Error querying podcasts: %w", err)
	}
	defer rows.Close()

	return populatePodcasts(rows)
}

// SaveEpisodeProgress saves the given EpisodeProgress to the database.
func SaveEpisodeProgress(ctx context.Context, progress *EpisodeProgress) error {
	sql := `INSERT INTO episode_progress
		(account_id, episode_id, position_secs, episode_complete)
		VALUES ($1, $2, $3, 0)
		ON CONFLICT (account_id, episode_id) DO UPDATE SET
		position_secs=$3`
	_, err := conn.Exec(ctx, sql, progress.AccountID, progress.EpisodeID, progress.PositionSecs)
	return err
}
