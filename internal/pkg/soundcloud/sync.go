package soundcloud

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	bq "github.com/charlieegan3/music/internal/pkg/bigquery"
	"github.com/charlieegan3/music/internal/pkg/config"
	"golang.org/x/net/context"
)

type soundcloudResponse struct {
	Collection []struct {
		PlayedAt int64 `json:"played_at"`
		Track    struct {
			ArtworkURL   string `json:"artwork_url"`
			Duration     int64  `json:"duration"`
			ID           int64  `json:"id"`
			PermalinkURL string `json:"permalink_url"`
			Title        string `json:"title"`
			User         struct {
				FullName string `json:"full_name"`
				Username string `json:"username"`
			} `json:"user"`
		} `json:"track"`
	} `json:"collection"`
}

type soundcloudPlay struct {
	ID           string
	PlayedAt     time.Time
	Title        string
	Artist       string
	Duration     int64
	PermalinkURL string
	Artwork      string
}

// Sync downloads plays from sound cloud
func Sync(cfg config.Config) error {
	content, err := fetchRecentPlayJSON(cfg.Soundcloud.URL, cfg.Soundcloud.Oauth)
	if err != nil {
		return fmt.Errorf("Failed to get recent plays: %v", err)
	}

	var response soundcloudResponse
	err = json.Unmarshal(content, &response)
	if err != nil {
		return fmt.Errorf("Failed to parse JSON response: %v", err)
	}

	var recentlyPlayed []soundcloudPlay
	for _, v := range response.Collection {
		artist := v.Track.User.FullName

		if artist == "" {
			artist = v.Track.User.Username
		}

		recentlyPlayed = append([]soundcloudPlay{soundcloudPlay{
			ID:           strconv.FormatInt(v.Track.ID, 10),
			PlayedAt:     time.Unix(0, v.PlayedAt*int64(time.Millisecond)),
			Title:        compressTitle(v.Track.Title, v.Track.User.Username),
			Artist:       artist,
			Duration:     v.Track.Duration,
			PermalinkURL: v.Track.PermalinkURL,
			Artwork:      v.Track.ArtworkURL,
		}}, recentlyPlayed...)
	}

	// Creates a bq client.
	ctx := context.Background()
	projectID := cfg.Google.Project
	datasetName := cfg.Google.Dataset
	tableName := cfg.Google.Table

	bigqueryClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("Failed to create client: %v", err)
	}
	// loads in the table schema from file
	jsonSchema, err := ioutil.ReadFile("schema.json")
	if err != nil {
		return fmt.Errorf("Failed to create schema: %v", err)
	}
	schema, err := bigquery.SchemaFromJSON(jsonSchema)
	if err != nil {
		return fmt.Errorf("Failed to parse schema: %v", err)
	}
	u := bigqueryClient.Dataset(datasetName).Table(tableName).Uploader()
	mostRecentTimestamp, err := bq.MostRecentTimestamp(ctx, bigqueryClient, projectID, datasetName, tableName, "soundcloud")
	if err != nil {
		return fmt.Errorf("Failed to get most recent entry: %v", err)
	}

	for _, item := range recentlyPlayed {
		if mostRecentTimestamp.Unix() > (item.PlayedAt.Unix() - 1) {
			fmt.Println("skipping earlier track", item.Title)
			continue
		}

		err = bq.SavePlay(ctx,
			schema,
			*u,
			item.Title,
			item.Artist,
			"", // album
			fmt.Sprintf("%d", item.PlayedAt.Unix()),
			item.Duration,
			"", // spotify_id
			item.Artwork,
			"soundcloud",      // source
			"",                // youtube_id
			"",                // youtube_category_id
			item.ID,           // soundcloud_id
			item.PermalinkURL, // soundcloud_permalink
			"",                // shazam_id
			"",                // shazam_permalink
		)

		if err != nil {
			return fmt.Errorf("Failed to upload item: %v", err)
		}

		fmt.Println("uploaded", item.Artist, " | ", item.Title)
	}

	return nil
}

func fetchRecentPlayJSON(url, oauth string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []byte{}, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:62.0) Gecko/20100101 Firefox/62.0")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.7,en-US;q=0.3")
	req.Header.Set("Referer", "https://soundcloud.com/")
	req.Header.Set("Authorization", oauth)

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

func compressTitle(title string, artist string) string {
	junkAtStart := regexp.MustCompile(`^\W+`)

	newTitle := title

	if strings.HasPrefix(title, artist) {
		newTitle = strings.Replace(title, artist, "", 1)
		newTitle = junkAtStart.ReplaceAllString(newTitle, "")
	}

	if newTitle == "" {
		return title
	}

	return newTitle
}
