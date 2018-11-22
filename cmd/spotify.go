package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/zmb3/spotify"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// Spotify gets a list of recently played tracks
func Spotify() {
	// Creates a bq client.
	ctx := context.Background()
	projectID := os.Getenv("GOOGLE_PROJECT")
	datasetName := os.Getenv("GOOGLE_DATASET")
	tableName := os.Getenv("GOOGLE_TABLE")
	accountJSON := os.Getenv("GOOGLE_JSON")

	creds, err := google.CredentialsFromJSON(ctx, []byte(accountJSON), bigquery.Scope)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	bigqueryClient, err := bigquery.NewClient(ctx, projectID, option.WithCredentials(creds))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
		os.Exit(1)
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
	mostRecentTimestamp := mostRecentTimestamp(ctx, bigqueryClient, projectID, datasetName, tableName)

	// Creates a spotify client
	spotifyClient := buildClient()
	recentlyPlayed, err := spotifyClient.PlayerRecentlyPlayedOpt(&spotify.RecentlyPlayedOptions{Limit: 50})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for _, item := range recentlyPlayed {
		if mostRecentTimestamp.Unix() < item.PlayedAt.Unix() {
			fullTrack, err := spotifyClient.GetTrack(item.Track.ID)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			var artists []string
			for _, a := range item.Track.Artists {
				artists = append(artists, a.Name)
			}
			var image string
			if len(fullTrack.Album.Images) > 0 {
				image = fullTrack.Album.Images[0].URL
			}
			// creates items to be saved in big query
			var vss []*bigquery.ValuesSaver
			vss = append(vss, &bigquery.ValuesSaver{
				Schema:   schema,
				InsertID: fmt.Sprintf("%v", item.PlayedAt.Unix()),
				Row: []bigquery.Value{
					item.Track.Name,
					strings.Join(artists, ", "),
					fullTrack.Album.Name,
					fmt.Sprintf("%d", item.PlayedAt.Unix()),
					bigquery.NullInt64{Int64: int64(item.Track.Duration), Valid: true},
					fmt.Sprintf("%s", item.Track.ID),
					image,
					fmt.Sprintf("%d", time.Now().Unix()),
					"spotify",
					"", // youtube_id
					"", // youtube_category_id
					"", // soundcloud_id
					"", // soundcloud_permalink
					"", // shazam_id
					"", // shazam_permalink
				},
			})

			// upload the items
			err = u.Put(ctx, vss)
			if err != nil {
				if pmErr, ok := err.(bigquery.PutMultiError); ok {
					for _, rowInsertionError := range pmErr {
						log.Println(rowInsertionError.Errors)
					}
					return
				}

				log.Println(err)
			}
			fmt.Printf("%v %s\n", item.PlayedAt, item.Track.Name)
		}
	}

}

func mostRecentTimestamp(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string) time.Time {
	queryString := fmt.Sprintf(
		"SELECT timestamp FROM `%s.%s.%s` WHERE source = \"spotify\" OR source IS NULL ORDER BY timestamp DESC LIMIT 1",
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
