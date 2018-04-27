// Package store contains methods and structures that we use to persist our data in the data store.
package store

import (
	"context"

	"google.golang.org/appengine/datastore"
)

// Podcast is the parent entity for a podcast.
type Podcast struct {
	// A unique ID for this podcast.
	ID int64 `datastore:"-"`

	// The title of the podcast.
	Title string

	// The description of the podcast.
	Description string

	// The URL of the title image for the podcast.
	ImageURL string

	// The URL of the podcast's RSS feed.
	FeedURL string
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
