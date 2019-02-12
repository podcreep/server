package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/podcreep/server/store"
)

// PlaybackState is the state of a single episode of a podcast.
type PlaybackState struct {
	// PodcastID is the ID of the podcast this episode belongs to.
	PodcastID int64 `json:"podcastID"`

	// EpisodeID is the ID of the episode.
	EpisodeID int64 `json:"episodeID"`

	// Position is the position, in seconds, that playback is up to.
	Position int32 `json:"position"`
}

// handlePlaybackStatePut handles requests to update the playback state of a single episode of a
// single podcast.
func handlePlaybackStatePut(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	acct, err := authenticate(ctx, r)
	if err != nil {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	decoder := json.NewDecoder(r.Body)
	playbackState := PlaybackState{}
	if err := decoder.Decode(&playbackState); err != nil {
		http.Error(w, "Request is not valid", http.StatusBadRequest)
		return
	}

	subs, err := store.GetSubscriptions(ctx, acct)
	if err != nil {
		log.Printf("Error fetching subscriptions: %v\n", err)
		http.Error(w, "Error fetching subscriptions.", http.StatusInternalServerError)
		return
	}

	var sub *store.Subscription
	for _, s := range subs {
		log.Printf("Checking %d==%d &&\n", s.PodcastID, playbackState.PodcastID)
		if s.PodcastID == playbackState.PodcastID {
			sub = s
			break
		}
	}
	if sub == nil {
		// You're not subscribed to this episode. We don't save the state if you're not subbed.
		log.Printf("No subscription found, can't update state.\n")
		return
	}

	found := false
	for i := 0; i < len(sub.Positions); i += 2 {
		if sub.Positions[i] == playbackState.EpisodeID {
			sub.Positions[i+1] = int64(playbackState.Position)
			found = true
			log.Printf("Found existing position, updating ep %d to %d\n",
				playbackState.EpisodeID, playbackState.Position)
		}
	}
	if !found {
		log.Printf("No existing position, making a new one ep %d to %d\n",
			playbackState.EpisodeID, playbackState.Position)
		sub.Positions = append(sub.Positions, playbackState.EpisodeID, int64(playbackState.Position))
	}

	sub, err = store.SaveSubscription(ctx, acct, sub)
	if err != nil {
		log.Printf("Error saving subscription: %v\n", err)
		// But we won't return an error to the client.
	}

	err = json.NewEncoder(w).Encode(sub)
	if err != nil {
		log.Printf("Error encoding subscription: %v\n", err)
		http.Error(w, "Error encoding subscription.", http.StatusInternalServerError)
		return
	}
}
