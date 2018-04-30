package store

import (
	"context"

	"google.golang.org/appengine/datastore"
)

// Podcast is the parent entity for a podcast.
type Podcast struct {
	// A unique ID for this podcast.
	ID int64 `datastore:"-" json:"id"`

	// The title of the podcast.
	Title string `json:"title"`

	// The description of the podcast.
	Description string `json:"description"`

	// The URL of the title image for the podcast.
	ImageURL string `json:"imageUrl"`

	// The URL of the podcast's RSS feed.
	FeedURL string `json:"-"`
}

// SavePodcast saves the given podcast to the store.
func SavePodcast(ctx context.Context, p *Podcast) (int64, error) {
	key := datastore.NewKey(ctx, "podcast", "", p.ID, nil)
	key, err := datastore.Put(ctx, key, p)
	if err != nil {
		return 0, err
	}
	return key.IntID(), nil
}

// LoadPodcasts loads all podcasts from the data store.
// TODO: support paging, filtering, sorting(?), etc.
func LoadPodcasts(ctx context.Context) ([]*Podcast, error) {
	q := datastore.NewQuery("podcast")
	var podcasts []*Podcast
	for t := q.Run(ctx); ; {
		var podcast Podcast
		key, err := t.Next(&podcast)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		podcast.ID = key.IntID()
		podcasts = append(podcasts, &podcast)
	}
	return podcasts, nil
}
