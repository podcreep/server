package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/podcreep/server/store"
)

// UpdatePodcast fetches the feed URL for the given podcast, parses it and updates all of the
// episodes we have stored for the podcast. This method updates the passed-in store.Podcast with
// the latest details.
func UpdatePodcast(ctx context.Context, p *store.Podcast) error {
	log.Printf("Updating podcast: [%d] %s", p.ID, p.Title)

	// Fetch the RSS feed via a HTTP request.
	req, err := http.NewRequest("GET", p.FeedURL, nil)
	if err != nil {
		log.Printf(" - error creating RSS request: %v", err)
		return err
	}

	if !p.LastFetchTime.IsZero() {
		req.Header.Set("If-Modified-Since", p.LastFetchTime.UTC().Format(time.RFC1123))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error fetching URL: %s: %v", p.FeedURL, err)
	}
	log.Printf(" - fetched %d bytes, status %d %s\n", resp.ContentLength, resp.StatusCode, resp.Status)
	if resp.StatusCode == 304 {
		log.Printf(" - podcast has not changed since %s, not updating\n", p.LastFetchTime)
		return nil
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("error fetching URL: %s status=%d", p.FeedURL, resp.StatusCode)
	}

	// Unmarshal the RSS feed into an object we can query.
	var feed Feed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return fmt.Errorf("error unmarshalling response: %v", err)
	}

	totalEpisodesToUpdate := len(feed.Channel.Items)
	log.Printf(" - updating %d episodes (%d existing episodes)\n", totalEpisodesToUpdate, len(p.Episodes))
	for i, item := range feed.Channel.Items {
		pubDate, err := parsePubDate(item.PubDate)
		if err != nil {
			log.Printf(" - [%d of %d] failed to parse pubdate '%s': %v\n", i, totalEpisodesToUpdate, item.PubDate, err)
			continue
		}

		ep := &store.Episode{
			GUID:        item.GUID,
			MediaURL:    item.Media.URL,
			Title:       item.Title,
			Description: item.Description,
			PubDate:     pubDate,
		}

		// If it's an existing episode, match by GUID.
		found := false
		for i, existing := range p.Episodes {
			if existing.GUID == ep.GUID {
				log.Printf(" - [%d of %d] existing episode, not updating (%v %s)\n", i, totalEpisodesToUpdate, ep.PubDate, ep.Title)
				ep.ID = existing.ID
				// Remove this element from the podcasts episodes, we'll re-add it later
				p.Episodes = append(p.Episodes[:i], p.Episodes[i+1:]...)
				found = true
				break
			}
		}

		// Note: we don't update existing episodes, there's not really much point.
		if !found {
			log.Printf(" - [%d of %d] new episode, updating (%v %s)\n", i, totalEpisodesToUpdate, ep.PubDate, ep.Title)
			id, err := store.SaveEpisode(ctx, p, ep)
			if err != nil {
				return fmt.Errorf("error saving episode: %v", err)
			}
			ep.ID = id
		}

		p.Episodes = append(p.Episodes, ep)
	}

	// Update the last fetch time.
	p.LastFetchTime = time.Now()
	store.SavePodcast(ctx, p)

	sort.Slice(p.Episodes, func(i, j int) bool {
		return p.Episodes[j].PubDate.Before(p.Episodes[i].PubDate)
	})
	return nil
}
