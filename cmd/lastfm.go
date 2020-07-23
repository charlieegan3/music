package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
)

type lastFMDataPage struct {
	Track []struct {
		Album struct {
			Text string `json:"#text"`
		} `json:"album"`
		Artist struct {
			Text string `json:"#text"`
		} `json:"artist"`
		Date struct {
			Uts string `json:"uts"`
		} `json:"date"`
		Image []struct {
			Text string `json:"#text"`
			Size string `json:"size"`
		} `json:"image"`
		Name string `json:"name"`
	} `json:"track"`
}

// LastFM gets a list of recently played tracks
// This can't really work in some generic way as the timestamps don't match
// with spotify
func LastFM() error {
	// Creates a client.
	ctx := context.Background()
	projectID := os.Getenv("GOOGLE_PROJECT")
	datasetName := os.Getenv("GOOGLE_DATASET")
	tableName := os.Getenv("GOOGLE_TABLE")

	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	u := client.Dataset(datasetName).Table(tableName).Uploader()

	// loads in the table schema from file
	jsonSchema, err := ioutil.ReadFile("schema.json")
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}
	schema, err := bigquery.SchemaFromJSON(jsonSchema)
	if err != nil {
		log.Fatalf("Failed to parse schema: %v", err)
	}

	// need to manually find the range of the gap as lastfm and spotify don't
	// have the same ts
	startTime := time.Unix(1595028112, 0)
	endTime := time.Unix(1595500783, 0)

	// loads in the data from file (https://mainstream.ghan.nl/export.html)
	jsonPages, err := ioutil.ReadFile("lastfm_data.json")
	if err != nil {
		log.Fatalf("Failed to read lastfm_data: %v", err)
	}
	var pages []lastFMDataPage
	err = json.Unmarshal(jsonPages, &pages)
	if err != nil {
		log.Fatalf("unmarshal error: %s", err)
	}

	// create a big query client to query for the music stats
	bigqueryClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("Failed to create client: %v", err)
	}

	plays, err := allTimestampsAndTracks(ctx, bigqueryClient, projectID, datasetName, tableName)
	if err != nil {
		return fmt.Errorf("Failed to get most recent plays: %v", err)
	}

	// TODO can be used to manually narrow in on the gap
	var filteredPlays []timestampTrack
	for _, v := range plays {
		if v.Timestamp.After(startTime) && v.Timestamp.Before(endTime) {
			filteredPlays = append(filteredPlays, v)
		}
	}

	for i, page := range pages {
		var vss []*bigquery.ValuesSaver
		for j, track := range page.Track {
			n, err := strconv.ParseInt(track.Date.Uts, 10, 64)
			if err != nil {
				continue
			}
			t := time.Unix(n, 0)

			if t.After(startTime) && t.Before(endTime) {
				fmt.Println(track.Name)
				image := ""
				if len(track.Image) > 0 {
					image = track.Image[0].Text
				}

				vss = append(vss, &bigquery.ValuesSaver{
					Schema:   schema,
					InsertID: fmt.Sprintf("%d-%d", i, j),
					Row: []bigquery.Value{
						track.Name,
						track.Artist.Text,
						track.Album.Text,
						fmt.Sprintf("%v", track.Date.Uts),
						bigquery.NullInt64{Valid: false},
						"",
						image,
						time.Now().Unix(),
						"lastfm",
						"",
						"",
						"",
						"",
						"",
						"",
					},
				})
			}
		}
		if err := u.Put(ctx, vss); err != nil {
			fmt.Println(err)
			return err
		}
	}

	fmt.Println("done")

	return nil
}

type timestampTrack struct {
	Track     string
	Timestamp time.Time
}

func allTimestampsAndTracks(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string) ([]timestampTrack, error) {
	var results []timestampTrack
	queryString :=
		"SELECT track, timestamp\n" +
			"FROM " + fmt.Sprintf("`%s.%s.%s`\n", projectID, datasetName, tableName) +
			`ORDER BY timestamp desc`

	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		return results, fmt.Errorf("Failed to get timestamps and tracks: %v", err)
	}

	var result timestampTrack
	for {
		err := it.Next(&result)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return results, fmt.Errorf("Failed to parse timestamps and tracks: %v", err)
		}
		results = append(results, result)
	}

	return results, nil
}
