package cron

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/rss"
	"github.com/podcreep/server/store"
)

func handleCronCheckUpdates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	podcasts, err := store.LoadPodcasts(ctx)
	if err != nil {
		log.Printf("Error loading podcasts: %v\n", err)
		http.Error(w, "Error loading podcasts.", http.StatusInternalServerError)
		return
	}

	log.Printf("Got %d podcasts.\n", len(podcasts))
	for _, p := range podcasts {
		updatePodcast(ctx, p)
	}
}

func updatePodcast(ctx context.Context, podcast *store.Podcast) {
	// if the latest episode from this podcast is < 4 hours old, we won't try to re-fetch it.
	// TODO: do it.
	var newestEpisodeDate time.Time
	for _, ep := range podcast.Episodes {
		if ep.PubDate.After(newestEpisodeDate) {
			newestEpisodeDate = ep.PubDate
		}
	}
	log.Printf("Newest episode was last updated: %v", newestEpisodeDate)

	// The podcast we get here will not have the episodes populates, as it comes from the list.
	// So fetch the episodes manually (this will get all of the episodes we have stored)
	episodes, err := store.LoadEpisodes(ctx, podcast.ID)
	if err != nil {
		log.Printf("Error fetching podcast: %v", err)
		return
	}
	podcast.Episodes = episodes

	err = rss.UpdatePodcast(ctx, podcast)
	if err != nil {
		log.Printf(" !! %v", err)
	}
}

// Setup is called from server.go and sets up our routes, etc.
func Setup(r *mux.Router) error {
	r.HandleFunc("/cron/check-updates", handleCronCheckUpdates).Methods("GET")

	return nil
}
