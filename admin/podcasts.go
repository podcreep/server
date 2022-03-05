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

func handlePodcastsList(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	log.Printf("loading podcasts...\n")
	podcasts, err := store.LoadPodcasts(ctx)
	if err != nil {
		return err
	}

	return render(w, "podcast/list.html", map[string]interface{}{
		"Podcasts": podcasts,
	})
}

func handlePodcastsAdd(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return render(w, "podcast/add.html", nil)
	}

	// It's a POST, so first, grab the URL of the RSS feed.
	r.ParseForm()
	url := r.Form.Get("url")
	log.Printf("Fetching RSS URL: %s\n", url)

	// Fetch the RSS feed via a HTTP request.
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error fetching URL: %s: %v", url, err)
	}
	log.Printf("Fetched %d bytes, status %d %s, type %s\n", resp.ContentLength, resp.StatusCode, resp.Status, resp.Header.Get("Content-Type"))
	if resp.StatusCode != 200 {
		return fmt.Errorf("error fetching URL: %s status=%d", url, resp.StatusCode)
	}

	// Unmarshal the RSS feed into an object we can query.
	var feed rss.Feed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return fmt.Errorf("error unmarshalling response: %v", err)
	}

	podcast := store.Podcast{
		Title:       feed.Channel.Title,
		Description: feed.Channel.Description,
		ImageURL:    feed.Channel.Image.URL,
		FeedURL:     feed.Channel.Link.Href,
	}

	return render(w, "podcast/edit.html", map[string]interface{}{
		"Podcast": podcast,
	})
}

func handlePodcastsEditPost(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		return httpError(fmt.Sprintf("error parsing form: %v", err), http.StatusBadRequest)
	}

	var podcast *store.Podcast
	sid := r.Form.Get("id")
	if sid != "" {
		// TODO: load podcast
	} else {
		podcast = &store.Podcast{}
	}

	if err := schema.NewDecoder().Decode(podcast, r.PostForm); err != nil {
		return httpError(fmt.Sprintf("error decoding form: %v", err), http.StatusBadRequest)
	}

	log.Printf("Saving: %v\n", podcast)
	id, err := store.SavePodcast(ctx, podcast)
	if err != nil {
		return err
	}
	podcast.ID = id

	return render(w, "podcast/edit.html", map[string]interface{}{
		"Podcast": podcast,
	})
}
