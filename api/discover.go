package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/discover"
	"github.com/podcreep/server/store"
)

// handleDiscoverTrendingGet handles POST requests for /api/discover. It allows clients to search for postcasts.
func handleDiscoverTrendingGet(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	_, err := authenticate(ctx, r)
	if err != nil {
		return apiError("Unauthorized.", http.StatusUnauthorized)
	}

	podcasts, err := discover.FetchTrending()
	if err != nil {
		// I'm not sure what the best HTTP status to return here is?
		return err
	}

	// Translate to our podcastDetails that the client understands.
	list := podcastList{}
	for _, podcast := range podcasts {
		list.Podcasts = append(list.Podcasts, &podcastDetails{
			Podcast: store.Podcast{
				ID:              podcast.ID,
				Title:           podcast.Title,
				Description:     podcast.Description,
				ImageURL:        podcast.ImageUrl,
				IsImageExternal: true, // Discover podcasts haven't been saved yet, so the icons are external.
				FeedURL:         podcast.Url,
			},
		})
	}
	err = json.NewEncoder(w).Encode(&list)
	if err != nil {
		return err
	}

	return nil
}

func handleDiscoverPodcastGet(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	vars := mux.Vars(r)

	_, err := authenticate(ctx, r)
	if err != nil {
		return apiError("Unauthorized.", http.StatusUnauthorized)
	}

	podcastID, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		return err
	}

	podcast, err := discover.FetchPodcast(podcastID)
	if err != nil {
		// I'm not sure what the best HTTP status to return here is?
		return err
	}

	// Translate to our podcastDetails that the client understands.
	details := &podcastDetails{
		Podcast: store.Podcast{
			ID:              podcast.ID,
			Title:           podcast.Title,
			Description:     podcast.Description,
			ImageURL:        podcast.ImageUrl,
			IsImageExternal: true, // Discover podcasts haven't been saved yet, so the icons are external.
			FeedURL:         podcast.Url,
			// TODO: link?
		},
	}
	err = json.NewEncoder(w).Encode(&details)
	if err != nil {
		return err
	}

	return nil
}
