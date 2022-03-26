package api

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/podcreep/server/rss"
	"github.com/podcreep/server/store"
	"golang.org/x/image/draw"
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
func handlePodcastsGet(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	acct, err := authenticate(ctx, r)
	if err != nil {
		return apiError("Not authorized", http.StatusUnauthorized)
	}

	podcasts, err := store.LoadPodcasts(ctx)
	if err != nil {
		return err
	}

	subs, err := store.LoadSubscriptionIDs(ctx, acct)
	if err != nil {
		return err
	}

	list := podcastList{}
	for _, podcast := range podcasts {
		_, is_subbed := subs[podcast.ID]
		list.Podcasts = append(list.Podcasts, &podcastDetails{*podcast, is_subbed})
	}
	err = json.NewEncoder(w).Encode(&list)
	if err != nil {
		return err
	}

	return nil
}

// handlePodcastGet handles requests to view a single podcast.
func handlePodcastGet(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	vars := mux.Vars(r)

	acct, err := authenticate(ctx, r)
	if err != nil {
		return apiError("Unauthorized.", http.StatusUnauthorized)
	}

	podcastID, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		return err
	}

	p, err := store.LoadPodcast(ctx, podcastID)
	if err != nil {
		return err
	}
	details := podcastDetails{*p, false}

	if store.IsSubscribed(ctx, acct, p.ID) {
		details.IsSubscribed = true

		// If they're subscribed, get the episode list for this subscription.
		details.Episodes, err = store.LoadEpisodesForSubscription(ctx, acct, p)
		if err != nil {
			return err
		}
	} else {
		// Otherwise, just get the latest 20 episodes
		details.Episodes, err = store.LoadEpisodes(ctx, p.ID, 20)
		if err != nil {
			return err
		}
	}

	if r.URL.Query().Get("refresh") == "1" {
		// They've asked us explicitly to refresh the podcast (and all it's episodes), so do that
		// first before fetching the podcast.
		if _, err := rss.UpdatePodcast(ctx, p, 0 /*flags*/); err != nil {
			log.Printf("Erroring updating podcast: %v\n", err)
			// Note: we just keep going, assuming the podcast didn't change.
		}
	}

	err = json.NewEncoder(w).Encode(&details)
	if err != nil {
		return err
	}

	return nil
}

// handlePodcastIconGet handles requests to view a podcast's icon.
func handlePodcastIconGet(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	vars := mux.Vars(r)

	podcastID, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		return err
	}

	p, err := store.LoadPodcast(ctx, podcastID)
	if err != nil {
		return err
	}

	if p.ImagePath == nil {
		// We haven't downloaded the image yet.
		return apiError("Image doesn't exist", http.StatusNotFound)
	}

	// TODO: compare vars["sha1"] (the requests SHA1) with the latest SHA1

	// We put the SHA1 in the filename, so if it ever changes, the URL would be different. Given that,
	// browsers are free to cache this response forever.
	// TODO: handle If-Modified-Since?
	w.Header().Add("Cache-Control", "public, max-age=31536000")

	if r.URL.Query().Get("width") != "" && r.URL.Query().Get("height") != "" {
		// They want a specific size, let's give them a a specific size.
		width, _ := strconv.Atoi(r.URL.Query().Get("width"))
		height, _ := strconv.Atoi(r.URL.Query().Get("height"))
		if width > 0 && height > 0 {
			file, err := os.Open(*p.ImagePath)
			if err != nil {
				return err
			}

			img, _, err := image.Decode(file)
			if err != nil {
				return fmt.Errorf("Error decoding image %s: %w", *p.ImagePath, err)
			}
			resized := image.NewRGBA(image.Rect(0, 0, width, height))
			draw.CatmullRom.Scale(resized, resized.Rect, img, img.Bounds(), draw.Over, nil)
			w.Header().Add("Content-Type", "image/png")
			png.Encode(w, resized)
			return nil
		}
	}

	http.ServeFile(w, r, *p.ImagePath)
	return nil
}
