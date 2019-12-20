package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
)

type shazamResponse struct {
	Tags []struct {
		Timestamp int64 `json:"timestamp"`
		Track     struct {
			Actions []struct {
				ID string `json:"id"`
			} `json:"actions"`
			Footnotes []struct {
				Title string `json:"title"`
				Value string `json:"value"`
			} `json:"footnotes"`
			Heading struct {
				Subtitle string `json:"subtitle"`
				Title    string `json:"title"`
			} `json:"heading"`
			Images struct {
				Default string `json:"default"`
			} `json:"images"`
			URL string `json:"url"`
		} `json:"track"`
	} `json:"tags"`
}

type shazamPlay struct {
	Album        string
	Artist       string
	Artwork      string
	ID           string
	PermalinkURL string
	PlayedAt     time.Time
	Track        string
}

// Shazam downloads plays from sound cloud
func Shazam() error {
	content, err := fetchRecentShazamJSON()
	if err != nil {
		log.Fatal(err)
	}

	var response shazamResponse
	err = json.Unmarshal(content, &response)
	if err != nil {
		log.Fatal(err)
	}

	var recentlyPlayed []shazamPlay
	lastTrackName := ""
	for _, v := range response.Tags {

		if lastTrackName == v.Track.Heading.Title {
			fmt.Println("skipping repeated shazam", v.Track.Heading.Title)
			continue
		}

		album := ""
		for _, footnote := range v.Track.Footnotes {
			if footnote.Title == "Album" {
				album = footnote.Value
				break
			}
		}

		recentlyPlayed = append([]shazamPlay{shazamPlay{
			Album:        album,
			Artist:       v.Track.Heading.Subtitle,
			Artwork:      v.Track.Images.Default,
			ID:           v.Track.Actions[0].ID,
			PermalinkURL: v.Track.URL,
			PlayedAt:     time.Unix(0, v.Timestamp*int64(time.Millisecond)),
			Track:        v.Track.Heading.Title,
		}}, recentlyPlayed...)

		lastTrackName = v.Track.Heading.Title
	}

	// Creates a bq client.
	ctx := context.Background()
	projectID := os.Getenv("GOOGLE_PROJECT")
	datasetName := os.Getenv("GOOGLE_DATASET")
	tableName := os.Getenv("GOOGLE_TABLE")

	bigqueryClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("Failed to create client: %v", err)
	}
	// loads in the table schema from file
	jsonSchema, err := ioutil.ReadFile("schema.json")
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
		os.Exit(1)
	}
	schema, err := bigquery.SchemaFromJSON(jsonSchema)
	if err != nil {
		log.Fatalf("Failed to parse schema: %v", err)
		os.Exit(1)
	}
	u := bigqueryClient.Dataset(datasetName).Table(tableName).Uploader()
	mostRecentTimestamp, err := mostRecentTimestamp(ctx, bigqueryClient, projectID, datasetName, tableName, "shazam")
	if err != nil {
		return fmt.Errorf("Failed to get most recently played: %v", err)
	}

	for _, item := range recentlyPlayed {
		if mostRecentTimestamp.Unix() > (item.PlayedAt.Unix() - 1) {
			fmt.Println("skipping earlier track", item.Track)
			continue
		}

		err = savePlay(ctx,
			schema,
			*u,
			item.Track,
			item.Artist,
			item.Album, // album
			fmt.Sprintf("%d", item.PlayedAt.Unix()),
			0,  // duration
			"", // spotify_id
			item.Artwork,
			"shazam",          // source
			"",                // youtube_id
			"",                // youtube_category_id
			"",                // soundcloud_id
			"",                // soundcloud_permalink
			item.ID,           // shazam_id
			item.PermalinkURL, // shazam_permalink
		)

		if err != nil {
			return fmt.Errorf("Failed to upload item: %v", err)
		}

		fmt.Println("uploaded", item.Artist, " | ", item.Track)
	}

	return nil
}

func fetchRecentShazamJSON() ([]byte, error) {
	req, err := http.NewRequest("GET", os.Getenv("SHAZAM_URL"), nil)
	if err != nil {
		return []byte{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:62.0) Gecko/20100101 Firefox/62.0")
	req.Header.Set("Referer", os.Getenv("SHAZAM_REFERRER"))
	req.Header.Set("Cookie", os.Getenv("SHAZAM_COOKIE"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}
