package main

import (
	"encoding/json"
	"fmt"
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
func SummaryRecent() error {
	// Gather env config values
	projectID := os.Getenv("GOOGLE_PROJECT")
	datasetName := os.Getenv("GOOGLE_DATASET")
	enrichedTableName := os.Getenv("GOOGLE_TABLE_ENRICHED")
	accountJSON := os.Getenv("GOOGLE_JSON")
	bucketName := os.Getenv("GOOGLE_SUMMARY_BUCKET")
	objectName := "stats-recent.json"

	// get the credentials from json
	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(ctx, []byte(accountJSON), bigquery.Scope, storage.ScopeReadWrite)
	if err != nil {
		return fmt.Errorf("Creds parse failed: %v", err)
	}

	// create a big query client to query for the music stats
	bigqueryClient, err := bigquery.NewClient(ctx, projectID, option.WithCredentials(creds))
	if err != nil {
		return fmt.Errorf("Failed to create client: %v", err)
	}

	plays, err := mostRecentPlays(ctx, bigqueryClient, projectID, datasetName, enrichedTableName)
	if err != nil {
		return fmt.Errorf("Failed to get most recent plays: %v", err)
	}
	// fetch and format data
	output := struct {
		LastUpdated string
		RecentPlays []recentPlay
	}{
		time.Now().UTC().Format(time.RFC3339),
		plays,
	}

	// format data as json
	bytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON MarshalIndent failed: %v", err)
	}

	storageClient, err := storage.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return fmt.Errorf("Client create Failed: %v", err)
	}

	bkt := storageClient.Bucket(bucketName)
	obj := bkt.Object(objectName)

	w := obj.NewWriter(ctx)
	w.ContentType = "application/json"
	w.ObjectAttrs.CacheControl = "max-age=300"

	if _, err := fmt.Fprintf(w, string(bytes)); err != nil {
		return fmt.Errorf("Write Failed: %v", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("Close Failed: %v", err)
	}
	return nil
}

func mostRecentPlays(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string) ([]recentPlay, error) {
	var results []recentPlay
	queryString :=
		"SELECT track, artist, album, timestamp, duration, spotify_id as spotify, album_cover as artwork\n" +
			"FROM " + fmt.Sprintf("`%s.%s.%s`\n", projectID, datasetName, tableName) +
			`ORDER BY timestamp desc
			LIMIT 100`

	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		return results, fmt.Errorf("Failed to get recent plays: %v", err)
	}

	var result recentPlay
	for {
		err := it.Next(&result)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return results, fmt.Errorf("Failed to parse recent plays: %v", err)
		}
		results = append(results, result)
	}

	return results, nil
}
