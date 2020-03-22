package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/rss"
	"github.com/podcreep/server/store"
)

type podcastList struct {
	Podcasts []*store.Podcast `json:"podcasts"`
}

// handlePodcastsGet handles requests to view all the podcasts we have in our DB.
// TODO: support filtering, sorting, paging, etc etc.
func handlePodcastsGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if _, err := authenticate(ctx, r); err != nil {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	podcasts, err := store.LoadPodcasts(ctx)
	if err != nil {
		log.Printf("Error fetching podcasts: %v\n", err)
		http.Error(w, "Error fetching podcasts.", http.StatusInternalServerError)
		return
	}

	list := podcastList{
		Podcasts: podcasts,
	}
	err = json.NewEncoder(w).Encode(&list)
	if err != nil {
		log.Printf("Error encoding podcasts: %v\n", err)
		http.Error(w, "Error encoding podcasts.", http.StatusInternalServerError)
		return
	}
}

// handlePodcastGet handles requests to view a single podcast.
func handlePodcastGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	acct, err := authenticate(ctx, r)
	if err != nil {
		log.Printf("Error authenticating: %v\n", err)
		http.Error(w, "Unauthorized.", http.StatusUnauthorized)
		return
	}

	podcastID, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		log.Printf("Error parsing ID: %s\n", vars["id"])
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	p, err := store.GetPodcast(ctx, podcastID)
	if err != nil {
		log.Printf("Error fetching podcast: %v\n", err)
		http.Error(w, "Error fetching podcast.", http.StatusInternalServerError)
		return
	}

	// Check whether this podcast is subscribed by the current user or not.
	for i := 0; i < len(p.Subscribers); i += 2 {
		if p.Subscribers[i] == acct.ID {
			sub, err := store.GetSubscription(ctx, acct, p.Subscribers[i+1])
			if err != nil {
				log.Printf("Error loading subscription: %v\n", err)
				// Just ignore the error...
			}
			p.Subscription = sub
			break
		}
	}

	if p.Subscription != nil {
		// If they're subscribed, get the episode list for this subscription.
		p.Episodes, err = store.GetEpisodesForSubscription(ctx, p, p.Subscription)
		if err != nil {
			log.Printf("Error fetching subscription's episodes: %v", err)
			http.Error(w, "Error fetching episodes.", http.StatusInternalServerError)
			return
		}
	} else {
		// Otherwise, just get the latest 20 episodes
		p.Episodes, err = store.LoadEpisodes(ctx, p.ID, 20)
		if err != nil {
			log.Printf("Error fetching latest episodes: %v", err)
			http.Error(w, "Error fetching episodes.", http.StatusInternalServerError)
			return
		}
	}

	if r.URL.Query().Get("refresh") == "1" {
		// They've asked us explicitly to refresh the podcast (and all it's episodes), so do that
		// first before fetching the podcast.
		if _, err := rss.UpdatePodcast(ctx, p, false); err != nil {
			log.Printf("Erroring updating podcast: %v\n", err)
			// Note: we just keep going, assuming the podcast didn't change.
		}
	}

	err = json.NewEncoder(w).Encode(&p)
	if err != nil {
		log.Printf("Error encoding podcasts: %v\n", err)
		http.Error(w, "Error encoding podcasts.", http.StatusInternalServerError)
		return
	}
}
