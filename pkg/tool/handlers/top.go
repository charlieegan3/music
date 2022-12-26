package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/foolin/goview"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/charlieegan3/music/pkg/tool/utils"
)

func BuildTopHandler(projectID, datasetName, tablename, googleJSON string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		bigqueryClient, err := bigquery.NewClient(
			r.Context(),
			projectID,
			option.WithCredentialsJSON([]byte(googleJSON)),
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		tableName := fmt.Sprintf("`%s.%s.%s`", projectID, datasetName, tablename)
		queryString := fmt.Sprintf(`
WITH
  month AS (
  SELECT
    "month" AS category,
    artist,
	MAX(album) as album,
    track,
    COUNT(track) AS count
  FROM
	%s
  WHERE
    timestamp > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)
  GROUP BY
    artist,
    track
  ORDER BY
    count DESC
  LIMIT
    10),
  year AS (
  SELECT
    "year" AS category,
    artist,
	MAX(album) as album,
    track,
    COUNT(track) AS count
  FROM
	%s
  WHERE
    timestamp > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 365 DAY)
  GROUP BY
    artist,
    track
  ORDER BY
    count DESC
  LIMIT
    10),
  alltime AS (
  SELECT
    "all" AS category,
    artist,
	MAX(album) as album,
    track,
    COUNT(track) AS count
  FROM
	%s
  GROUP BY
    artist,
    track
  ORDER BY
    count DESC
  LIMIT
    10)
SELECT
  *
FROM
  month
UNION ALL
SELECT
  *
FROM
  year
UNION ALL
SELECT
  *
FROM
  alltime
`, tableName, tableName, tableName)

		q := bigqueryClient.Query(queryString)

		it, err := q.Read(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		monthTop := []topPlayRow{}
		yearTop := []topPlayRow{}
		allTop := []topPlayRow{}

		for {
			var r topPlayRow
			err := it.Next(&r)
			if err == iterator.Done {
				break
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			r.Artists = strings.Split(r.Artist, ", ")

			r.Artwork = fmt.Sprintf(
				"/artworks/%s/%s.jpg",
				utils.CRC32Hash(r.Artist),
				utils.CRC32Hash(r.Album),
			)

			switch r.Category {
			case "month":
				monthTop = append(monthTop, r)
			case "year":
				yearTop = append(yearTop, r)
			case "all":
				allTop = append(allTop, r)
			}
		}

		err = gv.Render(
			w,
			http.StatusOK,
			"top",
			goview.M{
				"MonthTop": monthTop,
				"YearTop":  yearTop,
				"AllTop":   allTop,
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}

type topPlayRow struct {
	Category  string
	Track     string
	Artist    string
	Artists   []string
	Album     string
	Artwork   string
	AgoTime   string
	Count     int64
	Timestamp time.Time
}
