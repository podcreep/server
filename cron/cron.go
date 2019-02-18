package cron

import (
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
		podcast, err := store.GetPodcast(ctx, p.ID)
		if err != nil {
			log.Printf("Error loading podcasts: %v\n", err)
			continue
		}

		// if the latest episode from this podcast is < 6 hours old, we won't try to re-fetch it.
		var newestEpisodeDate time.Time
		for _, ep := range p.Episodes {
			if ep.PubDate.After(newestEpisodeDate) {
				newestEpisodeDate = ep.PubDate
			}
		}
		log.Printf("Newest episode was last updated: %v", newestEpisodeDate)
		// TODO...

		log.Printf("Updating podcast: %v", podcast)
		rss.UpdatePodcast(ctx, podcast)
	}
}

// Setup is called from server.go and sets up our routes, etc.
func Setup(r *mux.Router) error {
	r.HandleFunc("/cron/check-updates", handleCronCheckUpdates).Methods("GET")

	return nil
}
