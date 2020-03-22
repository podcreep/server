package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/podcreep/server/store"
)

// PlaybackState is the state of a single episode of a podcast.
type PlaybackState struct {
	// PodcastID is the ID of the podcast this episode belongs to.
	PodcastID int64 `json:"podcastID"`

	// EpisodeID is the ID of the episode.
	EpisodeID int64 `json:"episodeID"`

	// Position is the position, in seconds, that playback is up to. Negative means you've completely
	// finished the episode and we mark it "done".
	Position int32 `json:"position"`

	// UpdateDoneCutoffDate is true when the user wants to mark this and all older episodes as "done".
	UpdateDoneCutoffDate bool `json:"updateDoneCutoffDate"`
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
			if playbackState.Position < 0 {
				// Any number less than zero becomes -1.
				sub.Positions[i+1] = -1
			} else {
				sub.Positions[i+1] = int64(playbackState.Position)
			}
			found = true
		}
	}
	if !found {
		sub.Positions = append(sub.Positions, playbackState.EpisodeID, int64(playbackState.Position))
	}

	if playbackState.Position < 0 {
		// OK this episode is done. Let's check if we need to update DoneCutoffDate in the subscription.
		p, err := store.GetPodcast(ctx, playbackState.PodcastID)
		if err != nil {
			log.Printf("Error fetching podcast: %v", err)
			http.Error(w, "Error fetching podcast.", http.StatusInternalServerError)
			return
		}

		ep, err := store.GetEpisode(ctx, p, playbackState.EpisodeID)
		if err != nil {
			log.Printf("Error fetching episode: %v", err)
			http.Error(w, "Error fetching episode.", http.StatusInternalServerError)
			return
		}

		if playbackState.UpdateDoneCutoffDate {
			// If we've been asked explicitly to update the done cutoff date, then do it (actually, unless
			// the existing cutoff is newer...)
			doneCutoffDate := time.Unix(sub.DoneCutoffDate, 0)
			if doneCutoffDate.Before(ep.PubDate) {
				sub.DoneCutoffDate = ep.PubDate.Unix()
			}
		} else if sub.DoneCutoffDate > 0 {
			// Check if there's any non-done episodes between this one and the existing DoneCutoffDate
			// TODO and stuff
		}
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
