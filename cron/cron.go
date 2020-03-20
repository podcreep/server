package cron

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/rss"
	"github.com/podcreep/server/store"
)

// handleCronCheckUpdates is run every now and then to check for updates to our podcasts. We only
// do one podcast per call to this method (otherwise we tend to run out of memory parsing all that
// XML and stuff).
// To decide which podcast to update, we look at how long it has been since the last update: we
// pick the podcast with the oldest update, as long as it's been more than one hour.
func handleCronCheckUpdates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	podcasts, err := store.LoadPodcasts(ctx)
	if err != nil {
		log.Printf("Error loading podcasts: %v\n", err)
		http.Error(w, "Error loading podcasts.", http.StatusInternalServerError)
		return
	}

	if len(podcasts) == 0 {
		log.Printf("No podcasts.")
		return
	}

	// Sort the podcasts by LastFetchTime, so that the first podcast in the list is the one that
	// we haven't fetched for the longer time.
	sort.Slice(podcasts, func(i, j int) bool {
		return podcasts[i].LastFetchTime.Before(podcasts[j].LastFetchTime)
	})

	p := podcasts[0]
	if p.LastFetchTime.After(time.Now().Add(-1 * time.Hour)) {
		log.Printf("Oldest podcast ('%s') was only updated at %v, not updating again.", p.Title, p.LastFetchTime)
		io.WriteString(w, fmt.Sprintf("No podcasts to update. Oldest podcast, %s, was updated %v", p.Title, p.LastFetchTime))
		return
	}

	log.Printf("Updating podcast %s, LastFetchTime = %v", p.Title, p.LastFetchTime)
	numUpdated, err := updatePodcast(ctx, p)
	if err != nil {
		io.WriteString(w, fmt.Sprintf("Error occurred: %v", err))
	} else {
		io.WriteString(w, fmt.Sprintf("Updated: %s (%d new episodes)", p.Title, numUpdated))
	}
}

func updatePodcast(ctx context.Context, podcast *store.Podcast) (int, error) {
	// The podcast we get here will not have the episodes populated, as it comes from the list.
	// So fetch the episodes manually. We just get the latest 10 episodes. Anything older than this
	// we will ignore entirely.
	episodes, err := store.LoadEpisodes(ctx, podcast.ID, 10)
	if err != nil {
		log.Printf("Error fetching podcast: %v", err)
		return 0, err
	}
	podcast.Episodes = episodes

	return rss.UpdatePodcast(ctx, podcast)
}

// Setup is called from server.go and sets up our routes, etc.
func Setup(r *mux.Router) error {
	r.HandleFunc("/cron/check-updates", handleCronCheckUpdates).Methods("GET")

	return nil
}
