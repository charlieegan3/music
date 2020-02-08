package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
)

type monthTopPlayed struct {
	Month  string
	Pretty string
	Top    []struct {
		Track   string
		Artist  string
		Count   int
		Artwork string
		Spotify string
	}
}

// SummaryMonths gets a list of recently played tracks
func SummaryMonths() error {
	// Gather env config values
	projectID := os.Getenv("GOOGLE_PROJECT")
	datasetName := os.Getenv("GOOGLE_DATASET")
	enrichedTableName := os.Getenv("GOOGLE_TABLE_ENRICHED")
	bucketName := os.Getenv("GOOGLE_SUMMARY_BUCKET")
	objectName := "stats-months.json"

	// get the credentials from json
	ctx := context.Background()

	// create a big query client to query for the music stats
	bigqueryClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("Failed to create client: %v", err)
	}

	plays, err := monthsTopPlayed(ctx, bigqueryClient, projectID, datasetName, enrichedTableName)
	if err != nil {
		return fmt.Errorf("Failed to get most recent plays: %v", err)
	}
	// fetch and format data
	output := struct {
		LastUpdated string
		Months      []monthTopPlayed
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

func monthsTopPlayed(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string) ([]monthTopPlayed, error) {
	var results []monthTopPlayed
	queryString :=
		`SELECT
  month,
  max(pretty) as pretty,
  ARRAY_AGG(STRUCT(track, artist, count, artwork, spotify)
    ORDER BY count DESC
    LIMIT 10) AS top

FROM(
  SELECT
    COUNT(track) AS count,
    track,
    artist,
    month,
    pretty,
    max(album_cover) as artwork,
    max(spotify_id) as spotify
  FROM (
	  SELECT *, FORMAT_DATE('%Y-%m', DATE(timestamp)) as month, FORMAT_DATE('%B %Y', DATE(timestamp)) as pretty` +
			"\nFROM " + fmt.Sprintf("`%s.%s.%s`\n", projectID, datasetName, tableName) +
			`  )
  group by track, artist, month, pretty
  order by count desc
)

GROUP BY month
ORDER BY month desc
`

	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		return results, fmt.Errorf("Failed to get recent plays: %v", err)
	}

	for {
		var result monthTopPlayed
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
