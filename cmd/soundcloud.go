package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
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

// Soundcloud downloads plays from sound cloud
func Soundcloud() error {
	content, err := fetchRecentPlayJSON()
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
	projectID := os.Getenv("GOOGLE_PROJECT")
	datasetName := os.Getenv("GOOGLE_DATASET")
	tableName := os.Getenv("GOOGLE_TABLE")
	accountJSON := os.Getenv("GOOGLE_JSON")

	creds, err := google.CredentialsFromJSON(ctx, []byte(accountJSON), bigquery.Scope)
	if err != nil {
		return fmt.Errorf("Failed to load creds from json: %v", err)
	}
	bigqueryClient, err := bigquery.NewClient(ctx, projectID, option.WithCredentials(creds))
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
	mostRecentTimestamp := mostRecentSoundCloudTimestamp(ctx, bigqueryClient, projectID, datasetName, tableName)

	for _, item := range recentlyPlayed {
		if mostRecentTimestamp.Unix() > (item.PlayedAt.Unix() - 1) {
			fmt.Println("skipping earlier track", item.Title)
			continue
		}

		// creates items to be saved in big query
		var vss []*bigquery.ValuesSaver
		vss = append(vss, &bigquery.ValuesSaver{
			Schema:   schema,
			InsertID: fmt.Sprintf("%v", item.PlayedAt.Unix()),
			Row: []bigquery.Value{
				item.Title,
				item.Artist,
				"", // album
				fmt.Sprintf("%d", item.PlayedAt.Unix()),
				bigquery.NullInt64{Int64: int64(item.Duration), Valid: true},
				"", // spotify_id
				item.Artwork,
				fmt.Sprintf("%d", time.Now().Unix()), // created_at
				"soundcloud",                         // source
				"",                                   // youtube_id
				"",                                   // youtube_category_id
				item.ID,                              // soundcloud_id
				item.PermalinkURL,                    // soundcloud_permalink
				"",                                   // shazam_id
				"",                                   // shazam_permalink
			},
		})

		// upload the items
		err = u.Put(ctx, vss)
		if err != nil {
			if pmErr, ok := err.(bigquery.PutMultiError); ok {
				for _, rowInsertionError := range pmErr {
					log.Println(rowInsertionError.Errors)
				}
			}

			return fmt.Errorf("Failed to upload items: %v", err)
		}

		fmt.Println("uploaded", item.Artist, " | ", item.Title)
	}

	return nil
}

func fetchRecentPlayJSON() ([]byte, error) {
	req, err := http.NewRequest("GET", os.Getenv("SOUNDCLOUD_URL"), nil)
	if err != nil {
		return []byte{}, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:62.0) Gecko/20100101 Firefox/62.0")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.7,en-US;q=0.3")
	req.Header.Set("Referer", "https://soundcloud.com/")
	req.Header.Set("Authorization", os.Getenv("SOUNDCLOUD_OAUTH"))

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

func mostRecentSoundCloudTimestamp(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string) time.Time {
	queryString := fmt.Sprintf(
		"SELECT timestamp FROM `%s.%s.%s` WHERE source = \"soundcloud\" ORDER BY timestamp DESC LIMIT 1",
		projectID,
		datasetName,
		tableName,
	)
	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err == iterator.Done {
		fmt.Println("no soundcloud tracks found, returning early time")
		return time.Unix(0, 0)
	}
	var l struct {
		Timestamp time.Time
	}
	for {
		err := it.Next(&l)
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		break
	}
	return l.Timestamp
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
