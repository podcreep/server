package discover

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/podcreep/server/util"
)

var (
	apiKey    string
	apiSecret string

	// http.Client we'll use to make HTTP requests.
	httpClient = &http.Client{}
)

type Podcast struct {
	ID                    int64  `json:"id"`
	Url                   string `json:"url"`
	Title                 string `json:"title"`
	Description           string `json:"description"`
	Link                  string `json:"link"`
	Author                string `json:"author"`
	ImageUrl              string `json:"image"`
	ArtworkUrl            string `json:"artwork"`
	NewestItemPublishTime int64  `json:"newestItemPublishTime"`
	// TODO "itunesId": 269169796,
	// TODO "trendScore": 227,
	// TODO "language": "en",
	// TODO "categories": { "55": "News", "59": "Politics", "16": "Comedy" }
}

type PodcastListResult struct {
	Status string    `json:"status"`
	Feeds  []Podcast `json:"feeds"`
}

type PodcastResult struct {
	Status string  `json:"status"`
	Feed   Podcast `json:"feed"`
}

// makeRequest makes a request for the given URL and then appends all
func makeRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching %s: %w", url, err)
	}

	authDate := strconv.FormatInt(time.Now().Unix(), 10)

	req.Header["User-Agent"] = []string{util.GetUserAgent()}
	req.Header["X-Auth-Key"] = []string{apiKey}
	req.Header["X-Auth-Date"] = []string{authDate}

	// Add the Authorization header
	hasher := sha1.New()
	_, err = io.WriteString(hasher, apiKey+apiSecret+authDate)
	if err != nil {
		return nil, fmt.Errorf("error hashing auth header: %v", err)
	}
	req.Header["Authorization"] = []string{fmt.Sprintf("%x", hasher.Sum(nil))}

	return req, nil
}

func Setup() error {
	apiKey = os.Getenv("PODCASTINDEX_APIKEY")
	apiSecret = os.Getenv("PODCASTINDEX_APISECRET")
	return nil
}

func performQuery(req *http.Request) ([]Podcast, error) {
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching trending: %v", err)
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading trendingresp: %v", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	res := PodcastListResult{}
	err = decoder.Decode(&res)
	if err != nil {
		return nil, fmt.Errorf(string(bytes[0:100])+": %v", err)
	}

	return res.Feeds, nil
}

func FetchTrending() ([]Podcast, error) {
	req, err := makeRequest("https://api.podcastindex.org/api/1.0/podcasts/trending")
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}

	return performQuery(req)
}

func Search(query string) ([]Podcast, error) {
	req, err := makeRequest("https://api.podcastindex.org/api/1.0/search/byterm")
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	q := req.URL.Query()
	q.Add("q", query)
	q.Add("similar", "true") // TODO: allow this to be configured?
	req.URL.RawQuery = q.Encode()

	return performQuery(req)
}

func FetchPodcast(id int64) (*Podcast, error) {
	req, err := makeRequest(fmt.Sprintf("https://api.podcastindex.org/api/1.0/podcasts/byfeedid?id=%d", id))
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching trending: %v", err)
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading resp: %v", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	res := PodcastResult{}
	err = decoder.Decode(&res)
	if err != nil {
		return nil, fmt.Errorf(string(bytes[0:100])+": %v", err)
	}

	fmt.Printf("feed: %v", res.Feed)

	return &res.Feed, nil
}
