package admin

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/schema"
	"github.com/podcreep/server/rss"
	"github.com/podcreep/server/store"
)

func handlePodcastsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Printf("loading podcasts...\n")
	podcasts, err := store.LoadPodcasts(ctx)
	if err != nil {
		log.Printf("error: %v\n", err)
		// TODO: handle error
	}

	render(w, "podcast-list.html", map[string]interface{}{
		"Podcasts": podcasts,
	})
}

func handlePodcastsAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		render(w, "podcast-add.html", nil)
		return
	}

	// It's a POST, so first, grab the URL of the RSS feed.
	r.ParseForm()
	url := r.Form.Get("url")
	log.Printf("Fetching RSS URL: %s\n", url)

	// Fetch the RSS feed via a HTTP request.
	resp, err := http.Get(url)
	if err != nil {
		// TODO: report error more nicely than this
		http.Error(w, fmt.Sprintf("Error fetching URL: %s: %v", url, err), http.StatusInternalServerError)
		return
	}
	log.Printf("Fetched %d bytes, status %d %s, type %s\n", resp.ContentLength, resp.StatusCode, resp.Status, resp.Header.Get("Content-Type"))
	if resp.StatusCode != 200 {
		http.Error(w, fmt.Sprintf("Error fetching URL: %s status=%d", url, resp.StatusCode), http.StatusInternalServerError)
		return
	}

	// Unmarshal the RSS feed into an object we can query.
	var feed rss.Feed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		http.Error(w, fmt.Sprintf("Error unmarshalling response: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Decoded: %v", feed)

	podcast := store.Podcast{
		Title:       feed.Channel.Title,
		Description: feed.Channel.Description,
		ImageURL:    feed.Channel.Image.URL,
		FeedURL:     feed.Channel.Link.Href,
	}

	render(w, "podcast-edit.html", map[string]interface{}{
		"Podcast": podcast,
	})
}

func handlePodcastsEditPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		// TODO: handle error
	}

	var podcast *store.Podcast
	sid := r.Form.Get("id")
	if sid != "" {
		// TODO: load podcast
	} else {
		podcast = &store.Podcast{}
	}

	if err := schema.NewDecoder().Decode(podcast, r.PostForm); err != nil {
		// TODO: handle error
	}

	log.Printf("Saving: %v\n", podcast)
	id, err := store.SavePodcast(ctx, podcast)
	if err != nil {
		// TODO: handle error
	}
	podcast.ID = id

	render(w, "podcast-edit.html", map[string]interface{}{
		"Podcast": podcast,
	})
}
