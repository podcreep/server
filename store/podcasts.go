package store

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/iterator"
)

// Podcast is the parent entity for a podcast.
type Podcast struct {
	// A unique ID for this podcast.
	ID int64 `datastore:"-" json:"id"`

	// The title of the podcast.
	Title string `json:"title"`

	// The description of the podcast.
	Description string `datastore:",noindex" json:"description"`

	// The URL of the title image for the podcast.
	ImageURL string `datastore:",noindex" json:"imageUrl"`

	// The URL of the podcast's RSS feed.
	FeedURL string `datastore:",noindex" json:"feedUrl"`

	// The time that this podcast was last fetched When fetching the RSS feed again, we'll tell the
	// server to only give us new data if it has been changed since this time.
	LastFetchTime time.Time `datastore:",noindex" json:"lastFetchTime"`

	// Subscribers is the list of account IDs that are subscribed to this podcast. Each entry is
	// actually two numbers, the first is the account ID and the second is the subscription ID. This
	// is just because we can't easily do maps in the data store.
	Subscribers []int64 `json:"-"`

	// Episodes is the list of episodes that belong to this podcast.
	Episodes []*Episode `datastore:"-" json:"episodes"`

	// If non-nil, this is the subscription that the current user has to this podcast.
	Subscription *Subscription `datastore:"-" json:"subscription"`
}

// Episode is a single episode in a podcast.
type Episode struct {
	ID   int64  `datastore:"-" json:"id"`
	GUID string `json:"-"`

	Title            string    `datastore:",noindex" json:"title"`
	Description      string    `datastore:",noindex" json:"description"`
	DescriptionHTML  bool      `datastore:",noindex" json:"descriptionHtml"`
	ShortDescription string    `datastore:",noindex" json:"shortDescription"`
	PubDate          time.Time `json:"pubDate"`
	MediaURL         string    `datastore:",noindex" json:"mediaUrl"`
}

// SavePodcast saves the given podcast to the store.
func SavePodcast(ctx context.Context, p *Podcast) (int64, error) {
	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		return 0, err
	}

	key := datastore.IDKey("podcast", p.ID, nil)
	key, err = ds.Put(ctx, key, p)
	if err != nil {
		return 0, err
	}

	return key.ID, nil
}

// SaveEpisode saves the given episode to the data store.
func SaveEpisode(ctx context.Context, p *Podcast, ep *Episode) (int64, error) {
	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		return 0, err
	}

	pkey := datastore.IDKey("podcast", p.ID, nil)
	key := datastore.IDKey("episode", ep.ID, pkey)
	key, err = ds.Put(ctx, key, ep)
	if err != nil {
		return 0, err
	}
	return key.ID, nil
}

// GetPodcast returns the podcast with the given ID.
func GetPodcast(ctx context.Context, podcastID int64) (*Podcast, error) {
	podcast := &Podcast{}

	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		return nil, err
	}

	key := datastore.IDKey("podcast", podcastID, nil)

	err = ds.Get(ctx, key, podcast)
	if err != nil {
		return nil, err
	}
	podcast.ID = key.ID

	q := datastore.NewQuery("episode").Ancestor(key).Order("-PubDate").Limit(20)
	for t := ds.Run(ctx, q); ; {
		var ep Episode
		key, err := t.Next(&ep)
		if err == iterator.Done {
			break
		} else if err != nil {
			return nil, err
		}
		ep.ID = key.ID
		podcast.Episodes = append(podcast.Episodes, &ep)
	}

	return podcast, nil
}

// LoadEpisodes loads all episodes for the given podcast, up to the given limit. If limit is < 0
// then loads all episodes.
func LoadEpisodes(ctx context.Context, podcastID int64, limit int) ([]*Episode, error) {
	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		return nil, err
	}

	var episodes []*Episode

	key := datastore.IDKey("podcast", podcastID, nil)
	q := datastore.NewQuery("episode").Ancestor(key).Order("-PubDate")
	if limit > 0 {
		q = q.Limit(limit)
	}
	for t := ds.Run(ctx, q); ; {
		var ep Episode
		key, err := t.Next(&ep)
		if err == iterator.Done {
			break
		} else if err != nil {
			return nil, err
		}
		ep.ID = key.ID
		episodes = append(episodes, &ep)
	}

	return episodes, err
}

// ClearEpisodes removes all episodes for the given podcast.
func ClearEpisodes(ctx context.Context, podcastID int64) error {
	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		return err
	}

	for {
		key := datastore.IDKey("podcast", podcastID, nil)
		q := datastore.NewQuery("episode").Ancestor(key).KeysOnly()
		count, err := ds.Count(ctx, q)
		if err != nil {
			log.Printf("%d total entities matching query.", count)
		}

		// Fetch in batches of 1000.
		keys, err := ds.GetAll(ctx, q.Limit(1000), nil)
		if err != nil {
			return err
		}
		if len(keys) == 0 {
			return nil
		}
		log.Printf("Got %d (of %d total) episodes to delete (first one: %s)", len(keys), count, keys[0])

		if len(keys) < 100 {
			err = ds.DeleteMulti(ctx, keys)
			if err != nil {
				return err
			}
			return nil
		}

		// And delete in batches of 100.
		for i := 0; i < len(keys); i += 100 {
			err = ds.DeleteMulti(ctx, keys[i:i+100])
			if err != nil {
				return err
			}
			log.Printf("Deleted 100 episodes")
		}
	}
}

// LoadEpisodeGUIDs loads a map of GUID->ID for all the episodes of the given podcast.
func LoadEpisodeGUIDs(ctx context.Context, podcastID int64) (map[string]int64, error) {
	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		return nil, err
	}

	result := make(map[string]int64)

	key := datastore.IDKey("podcast", podcastID, nil)
	q := datastore.NewQuery("episode").Ancestor(key).Project("GUID")
	for t := ds.Run(ctx, q); ; {
		var ep Episode
		key, err := t.Next(&ep)
		if err == iterator.Done {
			break
		} else if err != nil {
			return nil, err
		}

		result[ep.GUID] = key.ID
	}

	return result, nil
}

// LoadPodcasts loads all podcasts from the data store.
// TODO: support paging, filtering, sorting(?), etc.
func LoadPodcasts(ctx context.Context) ([]*Podcast, error) {
	ds, err := datastore.NewClient(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("Error creating datastore client: %v", err)
	}

	q := datastore.NewQuery("podcast")
	var podcasts []*Podcast
	for t := ds.Run(ctx, q); ; {
		var podcast Podcast
		key, err := t.Next(&podcast)
		if err == iterator.Done {
			break
		} else if err != nil {
			return nil, err
		}
		podcast.ID = key.ID
		podcasts = append(podcasts, &podcast)
	}
	return podcasts, nil
}
