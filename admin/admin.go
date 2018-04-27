// Package admin contains the backend-management features we use to manage feeds, etc.
package admin

import (
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/podcreep/server/rss"
	"github.com/podcreep/server/store"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

func handleHome(w http.ResponseWriter, r *http.Request) {
	//ctx := appengine.NewContext(r)
	data := struct {
	}{}

	render(w, "index.html", data)
}

func handlePodcastsAdd(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	if r.Method == "GET" {
		render(w, "podcast-add.html", nil)
		return
	}

	// It's a POST, so first, grab the URL of the RSS feed.
	r.ParseForm()
	url := r.Form.Get("url")
	log.Infof(ctx, "Fetching RSS URL: %s", url)

	// Fetch the RSS feed via a HTTP request.
	fetchClient := urlfetch.Client(ctx)
	resp, err := fetchClient.Get(url)
	if err != nil {
		// TODO: report error more nicely than this
		http.Error(w, fmt.Sprintf("Error fetching URL: %s: %v", url, err), http.StatusInternalServerError)
		return
	}
	log.Infof(ctx, "Fetched %d bytes, status %d %s, type %s", resp.ContentLength, resp.StatusCode, resp.Status, resp.Header.Get("Content-Type"))
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
	ctx := appengine.NewContext(r)
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

	log.Infof(ctx, "Saving: %v", podcast)
	id, err := store.SavePodcast(ctx, podcast)
	if err != nil {
		// TODO: handle error
	}
	podcast.ID = id

	render(w, "podcast-edit.html", map[string]interface{}{
		"Podcast": podcast,
	})
}

// Setup is called from server.go and sets up our routes, etc.
func Setup(r *mux.Router) error {
	if err := initTemplates(); err != nil {
		return err
	}

	r.HandleFunc("/admin", handleHome)
	r.HandleFunc("/admin/podcasts/add", handlePodcastsAdd).Methods("GET", "POST")
	r.HandleFunc("/admin/podcasts/edit", handlePodcastsEditPost).Methods("POST")

	return nil
}
