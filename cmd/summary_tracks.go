package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
)

type trackWithCount struct {
	Track   string
	Artist  string
	Count   int
	Spotify string
	Artwork string
}

type artistWithTracks struct {
	Name   string
	Tracks []trackWithCount
}

func (a *artistWithTracks) TotalPlays() int {
	total := 0

	for _, v := range a.Tracks {
		total += v.Count
	}

	return total
}

type byPlays []artistWithTracks

func (a byPlays) Len() int           { return len(a) }
func (a byPlays) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPlays) Less(i, j int) bool { return a[i].TotalPlays() > a[j].TotalPlays() }

// SummaryTracks returns a list of all tracks and the number of plays for them
func SummaryTracks() error {
	// Gather env config values
	projectID := os.Getenv("GOOGLE_PROJECT")
	datasetName := os.Getenv("GOOGLE_DATASET")
	enirchedTableName := os.Getenv("GOOGLE_TABLE_ENRICHED")
	bucketName := os.Getenv("GOOGLE_SUMMARY_BUCKET")
	objectName := "stats-tracks.json"

	// get the credentials from json
	ctx := context.Background()

	// create a big query client to query for the music stats
	bigqueryClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("Failed to create client: %v", err)
	}

	// fetch and format data
	tracks := tracksWithCounts(ctx, bigqueryClient, projectID, datasetName, enirchedTableName)
	artists := groupByArtist(tracks)

	output := struct {
		LastUpdated string
		Artists     []artistWithTracks
	}{
		time.Now().UTC().Format(time.RFC3339),
		artists,
	}

	// format data as json
	bytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to indent JSON: %v", err)
	}

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("Client create Failed: %v", err)
	}

	bkt := storageClient.Bucket(bucketName)
	obj := bkt.Object(objectName)

	w := obj.NewWriter(ctx)
	w.ContentType = "application/json"
	w.ObjectAttrs.CacheControl = "max-age=3600"

	if _, err := w.Write(bytes); err != nil {
		return fmt.Errorf("Write Failed: %v", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("Close Failed: %v", err)
	}

	return nil
}

func tracksWithCounts(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string) []trackWithCount {
	queryString :=
		`SELECT
		   track,
		   artist,
		   count(track) as count,
		   ANY_VALUE(spotify_id) as spotify_id,
		   STRING_AGG(album_cover, "" ORDER BY LENGTH(album_cover) DESC LIMIT 1) as artwork` + "\n" +
			"FROM " + fmt.Sprintf("`%s.%s.%s`\n", projectID, datasetName, tableName) +
			`GROUP BY track, artist
		    ORDER BY artist ASC, count DESC`

	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var results []trackWithCount
	var result trackWithCount
	for {
		err := it.Next(&result)
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		results = append(results, result)
	}

	return results
}

func groupByArtist(tracks []trackWithCount) []artistWithTracks {
	artistsMap := make(map[string][]trackWithCount)

	for _, v := range tracks {
		artistsMap[v.Artist] = append(artistsMap[v.Artist], v)
	}

	artists := []artistWithTracks{}
	for k, v := range artistsMap {
		artists = append(artists, artistWithTracks{Name: k, Tracks: v})
	}
	sort.Sort(byPlays(artists))

	return artists
}
