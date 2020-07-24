package backup

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/charlieegan3/music/internal/pkg/config"
)

// Plays backs up the bigquery table data to GCS
func Plays(cfg config.Config) error {
	// Gather env config values
	projectID := cfg.Google.Project
	datasetName := cfg.Google.Dataset
	tableName := cfg.Google.Table
	enrichedTableName := cfg.Google.TableEnrich
	backupBucketName := cfg.Google.BucketBackup

	// get the credentials from json
	ctx := context.Background()

	// create a big query client to query for the music stats
	client, err := bigquery.NewClient(ctx, projectID)
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
