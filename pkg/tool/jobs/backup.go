package jobs

import (
	"cloud.google.com/go/bigquery"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"google.golang.org/api/option"
	"time"
)

// Backup is a job that copies the data in the bq table to GCS
type Backup struct {
	DB *sql.DB

	ScheduleOverride string

	GoogleCredentialsJSON string
	ProjectID             string
	DatasetName           string
	TableName             string
	BackupBucketName      string
}

func (b *Backup) Name() string {
	return "backup"
}

func (b *Backup) Run(ctx context.Context) error {
	doneCh := make(chan bool)
	errCh := make(chan error)

	go func() {
		// create a big query client to query for the music stats
		client, err := bigquery.NewClient(
			ctx,
			b.ProjectID,
			option.WithCredentialsJSON([]byte(b.GoogleCredentialsJSON)),
		)
		if err != nil {
			errCh <- fmt.Errorf("failed to create client: %v", err)
			return
		}

		// create handles to the dataset and table
		dataset := client.Dataset(b.DatasetName)
		table := dataset.Table(b.TableName)

		// create a handle to a gcs location
		// year-mm-dd-time
		name := "/plays-backup-" + time.Now().UTC().Format("2006-01-02-1504") + ".json"
		gcsRef := &bigquery.GCSReference{
			URIs:              []string{"gs://" + b.BackupBucketName + name},
			DestinationFormat: bigquery.JSON,
		}

		extractor := table.ExtractorTo(gcsRef)

		job, err := extractor.Run(ctx)
		if err != nil {
			errCh <- fmt.Errorf("create backup job failed: %v", err)
			return
		}
		_, err = job.Wait(ctx)
		if err != nil {
			errCh <- fmt.Errorf("job failed: %v", err)
			return
		}

		// create a handle to a gcs location latest
		name = "/plays-backup-latest.json"
		gcsRef = &bigquery.GCSReference{
			URIs:              []string{"gs://" + b.BackupBucketName + name},
			DestinationFormat: bigquery.JSON,
		}

		extractor = table.ExtractorTo(gcsRef)

		job, err = extractor.Run(ctx)
		if err != nil {
			errCh <- fmt.Errorf("Create backup job failed: %v", err)
			return
		}
		_, err = job.Wait(ctx)
		if err != nil {
			errCh <- fmt.Errorf("job failed: %v", err)
			return
		}

		doneCh <- true
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-errCh:
		return fmt.Errorf("job failed with error: %b", e)
	case <-doneCh:
		return nil
	}
}

func (b *Backup) Timeout() time.Duration {
	return 30 * time.Second
}

func (b *Backup) Schedule() string {
	if b.ScheduleOverride != "" {
		return b.ScheduleOverride
	}
	return "0 0 6 * * *"
}
