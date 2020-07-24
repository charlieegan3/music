package lastfm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
)

// EnableUpload should be set to false to import data
var EnableUpload = true

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

// Import gets a list of recently played tracks
// This can't really work in some generic way as the timestamps don't match
// with spotify. Manually find the start and the end of the gap and set them as
// flags
func Import(startTime, endTime time.Time, projectID, datasetName, tableName string, dryrun bool) error {
	// Creates a client.
	ctx := context.Background()
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

		if EnableUpload {
			if err := u.Put(ctx, vss); err != nil {
				fmt.Println(err)
				return err
			}
		} else {
			log.Println("skipping upload of page")
		}
	}

	fmt.Println("done")

	return nil
}
