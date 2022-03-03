package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/podcreep/server/store"

	"github.com/microcosm-cc/bluemonday"
)

var (
	// An empty policy will strip all HTML tags, which is what we actually want.
	htmlPolicy = bluemonday.NewPolicy()
)

func updateEpisode(ctx context.Context, item Item, p *store.Podcast) error {
	pubDate, err := parsePubDate(item.PubDate)
	if err != nil {
		return fmt.Errorf("error parsing date: %v", err)
	}

	var ep = store.Episode{
		GUID:             item.GUID,
		MediaURL:         item.Media.URL,
		Title:            item.Title,
		Description:      item.Description,
		DescriptionHTML:  false,
		ShortDescription: item.Description,
		PubDate:          pubDate,
	}

	if item.EncodedDescription != "" {
		ep.Description = item.EncodedDescription
		ep.DescriptionHTML = true
	} else {
		ep.Description = htmlPolicy.Sanitize(ep.Description)
	}
	ep.ShortDescription = htmlPolicy.Sanitize(ep.ShortDescription)
	if len(ep.ShortDescription) > 80 {
		index := strings.Index(ep.ShortDescription[77:], " ")
		ep.ShortDescription = ep.ShortDescription[0:77+index] + "..."
	}

	log.Printf(" - episode [%v] '%s', updating", ep.PubDate, ep.Title)
	if err := store.SaveEpisode(ctx, p, &ep); err != nil {
		return fmt.Errorf("error saving episode: %v", err)
	}

	return nil
}

// UpdatePodcast fetches the feed URL for the given podcast, parses it and updates all of the
// episodes we have stored for the podcast. This method updates the passed-in store.Podcast with
// the latest details.
//
// To keep memory usage managable, we use the xml.Decoder interface to decode the XML file in a
// streaming fashion.
//
// If force is false, then we assume the podcast only has the latest handful of episodes -- anything
// older than the oldest episode we have already stored is ignored (if there's no existing episode
// then we assume this is a new podcast and load everything).
//
// If force is true, then we ignore existing episodes and re-store all episodes in the RSS file.
func UpdatePodcast(ctx context.Context, p *store.Podcast, force bool) (int, error) {
	log.Printf("Updating podcast: [%d] %s", p.ID, p.Title)

	// Fetch the RSS feed via a HTTP request.
	req, err := http.NewRequest("GET", p.FeedURL, nil)
	if err != nil {
		log.Printf(" - error creating RSS request: %v", err)
		return 0, err
	}

	if !p.LastFetchTime.IsZero() && !force && len(p.Episodes) > 0 {
		req.Header.Set("If-Modified-Since", p.LastFetchTime.UTC().Format(time.RFC1123))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error fetching URL: %s: %v", p.FeedURL, err)
	}
	log.Printf(" - fetched %d bytes, status %d %s\n", resp.ContentLength, resp.StatusCode, resp.Status)
	if resp.StatusCode == 304 {
		log.Printf(" - podcast has not changed since %s, not updating\n", p.LastFetchTime)
		return 0, nil
	}
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("error fetching URL: %s status=%d", p.FeedURL, resp.StatusCode)
	}

	// No episodes, so we do the same as if you'd specified force=true -- download everything
	if len(p.Episodes) == 0 {
		force = true
	}

	// Unmarshal the RSS feed, loading epsiodes as we go. We are extremely forgiving on the XML
	// structure, basically skipping everything that's not an <item> element (where the episode
	// details are stored).
	var item Item
	decoder := xml.NewDecoder(resp.Body)
	numUpdated := 0
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
					return 0, err
				}

				if err := updateEpisode(ctx, item, p); err != nil {
					log.Printf("Error updating item: %v", err)
					return numUpdated, err
				}
				numUpdated++
			}
		}
	}

	return numUpdated, err
}
