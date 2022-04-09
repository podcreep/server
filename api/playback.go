package api

import (
	"encoding/json"
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

	// LastUpdated is the time this playback state was recorded by the client. It could be a while
	// ago, if it's taken a while for the client to sync.
	LastUpdated time.Time `json:"lastUpdated"`
}

// handlePlaybackStatePut handles requests to update the playback state of a single episode of a
// single podcast.
func handlePlaybackStatePut(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	acct, err := authenticate(ctx, r)
	if err != nil {
		return apiError("Not authorized", http.StatusUnauthorized)
	}

	playbackState := PlaybackState{}
	if err := json.NewDecoder(r.Body).Decode(&playbackState); err != nil {
		return apiError("Request is not valid", http.StatusBadRequest)
	}

	if !store.IsSubscribed(ctx, acct, playbackState.PodcastID) {
		// You're not subscribed to this episode. We don't save the state if you're not subbed.
		return apiError("No subscription found, can't update state.", http.StatusBadRequest)
	}

	progress := store.EpisodeProgress{
		AccountID:       acct.ID,
		EpisodeID:       playbackState.EpisodeID,
		PositionSecs:    playbackState.Position,
		EpisodeComplete: false, // TODO
		LastUpdated:     playbackState.LastUpdated,
	}
	if err := store.SaveEpisodeProgress(ctx, &progress); err != nil {
		return err
	}

	return nil
}
