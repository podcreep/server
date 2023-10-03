package rss

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/podcreep/server/store"
	"github.com/podcreep/server/util"

	"github.com/microcosm-cc/bluemonday"
)

var (
	// An empty policy will strip all HTML tags, which is what we actually want.
	htmlPolicy = bluemonday.NewPolicy()

	// http.Client we'll use to make HTTP requests.
	httpClient = &http.Client{}
)

type UpdatePodcastFlags int

const (
	ForceUpdate UpdatePodcastFlags = 1 << iota
	IconOnly
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
	ep.ShortDescription = html.UnescapeString(ep.ShortDescription)
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

func updateChannelImage(ctx context.Context, url string, p *store.Podcast) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error fetching %s: %w", url, err)
	}
	if p.ImagePath != nil {
		maybeAddIfModifiedSince(req, p)
	}
	req.Header["User-Agent"] = []string{util.GetUserAgent()}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error fetching image URL: %s %v", url, err)
		// We don't consider this a bad enough error to stop fetching the rest of the podcast URL.
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 304 {
		log.Printf("Image hasn't been updated, no need to fetch again.")
	} else {
		file, err := os.CreateTemp("", "icon")
		if err != nil {
			return fmt.Errorf("error creating a temporary file: %w", err)
		}
		defer os.Remove(file.Name())

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return fmt.Errorf("error saving image: %w", err)
		}

		newSha1, _ := calculateSha1(file.Name())
		log.Printf("New image SHA1: %s", newSha1)

		oldSha1 := ""
		if p.ImagePath != nil {
			oldSha1, err = calculateSha1(*p.ImagePath)
			if err != nil {
				return fmt.Errorf("error calculating SHA1: %w", err)
			}
		}

		if oldSha1 != newSha1 {
			log.Printf("SHA mismatch: %s != %s", newSha1, oldSha1)
			basePath, err := store.GetBlobStorePath("icons")
			if err != nil {
				return err
			}

			iconPath := path.Join(basePath, newSha1+".png")
			iconFile, err := os.Create(iconPath)
			if err != nil {
				return fmt.Errorf("error opening icon file %s: %w", iconPath, err)
			}
			file.Seek(0, 0)
			_, err = io.Copy(iconFile, file)
			if err != nil {
				return err
			}

			p.ImagePath = &iconPath
			p.ImageURL = fmt.Sprintf("/blobs/podcasts/%d/icon/%s.png", p.ID, newSha1)
			p.IsImageExternal = false
		}
	}

	return nil
}

func decodeChannelElement(ctx context.Context, se xml.StartElement, decoder *xml.Decoder, p *store.Podcast, flags UpdatePodcastFlags) (int, error) {
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

		var item Item
		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "item" {
				err := decoder.DecodeElement(&item, &se)
				if err != nil {
					return 0, fmt.Errorf("error parsing item: %w", err)
				}

				if (flags & IconOnly) == 0 {
					if err := updateEpisode(ctx, item, p); err != nil {
						return numUpdated, fmt.Errorf("error updating item: %w", err)
					}
					numUpdated++
				}
			} else if se.Name.Local == "image" {
				var image Image
				if err := decoder.DecodeElement(&image, &se); err != nil {
					return 0, fmt.Errorf("error parsing image: %w", err)
				}

				url := image.URL
				if url == "" {
					// Sometimes there's an <itunes:image>, which has a 'href' attribute. We'll try that
					// as well. If the XML has both (and they are different) the last one wins.
					url = image.Href
				}

				if err := updateChannelImage(ctx, url, p); err != nil {
					return 0, fmt.Errorf("error updating channel image: %w", err)
				}
			}
		}
	}

	return numUpdated, nil
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
// If flags contains ForceUpdate, then we ignore existing episodes and re-store all episodes in the
// RSS file. If it contains IconOnly, we skip updating episodes and just update the icon.
func UpdatePodcast(ctx context.Context, p *store.Podcast, flags UpdatePodcastFlags) (int, error) {
	log.Printf("Updating podcast: [%d] %s", p.ID, p.Title)

	// Fetch the RSS feed via a HTTP request.
	req, err := http.NewRequest("GET", p.FeedURL, nil)
	if err != nil {
		log.Printf(" - error creating RSS request: %v", err)
		return 0, err
	}
	if (flags & ForceUpdate) == 0 {
		maybeAddIfModifiedSince(req, p)
	}
	req.Header["User-Agent"] = []string{util.GetUserAgent()}

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
		dump, _ := httputil.DumpResponse(resp, true)
		return 0, fmt.Errorf("error fetching URL: %s status=%d\n%s", p.FeedURL, resp.StatusCode, string(dump))
	}

	// Unmarshal the RSS feed, loading epsiodes as we go. We are extremely forgiving on the XML
	// structure, basically skipping everything that's not an <item> element (where the episode
	// details are stored).
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
			if se.Name.Local == "channel" {
				return decodeChannelElement(ctx, se, decoder, p, flags)
			}
		}
	}

	return numUpdated, err
}
