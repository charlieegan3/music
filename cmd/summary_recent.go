package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type recentPlay struct {
	Track     string
	Artist    string
	Album     string
	Timestamp time.Time
	// Duration int
	Artwork string
}

// SummaryRecent gets a list of recently played tracks
func SummaryRecent() {
	// Gather env config values
	projectID := os.Getenv("GOOGLE_PROJECT")
	datasetName := os.Getenv("GOOGLE_DATASET")
	tableName := os.Getenv("GOOGLE_TABLE")
	accountJSON := os.Getenv("GOOGLE_JSON")
	bucketName := os.Getenv("GOOGLE_SUMMARY_BUCKET")
	objectName := "stats-recent.json"

	// get the credentials from json
	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(ctx, []byte(accountJSON), bigquery.Scope, storage.ScopeReadWrite)
	if err != nil {
		log.Fatalf("Creds parse failed: %v", err)
		os.Exit(1)
	}

	// create a big query client to query for the music stats
	bigqueryClient, err := bigquery.NewClient(ctx, projectID, option.WithCredentials(creds))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
		os.Exit(1)
	}

	// fetch and format data
	output := struct {
		LastUpdated string
		RecentPlays []recentPlay
	}{
		time.Now().UTC().Format(time.RFC3339),
		mostRecentPlays(ctx, bigqueryClient, projectID, datasetName, tableName),
	}

	// format data as json
	bytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	storageClient, err := storage.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalf("Client create Failed: %v", err)
		os.Exit(1)
	}

	bkt := storageClient.Bucket(bucketName)
	obj := bkt.Object(objectName)

	w := obj.NewWriter(ctx)
	w.ContentType = "application/json"
	w.ObjectAttrs.CacheControl = "no-cache=300"

	if _, err := fmt.Fprintf(w, string(bytes)); err != nil {
		log.Fatalf("Write Failed: %v", err)
		os.Exit(1)
	}
	if err := w.Close(); err != nil {
		log.Fatalf("Close Failed: %v", err)
		os.Exit(1)
	}
}

func mostRecentPlays(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string) []recentPlay {
	queryString :=
		"SELECT track, artist, album, timestamp, duration, spotify_id as spotify, album_cover as artwork\n" +
			"FROM " + fmt.Sprintf("`%s.%s.%s`\n", projectID, datasetName, tableName) +
			`ORDER BY timestamp desc
			LIMIT 100`

	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var results []recentPlay
	var result recentPlay
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
