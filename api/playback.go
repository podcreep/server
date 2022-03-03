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

	if !store.IsSubscribed(ctx, acct, playbackState.PodcastID) {
		// You're not subscribed to this episode. We don't save the state if you're not subbed.
		log.Printf("No subscription found, can't update state.\n")
		return
	}

	progress := store.EpisodeProgress{
		AccountID:       acct.ID,
		EpisodeID:       playbackState.EpisodeID,
		PositionSecs:    playbackState.Position,
		EpisodeComplete: false, // TODO
	}
	// TODO: update playback state
	if err := store.SaveEpisodeProgress(ctx, &progress); err != nil {
		log.Printf("Error saving progress. %v", err)
		return
	}
}
