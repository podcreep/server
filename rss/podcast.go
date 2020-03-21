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

func updateEpisode(ctx context.Context, ep *store.Episode, item Item, p *store.Podcast) error {
	pubDate, err := parsePubDate(item.PubDate)
	if err != nil {
		return fmt.Errorf("error parsing date: %v", err)
	}

	ep.GUID = item.GUID
	ep.MediaURL = item.Media.URL
	ep.Title = item.Title
	ep.Description = item.Description
	ep.DescriptionHTML = false
	ep.ShortDescription = item.Description
	ep.PubDate = pubDate

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

	if ep.ID == 0 {
		log.Printf(" - new episode [%v] '%s', updating", ep.PubDate, ep.Title)
	} else {
		log.Printf(" - updated episode %d [%v] '%s', updating", ep.ID, ep.PubDate, ep.Title)
	}
	_, err = store.SaveEpisode(ctx, p, ep)
	if err != nil {
		return fmt.Errorf("error saving episode: %v", err)
	}

	return nil
}

// maybeUpdateEpisode will update any new episode we're given.
func maybeUpdateEpisode(ctx context.Context, item Item, p *store.Podcast) (bool, error) {
	pubDate, err := parsePubDate(item.PubDate)
	if err != nil {
		log.Printf(" - failed to parse pubdate '%s' of '%s': %v\n", item.PubDate, item.Title, err)
		return false, fmt.Errorf("error parsing date: %v", err)
	}

	// First, check if it's too old entirely. The last episode in the podcast we have is the oldest
	// thing we'll bother to update.
	if len(p.Episodes) > 0 {
		if p.Episodes[len(p.Episodes)-1].PubDate.After(pubDate) {
			return false, nil
		}
	}

	// If it's in the list of episodes we already have, just ignore it as well (we don't bother
	// to update existing episodes)
	// TODO: is it OK to skip episodes?
	for _, existing := range p.Episodes {
		if existing.GUID == item.GUID {
			return false, nil
		}
	}

	// OK, seems to be new, add it.
	ep := &store.Episode{}
	return true, updateEpisode(ctx, ep, item, p)
}

// forceUpdateEpisode updates an episode even if it already exists in the data store.
func forceUpdateEpisode(ctx context.Context, item Item, p *store.Podcast, guidMap map[string]int64) error {
	ep := &store.Episode{
		ID: guidMap[item.GUID],
	}
	return updateEpisode(ctx, ep, item, p)
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
//
// Because the website we're downloading from can some times time out before we finish processing
// the file, if we're downloading all episodes (either force is true, or there's no existing
// episodes), then we download all items first before processing them.
func UpdatePodcast(ctx context.Context, p *store.Podcast, force bool) (int, error) {
	log.Printf("Updating podcast: [%d] %s", p.ID, p.Title)

	// Fetch the RSS feed via a HTTP request.
	req, err := http.NewRequest("GET", p.FeedURL, nil)
	if err != nil {
		log.Printf(" - error creating RSS request: %v", err)
		return 0, err
	}

	if !p.LastFetchTime.IsZero() && !force {
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
	var items []Item
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

				if force {
					items = append(items, item)
				} else {
					wasUpdated, err := maybeUpdateEpisode(ctx, item, p)
					if err != nil {
						log.Printf("Error updating item: %v", err)
						return 0, err
					}
					if wasUpdated {
						numUpdated++
					}
				}
			}
		}
	}

	if force {
		guidMap, err := store.LoadEpisodeGUIDs(ctx, p.ID)
		if err != nil {
			return 0, err
		}
		log.Printf("%d entries in GUID map", len(guidMap))
		for _, item := range items {
			err := forceUpdateEpisode(ctx, item, p, guidMap)
			if err != nil {
				log.Printf("Error updating item: %v", err)
				return 0, err
			}
			numUpdated++
		}
	}

	return numUpdated, err
}
