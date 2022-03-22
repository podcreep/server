package rss

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/podcreep/server/store"

	"github.com/microcosm-cc/bluemonday"
)

var (
	// An empty policy will strip all HTML tags, which is what we actually want.
	htmlPolicy = bluemonday.NewPolicy()

	// http.Client we'll use to make HTTP requests.
	httpClient = &http.Client{}
)

// maybeAddIfModifiedSince will add an If-Modified-Since header to the given request, based on the
// last fetch time of the given podcast.
func maybeAddIfModifiedSince(req *http.Request, p *store.Podcast) {
	if !p.LastFetchTime.IsZero() && len(p.Episodes) > 0 {
		req.Header.Set("If-Modified-Since", p.LastFetchTime.UTC().Format(time.RFC1123))
	}
}

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

func calculateSha1(filepath string) (string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func updateChannel(ctx context.Context, channel Channel, p *store.Podcast) error {
	// Note: we do not update the channel title or description, as these can be edited by the admin.
	// But we do want to check if the image URL has changed.
	// TODO: remember if they customized it and don't modify it.

	url := channel.Image.URL
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	maybeAddIfModifiedSince(req, p)

	resp, err := httpClient.Do(req)
	if err != nil {
		// Note: We'll ignore errors here, if anything goes wrong, we just ignore this update. Hopefully
		// next time it'll work. Some errors we don't ignore, but stuff like this one would most likely
		// just be a HTTP error on the server or something.
		log.Printf("Error fetching podcast URL: %s %v", url, err)
		// TODO: if we do more than update the icon, we'll want to add this to a helper function.
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 304 {
		log.Printf("Image hasn't been updated, no need to fetch again.")
	} else {
		file, err := ioutil.TempFile("", "icon")
		if err != nil {
			return fmt.Errorf("Error creating a temporary file: %w", err)
		}
		defer os.Remove(file.Name())

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return fmt.Errorf("Error saving image: %w", err)
		}

		newSha1, err := calculateSha1(file.Name())
		log.Printf("New image SHA1: %s", newSha1)

		oldSha1 := ""
		if p.ImagePath != nil {
			oldSha1, err = calculateSha1(*p.ImagePath)
			if err != nil {
				return fmt.Errorf("Error calculating SHA1: %w", err)
			}
		}

		if oldSha1 != newSha1 {
			log.Printf("SHA mismatch: %s != %s", newSha1, oldSha1)

		}
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
	if !force {
		maybeAddIfModifiedSince(req, p)
	}

	resp, err := httpClient.Do(req)
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
					return 0, fmt.Errorf("Error parsing item: %w", err)
				}

				if err := updateEpisode(ctx, item, p); err != nil {
					return numUpdated, fmt.Errorf("Error updating item: %w", err)
				}
				numUpdated++
			} /* TODO else if se.Name.Local == "channel" {
				var channel Channel
				err := decoder.DecodeElement(&channel, &se)
				if err != nil {
					return 0, fmt.Errorf("Error parsing channel: %w", err)
				}

				if err := updateChannel(ctx, channel, p); err != nil {
					log.Printf("Error updating channel")
				}
			}*/
		}
	}

	return numUpdated, err
}
