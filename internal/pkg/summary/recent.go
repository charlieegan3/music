package summary

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/charlieegan3/music/internal/pkg/config"
	"google.golang.org/api/iterator"
)

type recentPlay struct {
	Track     string
	Artist    string
	Album     string
	Timestamp time.Time
	// Duration int
	Artwork string
}

// Recent gets a list of recently played tracks
func Recent(cfg config.Config) error {
	projectID := cfg.Google.Project
	datasetName := cfg.Google.Dataset
	enrichedTableName := cfg.Google.TableEnrich
	bucketName := cfg.Google.BucketSummary
	objectName := "stats-recent.json"

	// get the credentials from json
	ctx := context.Background()

	// create a big query client to query for the music stats
	bigqueryClient, err := bigquery.NewClient(ctx, projectID)
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

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("Client create Failed: %v", err)
	}

	bkt := storageClient.Bucket(bucketName)
	obj := bkt.Object(objectName)

	w := obj.NewWriter(ctx)
	w.ContentType = "application/json"
	w.ObjectAttrs.CacheControl = "max-age=300"

	if _, err := w.Write(bytes); err != nil {
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
