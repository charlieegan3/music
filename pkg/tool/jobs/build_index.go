package jobs

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"github.com/charlieegan3/music/pkg/tool/utils"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/doug-martin/goqu/v9"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// BuildIndex will create a mapping of crc32(artist/album/track) -> name in the database
type BuildIndex struct {
	DB *sql.DB

	ScheduleOverride string

	GoogleCredentialsJSON string
	ProjectID             string
	DatasetName           string
	TableName             string
}

func (a *BuildIndex) Name() string {
	return "build-index"
}

func (a *BuildIndex) Run(ctx context.Context) error {
	doneCh := make(chan bool)
	errCh := make(chan error)

	go func() {
		bigqueryClient, err := bigquery.NewClient(
			ctx,
			a.ProjectID,
			option.WithCredentialsJSON([]byte(a.GoogleCredentialsJSON)),
		)
		if err != nil {
			errCh <- fmt.Errorf("failed to create bq client: %v", err)
			return
		}

		var rows []goqu.Record

		queryString := fmt.Sprintf(`
select distinct artist from %s
order by artist asc
`, fmt.Sprintf("`%s.%s.%s`", a.ProjectID, a.DatasetName, a.TableName))
		q := bigqueryClient.Query(queryString)
		it, err := q.Read(ctx)
		if err != nil {
			errCh <- fmt.Errorf("failed to read from bq: %v", err)
		}
		for {
			var r struct {
				Artist string `bigquery:"artist"`
			}
			err := it.Next(&r)
			if err == iterator.Done {
				break
			}
			if err != nil {
				errCh <- fmt.Errorf("failed to read row from bq result: %v", err)
			}

			rows = append(rows, goqu.Record{
				"name": r.Artist,
				"id":   utils.CRC32Hash(r.Artist),
			})

			if strings.Contains(r.Artist, ",") {
				for _, artist := range strings.Split(r.Artist, ", ") {
					formattedName := strings.TrimSpace(artist)
					if formattedName == "" {
						continue
					}
					rows = append(rows, goqu.Record{
						"name": formattedName,
						"id":   utils.CRC32Hash(formattedName),
					})
				}
			}
		}
		queryString = fmt.Sprintf(`
select distinct album from %s
order by album asc
`, fmt.Sprintf("`%s.%s.%s`", a.ProjectID, a.DatasetName, a.TableName))
		q = bigqueryClient.Query(queryString)
		it, err = q.Read(ctx)
		if err != nil {
			errCh <- fmt.Errorf("failed to read from bq: %v", err)
		}
		for {
			var r struct {
				Album string `bigquery:"album"`
			}
			err := it.Next(&r)
			if err == iterator.Done {
				break
			}
			if err != nil {
				errCh <- fmt.Errorf("failed to read row from bq result: %v", err)
			}

			rows = append(rows, goqu.Record{
				"name": r.Album,
				"id":   utils.CRC32Hash(r.Album),
			})
		}

		queryString = fmt.Sprintf(`
select distinct track from %s
order by track asc
`, fmt.Sprintf("`%s.%s.%s`", a.ProjectID, a.DatasetName, a.TableName))
		q = bigqueryClient.Query(queryString)
		it, err = q.Read(ctx)
		if err != nil {
			errCh <- fmt.Errorf("failed to read from bq: %v", err)
		}
		for {
			var r struct {
				Track string `bigquery:"track"`
			}
			err := it.Next(&r)
			if err == iterator.Done {
				break
			}
			if err != nil {
				errCh <- fmt.Errorf("failed to read row from bq result: %v", err)
			}

			rows = append(rows, goqu.Record{
				"name": r.Track,
				"id":   utils.CRC32Hash(r.Track),
			})
		}

		goquDB := goqu.New("postgres", a.DB)
		query := goquDB.Insert("music.name_index").Rows(rows).OnConflict(goqu.DoNothing())
		res, err := query.Executor().ExecContext(ctx)
		if err != nil {
			errCh <- fmt.Errorf("failed to insert: %v", err)
			return
		}
		rowCount, err := res.RowsAffected()
		if err != nil {
			errCh <- fmt.Errorf("failed to get row count: %v", err)
			return
		}

		log.Println("New rows:", rowCount)

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

func (a *BuildIndex) Timeout() time.Duration {
	return 30 * time.Second
}

func (a *BuildIndex) Schedule() string {
	if a.ScheduleOverride != "" {
		return a.ScheduleOverride
	}
	return "0 0 6 * * *"
}
