package jobs

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/doug-martin/goqu/v9"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// CoversSync is a job that maintains a list of artists and
// album pairings in the database based on data in bigquery
type CoversSync struct {
	DB *sql.DB

	ScheduleOverride string

	GoogleCredentialsJSON string
	ProjectID             string
	DatasetName           string
	TableName             string
}

func (s *CoversSync) Name() string {
	return "covers-sync"
}

func (s *CoversSync) Run(ctx context.Context) error {
	doneCh := make(chan bool)
	errCh := make(chan error)

	go func() {
		bigqueryClient, err := bigquery.NewClient(
			ctx,
			s.ProjectID,
			option.WithCredentialsJSON([]byte(s.GoogleCredentialsJSON)),
		)
		if err != nil {
			errCh <- fmt.Errorf("failed to create bq client: %v", err)
			return
		}

		queryString := "select distinct artist, album from `charlieegan3-music-001.music.plays` order by artist, album;"
		q := bigqueryClient.Query(queryString)
		it, err := q.Read(ctx)
		if err != nil {
			errCh <- fmt.Errorf("failed to read from bq: %v", err)
		}
		var rows []goqu.Record
		for {
			var r artistAlbumRow
			err := it.Next(&r)
			if err == iterator.Done {
				break
			}
			if err != nil {
				errCh <- fmt.Errorf("failed to read row from bq result: %v", err)
			}

			rows = append(rows, goqu.Record{"artist": r.Artist, "album": r.Album})
		}

		goquDB := goqu.New("postgres", s.DB)
		exe := goquDB.Insert("music.covers").Rows(rows).OnConflict(goqu.DoNothing()).Executor()
		res, err := exe.Exec()
		if err != nil {
			errCh <- fmt.Errorf("failed to generate insert sql: %v", err)
		}
		rowCount, err := res.RowsAffected()
		if err != nil {
			errCh <- fmt.Errorf("failed to get row count: %v", err)
		}

		fmt.Println("New covers:", rowCount)

		doneCh <- true
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-errCh:
		return fmt.Errorf("job failed with error: %s", e)
	case <-doneCh:
		return nil
	}
}

func (s *CoversSync) Timeout() time.Duration {
	return 30 * time.Second
}

func (s *CoversSync) Schedule() string {
	if s.ScheduleOverride != "" {
		return s.ScheduleOverride
	}
	return "0 0 6 * * *"
}

type artistAlbumRow struct {
	Artist string
	Album  string
}
