package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Paste represents a paste from pastebin.com
type Paste struct {
	ScrapeURL string `json:"scrape_url"`
	FullURL   string `json:"full_url"`
	Date      string `json:"date"`
	Key       string `json:"key"`
	Size      string `json:"size"`
	Expire    string `json:"expire"`
	Title     string `json:"title"`
	Syntax    string `json:"syntax"`
	User      string `json:"user"`
}

// Pastes represents a slice of pastes to pastebin.com
type Pastes []Paste

var limit = flag.Int("limit", 50, "number of recent pastes to request (1 <= limit <= 250)")
var rate = flag.Duration("rate", time.Minute, "rate to make requests (1 minute <= rate)")

func scrape(limit int, q chan struct{}) {
	query := &url.Values{}
	query.Set("limit", fmt.Sprintf("%d", limit))

	url := &url.URL{
		Scheme:   "https",
		Host:     "pastebin.com",
		Path:     "api_scraping.php",
		RawQuery: query.Encode(),
	}

	r, err := http.Get(url.String())
	if err != nil {
		fmt.Println(err)
		close(q)
	}

	defer r.Body.Close()

	var pastes = new(Pastes)

	if err = json.NewDecoder(r.Body).Decode(&pastes); err != nil {
		fmt.Println(err)
		close(q)
	}

	if err = json.NewEncoder(os.Stdout).Encode(pastes); err != nil {
		fmt.Println(err)
		close(q)
	}
}

func main() {
	flag.Parse()

	if *limit <= 0 || 251 <= *limit {
		panic("limit must be between 1 and 250")
	}

	if rate.Minutes() < 1 {
		panic("rate must be at least 1 minute")
	}

	t := time.NewTicker(*rate)
	q := make(chan struct{})
	go func() {
		for {
			select {
			case <-t.C:
				scrape(*limit, q)
			case <-q:
				t.Stop()
				return
			}
		}
	}()

	<-q
}
