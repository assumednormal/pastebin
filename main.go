package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"time"

	_ "github.com/lib/pq"
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
	UserID    string `json:"user"`
}

// Pastes represents a slice of pastes to pastebin.com
type Pastes []Paste

var (
	dbname   = flag.String("db", "pastes", "Postgres database")
	limit    = flag.Int("limit", 50, "number of recent pastes to request (1 <= limit <= 250)")
	password = flag.String("password", "", "Postgres password")
	rate     = flag.Duration("rate", time.Minute, "rate to make requests (1 minute <= rate)")
	user     = flag.String("user", "", "Postgres username")
)

func scrape(limit int, q chan struct{}, db *sql.DB) {
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

	stmt, err := db.Prepare("insert into pastes values ($1, $2, $3, $4, $5, $6, $7, $8, $9)")
	if err != nil {
		fmt.Println(err)
		close(q)
	}

	for _, paste := range *pastes {
		_, err := stmt.Exec(paste.ScrapeURL, paste.FullURL, paste.Date, paste.Key,
			paste.Size, paste.Expire, paste.Title, paste.Syntax, paste.UserID)
		if err != nil {
			fmt.Println(err)
			close(q)
		}
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

	dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", *user, *password, *dbname)
	db, err := sql.Open("postgres", dbinfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	t := time.NewTicker(*rate)
	q := make(chan struct{})
	go func() {
		for {
			select {
			case <-t.C:
				scrape(*limit, q, db)
			case <-q:
				t.Stop()
				return
			}
		}
	}()

	<-q
}
