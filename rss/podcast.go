package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/podcreep/server/store"
)

// maybeUpdateEpisode will update any new episode we're given.
func maybeUpdateEpisode(ctx context.Context, item Item, p *store.Podcast) error {
	pubDate, err := parsePubDate(item.PubDate)
	if err != nil {
		log.Printf(" - failed to parse pubdate '%s' of '%s': %v\n", item.PubDate, item.Title, err)
		return fmt.Errorf("error parsing date: %v", err)
	}

	// First, check if it's too old entirely. The last episode in the podcast we have is the oldest
	// thing we'll bother to update.
	if len(p.Episodes) > 0 {
		if p.Episodes[len(p.Episodes)-1].PubDate.After(pubDate) {
			return nil
		}
	}

	// If it's in the list of episodes we already have, just ignore it as well (we don't bother
	// to update existing episodes)
	// TODO: is it OK to skip episodes?
	for _, existing := range p.Episodes {
		if existing.GUID == item.GUID {
			return nil
		}
	}

	// OK, seems to be new, add it.
	ep := &store.Episode{
		GUID:        item.GUID,
		MediaURL:    item.Media.URL,
		Title:       item.Title,
		Description: item.Description,
		PubDate:     pubDate,
	}

	log.Printf(" - new episode [%v] '%s', updating", ep.PubDate, ep.Title)
	_, err = store.SaveEpisode(ctx, p, ep)
	if err != nil {
		return fmt.Errorf("error saving episode: %v", err)
	}

	return nil
}

// UpdatePodcast fetches the feed URL for the given podcast, parses it and updates all of the
// episodes we have stored for the podcast. This method updates the passed-in store.Podcast with
// the latest details.
//
// To keep memory usage managable, we use the xml.Decoder interface to decode the XML file in a
// streaming fashion. We also assume the podcast only has the latest handful of episodes -- anything
// older than the oldest episode we have already stored is ignore (if there's no existing episode
// then we assume this is a new podcast and load everything).
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

	// Unmarshal the RSS feed, loading epsiodes as we go. We are extremely forgiving on the XML
	// structure, basically skipping everything that's not an <item> element (where the episode
	// details are stored).
	var item Item
	decoder := xml.NewDecoder(resp.Body)
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				// that's fine, we're at the end of the stream.
				break
			} else {
				// TODO: not just end of stream but maybe some other error?
				log.Printf("Error in top-level decoding: %v", err)
				break
			}
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "item" {
				err := decoder.DecodeElement(&item, &se)
				if err != nil {
					log.Printf("Error parsing item: %v", err)
					return err
				}

				err = maybeUpdateEpisode(ctx, item, p)
				if err != nil {
					log.Printf("Error updating item: %v", err)
					return err
				}
			}
		}
	}

	// Update the last fetch time.
	p.LastFetchTime = time.Now()
	_, err = store.SavePodcast(ctx, p)
	return err
}
