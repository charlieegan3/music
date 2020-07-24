package spotify

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	bq "github.com/charlieegan3/music/internal/pkg/bigquery"
	"github.com/zmb3/spotify"
	"google.golang.org/api/iterator"
)

// Sync gets a list of recently played tracks and uploads to bigquery
func Sync(accessToken, refreshToken, clientID, clientSecret, projectID, datasetName, tableName string) error {
	// Creates a bq client.
	ctx := context.Background()

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

	// Creates a spotify client
	spotifyClient := buildClient(accessToken, refreshToken, clientID, clientSecret)
	recentlyPlayed, err := spotifyClient.PlayerRecentlyPlayedOpt(&spotify.RecentlyPlayedOptions{Limit: 50})
	if err != nil {
		return fmt.Errorf("Failed to get recent plays: %v", err)
	}

	timestamps, err := nMostRecentTimestamps(ctx, bigqueryClient, projectID, datasetName, tableName, "spotify", 100)
	if err != nil {
		return fmt.Errorf("Failed to get most recent timestamps: %v", err)
	}

	// reverse to import in order in case of failure
	for i, j := 0, len(recentlyPlayed)-1; i < j; i, j = i+1, j-1 {
		recentlyPlayed[i], recentlyPlayed[j] = recentlyPlayed[j], recentlyPlayed[i]
	}
	failures := 0
	for _, item := range recentlyPlayed {
		// look for songs that arrive in recently played too late out of order
		found := false
		for _, v := range timestamps {
			if v.Truncate(time.Second) == item.PlayedAt.Truncate(time.Second) {
				found = true
				break
			}
		}

		if found == false {
			fullTrack, err := spotifyClient.GetTrack(item.Track.ID)
			if err != nil {
				fmt.Println(err)
				failures++
				continue
			}

			var artists []string
			for _, a := range item.Track.Artists {
				artists = append(artists, a.Name)
			}
			var image string
			if len(fullTrack.Album.Images) > 0 {
				image = fullTrack.Album.Images[0].URL
			}

			err = bq.SavePlay(ctx,
				schema,
				*u,
				item.Track.Name,
				strings.Join(artists, ", "),
				fullTrack.Album.Name,
				fmt.Sprintf("%d", item.PlayedAt.Unix()),
				int64(item.Track.Duration),
				fmt.Sprintf("%s", item.Track.ID),
				image,
				"spotify",
				"", // youtube_id
				"", // youtube_category_id
				"", // soundcloud_id
				"", // soundcloud_permalink
				"", // shazam_id
				"", // shazam_permalink
			)

			if err != nil {
				return fmt.Errorf("Failed to upload item: %v", err)
			}

			fmt.Printf("%v %s\n", item.PlayedAt, item.Track.Name)
		}
	}

	if failures > 0 {
		return fmt.Errorf("Some tracks failed (%d failures)", failures)
	}

	return nil
}

func nMostRecentTimestamps(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string, source string, count int) ([]time.Time, error) {
	var t []time.Time
	queryString := fmt.Sprintf(
		"SELECT timestamp FROM `%s.%s.%s` WHERE source = \"%s\" ORDER BY timestamp DESC LIMIT %d",
		projectID,
		datasetName,
		tableName,
		source,
		count,
	)
	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		return t, fmt.Errorf("Failed query for recent timestamps: %v", err)
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
			return t, fmt.Errorf("Failed reading results for time: %v", err)
		}
		t = append(t, l.Timestamp)
	}

	return t, nil
}
