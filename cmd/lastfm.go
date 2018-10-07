package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
)

type lastFMDataPage []struct {
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
	} `json:"image"`
	Name string `json:"name"`
}

// LastFM gets a list of recently played tracks
func LastFM() {
	// loads in the data from file (https://mainstream.ghan.nl/export.html)
	jsonPages, err := ioutil.ReadFile("lastfm_data.json")
	if err != nil {
		log.Fatalf("Failed to read lastfm_data: %v", err)
	}
	var pages []lastFMDataPage
	err = json.Unmarshal(jsonPages, &pages)
	if err != nil {
		fmt.Println("error:", err)
	}

	// Creates a client.
	ctx := context.Background()
	projectID := os.Getenv("GOOGLE_PROJECT")
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	u := client.Dataset("music").Table("plays").Uploader()

	// loads in the table schema from file
	jsonSchema, err := ioutil.ReadFile("schema.json")
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}
	schema, err := bigquery.SchemaFromJSON(jsonSchema)
	if err != nil {
		log.Fatalf("Failed to parse schema: %v", err)
	}

	// creates items to be saved in big query
	for i, page := range pages {
		var vss []*bigquery.ValuesSaver
		for j, play := range page {
			image := ""
			if len(play.Image) > 0 {
				image = play.Image[0].Text
			}

			vss = append(vss, &bigquery.ValuesSaver{
				Schema:   schema,
				InsertID: fmt.Sprintf("%d-%d", i, j),
				Row: []bigquery.Value{
					play.Name,
					play.Artist.Text,
					play.Album.Text,
					fmt.Sprintf("%v", play.Date.Uts),
					bigquery.NullInt64{Valid: false}, // no duration in this dataset :(
					"", // no spotify id
					image,
				},
			})
		}
		// Sets the name for the new dataset.
		if err := u.Put(ctx, vss); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("%v/%v\n", i, len(pages))
	}

	fmt.Println("done")
}
