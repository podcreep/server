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

type podcastDetails struct {
	store.Podcast

	// IsSubscribed will be true if the current user is subscribed to this podcast.
	IsSubscribed bool `json:"isSubscribed"`
}

type podcastList struct {
	Podcasts []*podcastDetails `json:"podcasts"`
}

// handlePodcastsGet handles requests to view all the podcasts we have in our DB.
// TODO: support filtering, sorting, paging, etc etc.
func handlePodcastsGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	acct, err := authenticate(ctx, r)
	if err != nil {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	podcasts, err := store.LoadPodcasts(ctx)
	if err != nil {
		log.Printf("Error fetching podcasts: %v\n", err)
		http.Error(w, "Error fetching podcasts.", http.StatusInternalServerError)
		return
	}

	subs, err := store.LoadSubscriptionIDs(ctx, acct)
	if err != nil {
		log.Printf("Error fetching subscriptions: %v\n", err)
		http.Error(w, "Error fetching subscriptions.", http.StatusInternalServerError)
		return
	}

	list := podcastList{}
	for _, podcast := range podcasts {
		_, is_subbed := subs[podcast.ID]
		list.Podcasts = append(list.Podcasts, &podcastDetails{*podcast, is_subbed})
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
	details := podcastDetails{*p, false}

	if store.IsSubscribed(ctx, acct, p.ID) {
		details.IsSubscribed = true

		// If they're subscribed, get the episode list for this subscription.
		details.Episodes, err = store.GetEpisodesForSubscription(ctx, acct, p)
		if err != nil {
			log.Printf("Error fetching subscription's episodes: %v", err)
			http.Error(w, "Error fetching episodes.", http.StatusInternalServerError)
			return
		}
	} else {
		// Otherwise, just get the latest 20 episodes
		details.Episodes, err = store.LoadEpisodes(ctx, p.ID, 20)
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

	err = json.NewEncoder(w).Encode(&details)
	if err != nil {
		log.Printf("Error encoding podcasts: %v\n", err)
		http.Error(w, "Error encoding podcasts.", http.StatusInternalServerError)
		return
	}
}
