package main

import (
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// BackupPlaysTable gets a list of recently played tracks
func BackupPlaysTable() error {
	// Gather env config values
	projectID := os.Getenv("GOOGLE_PROJECT")
	datasetName := os.Getenv("GOOGLE_DATASET")
	tableName := os.Getenv("GOOGLE_TABLE")
	accountJSON := os.Getenv("GOOGLE_JSON")
	backupBucketName := os.Getenv("GOOGLE_BACKUP_BUCKET")

	// get the credentials from json
	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(ctx, []byte(accountJSON), bigquery.Scope, storage.ScopeReadWrite)
	if err != nil {
		return fmt.Errorf("Creds parse failed: %v", err)
	}

	// create a big query client to query for the music stats
	client, err := bigquery.NewClient(ctx, projectID, option.WithCredentials(creds))
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

	return nil
}
