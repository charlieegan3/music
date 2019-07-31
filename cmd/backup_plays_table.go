package main

import (
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

// BackupPlaysTable gets a list of recently played tracks
func BackupPlaysTable() error {
	// Gather env config values
	projectID := os.Getenv("GOOGLE_PROJECT")
	datasetName := os.Getenv("GOOGLE_DATASET")
	tableName := os.Getenv("GOOGLE_TABLE")
	enrichedTableName := os.Getenv("GOOGLE_TABLE_ENRICHED")
	backupBucketName := os.Getenv("GOOGLE_BACKUP_BUCKET")
	ctx := context.Background()

	httpClient, err := getGoogleHTTPClient()
	if err != nil {
		return fmt.Errorf("Failed to get auth %s", err)
	}

	// create a big query client to query for the music stats
	client, err := bigquery.NewClient(ctx, projectID, option.WithHTTPClient(&httpClient))
	if err != nil {
		return fmt.Errorf("Failed to create client: %v", err)
	}

	// create handles to the dataset and table
	dataset := client.Dataset(datasetName)
	table := dataset.Table(tableName)

	// create a handle to a gcs location
	// year-mm-dd-time
	name := "/plays-backup-" + time.Now().UTC().Format("2006-01-02-1504") + ".json"
	gcsRef := &bigquery.GCSReference{
		URIs:              []string{"gs://" + backupBucketName + name},
		DestinationFormat: bigquery.JSON,
	}

	extractor := table.ExtractorTo(gcsRef)

	job, err := extractor.Run(ctx)
	if err != nil {
		return fmt.Errorf("Create backup job failed: %v", err)
	}
	_, err = job.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Job failed: %v", err)
	}

	// create a handle to a gcs location
	// latest
	name = "/plays-backup-latest.json"
	gcsRef = &bigquery.GCSReference{
		URIs:              []string{"gs://" + backupBucketName + name},
		DestinationFormat: bigquery.JSON,
	}

	extractor = table.ExtractorTo(gcsRef)

	job, err = extractor.Run(ctx)
	if err != nil {
		return fmt.Errorf("Create backup job failed: %v", err)
	}
	_, err = job.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Job failed: %v", err)
	}

	// create a handle to a gcs location
	// enriched latest
	name = "/enriched-backup-latest.json"
	gcsRef = &bigquery.GCSReference{
		URIs:              []string{"gs://" + backupBucketName + name},
		DestinationFormat: bigquery.JSON,
	}

	extractor = dataset.Table(enrichedTableName).ExtractorTo(gcsRef)

	job, err = extractor.Run(ctx)
	if err != nil {
		return fmt.Errorf("Create backup job failed: %v", err)
	}
	_, err = job.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Job failed: %v", err)
	}

	return nil
}
