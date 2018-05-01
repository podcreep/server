package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"sort"

	"github.com/podcreep/server/store"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

// UpdatePodcast fetches the feed URL for the given podcast, parses it and updates all of the
// episodes we have stored for the podcast. This method updates the passed-in store.Podcast with
// the latest details.
func UpdatePodcast(ctx context.Context, p *store.Podcast) error {
	// Fetch the RSS feed via a HTTP request.
	fetchClient := urlfetch.Client(ctx)
	resp, err := fetchClient.Get(p.FeedURL)
	if err != nil {
		return fmt.Errorf("error fetching URL: %s: %v", p.FeedURL, err)
	}
	log.Infof(ctx, "Fetched %d bytes, status %d %s", resp.ContentLength, resp.StatusCode, resp.Status)
	if resp.StatusCode != 200 {
		return fmt.Errorf("error fetching URL: %s status=%d", p.FeedURL, resp.StatusCode)
	}

	// Unmarshal the RSS feed into an object we can query.
	var feed Feed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return fmt.Errorf("error unmarshalling response: %v", err)
	}

	for _, item := range feed.Channel.Items {
		pubDate, err := parsePubDate(item.PubDate)
		if err != nil {
			log.Warningf(ctx, "Failed to parse pubdate '%s': %v", item.PubDate, err)
		}

		ep := &store.Episode{
			GUID:        item.GUID,
			MediaURL:    item.Media.URL,
			Title:       item.Title,
			Description: item.Description,
			PubDate:     pubDate,
		}

		// If it's an existing episode, match by GUID.
		for i, existing := range p.Episodes {
			if existing.GUID == ep.GUID {
				ep.ID = existing.ID
				// Remove this element from the podcasts episodes, we'll re-add it later
				p.Episodes = append(p.Episodes[:i], p.Episodes[i+1:]...)
				break
			}
		}

		id, err := store.SaveEpisode(ctx, p, ep)
		if err != nil {
			return fmt.Errorf("error saving episode: %v", err)
		}
		ep.ID = id
		p.Episodes = append(p.Episodes, ep)
	}

	sort.Slice(p.Episodes, func(i, j int) bool {
		return p.Episodes[j].PubDate.Before(p.Episodes[i].PubDate)
	})
	return nil
}
