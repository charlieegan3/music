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

type resultPlayFromPeriod struct {
	Track   string
	Artist  string
	Album   string
	Artwork string
	// Duration int
	Spotify string

	Count int
}

type resultArtistFromPeriod struct {
	Artist string
	Count  int
}

type resultCountForMonth struct {
	Month  string
	Count  int
	Pretty string
}

// Overview saves a summary for the music homepage of top tracks for the month
// and counts by month
func Overview(cfg config.Config) error {
	projectID := cfg.Google.Project
	datasetName := cfg.Google.Dataset
	enrichedTableName := cfg.Google.TableEnrich
	bucketName := cfg.Google.BucketSummary
	objectName := "stats.json"

	// get the credentials from json
	ctx := context.Background()

	// create a big query client to query for the music stats
	bigqueryClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("Failed to create client: %v", err)
	}

	cly, err := countsForMonths(ctx, bigqueryClient, projectID, datasetName, enrichedTableName)
	if err != nil {
		return fmt.Errorf("Failed to get counts for last year %v", err)
	}
	pfly, err := playsFromLastNDays(ctx, bigqueryClient, projectID, datasetName, enrichedTableName, 365)
	if err != nil {
		return fmt.Errorf("Failed to get plays for last year %v", err)
	}
	pflm, err := playsFromLastNDays(ctx, bigqueryClient, projectID, datasetName, enrichedTableName, 30)
	if err != nil {
		return fmt.Errorf("Failed to get plays for last month %v", err)
	}
	pflw, err := playsFromLastNDays(ctx, bigqueryClient, projectID, datasetName, enrichedTableName, 7)
	if err != nil {
		return fmt.Errorf("Failed to get plays for last week %v", err)
	}
	afly, err := artistsForLastNDays(ctx, bigqueryClient, projectID, datasetName, enrichedTableName, 365)
	if err != nil {
		return fmt.Errorf("Failed to get artists for last year %v", err)
	}
	aflm, err := artistsForLastNDays(ctx, bigqueryClient, projectID, datasetName, enrichedTableName, 30)
	if err != nil {
		return fmt.Errorf("Failed to get artists for last month %v", err)
	}
	aflw, err := artistsForLastNDays(ctx, bigqueryClient, projectID, datasetName, enrichedTableName, 7)
	if err != nil {
		return fmt.Errorf("Failed to get artists for last week %v", err)
	}

	// fetch and format data
	output := struct {
		PlaysByMonth []resultCountForMonth

		PlaysYear  []resultPlayFromPeriod
		PlaysMonth []resultPlayFromPeriod
		PlaysWeek  []resultPlayFromPeriod

		ArtistsYear  []resultArtistFromPeriod
		ArtistsMonth []resultArtistFromPeriod
		ArtistsWeek  []resultArtistFromPeriod

		LastUpdated string
	}{
		cly,
		pfly,
		pflm,
		pflw,
		afly,
		aflm,
		aflw,

		time.Now().UTC().Format(time.RFC3339),
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
	w.ObjectAttrs.CacheControl = "max-age=3600"

	if _, err := w.Write(bytes); err != nil {
		return fmt.Errorf("Write Failed: %v", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("Close Failed: %v", err)
	}
	return nil
}

func countsForMonths(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string) ([]resultCountForMonth, error) {
	var results []resultCountForMonth
	queryString :=
		"SELECT FORMAT_DATE(\"%Y-%m\", DATE(timestamp)) as month, FORMAT_DATE(\"%B %Y\", DATE(timestamp)) as pretty, count(track) as count\n" +
			"FROM " + fmt.Sprintf("`%s.%s.%s`\n", projectID, datasetName, tableName) +
			`WHERE timestamp > TIMESTAMP("2000-01-01 00:00:00")
			GROUP BY month, pretty
			ORDER BY month ASC`

	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		return results, fmt.Errorf("Failed to get counts for year %v", err)
	}
	var result resultCountForMonth
	for {
		err := it.Next(&result)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return results, fmt.Errorf("Failed to extract count for response %v", err)
		}
		results = append(results, result)
	}
	return results, nil
}

func playsFromLastNDays(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string, days int) ([]resultPlayFromPeriod, error) {
	queryString := fmt.Sprintf(
		`SELECT
		  track,
		  artist,
		  album,
		  count(track) as count,
		  STRING_AGG(album_cover, "" ORDER BY LENGTH(album_cover) DESC LIMIT 1) as artwork,
		  ANY_VALUE(duration) as duration,
		  ANY_VALUE(spotify_id) as spotify
		FROM `+"`"+"%s.%s.%s"+"`"+
			`WHERE timestamp BETWEEN TIMESTAMP_ADD(CURRENT_TIMESTAMP(), INTERVAL -%d DAY) AND CURRENT_TIMESTAMP()
		GROUP BY track, artist, album
		ORDER BY count DESC
		LIMIT 10`,
		projectID,
		datasetName,
		tableName,
		days,
	)

	var results []resultPlayFromPeriod
	var result resultPlayFromPeriod
	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		return results, fmt.Errorf("failed to read results for query %v", err)
	}
	for {
		err := it.Next(&result)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return results, fmt.Errorf("Failed to get parse play result %v", err)
		}
		results = append(results, result)
	}

	return results, nil
}

func artistsForLastNDays(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string, days int) ([]resultArtistFromPeriod, error) {
	var results []resultArtistFromPeriod
	queryString := fmt.Sprintf(
		`SELECT
		  artist,
		  count(track) as count
		FROM `+"`"+"%s.%s.%s"+"`"+
			`WHERE timestamp BETWEEN TIMESTAMP_ADD(CURRENT_TIMESTAMP(), INTERVAL -%d DAY) AND CURRENT_TIMESTAMP()
		GROUP BY artist
		ORDER BY count DESC
		LIMIT 10`,
		projectID,
		datasetName,
		tableName,
		days,
	)

	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		return results, fmt.Errorf("Failed to get artists for last %d days %v", days, err)
	}
	var result resultArtistFromPeriod
	for {
		err := it.Next(&result)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return results, fmt.Errorf("Failed to get parse artist result %v", err)
		}
		results = append(results, result)
	}
	return results, nil
}
