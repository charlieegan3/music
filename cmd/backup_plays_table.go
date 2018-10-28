package main

import (
	"log"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// BackupPlaysTable gets a list of recently played tracks
func BackupPlaysTable() {
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
		log.Fatalf("Creds parse failed: %v", err)
		os.Exit(1)
	}

	// create a big query client to query for the music stats
	client, err := bigquery.NewClient(ctx, projectID, option.WithCredentials(creds))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
		os.Exit(1)
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
		log.Fatalf("Create backup job failed: %v", err)
		os.Exit(1)
	}
	_, err = job.Wait(ctx)
	if err != nil {
		log.Fatalf("Job failed: %v", err)
		os.Exit(1)
	}
}
