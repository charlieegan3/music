package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/foolin/goview"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/charlieegan3/music/pkg/tool/utils"
)

func BuildMonthsHandler(projectID, datasetName, tablename, googleJSON string) func(http.ResponseWriter, *http.Request) {
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
		monthFormat := "%Y-%m"
		monthFormatPretty := "%B %Y"
		queryString := fmt.Sprintf(`
  SELECT
  month,
  MAX(pretty) AS pretty,
  ARRAY_AGG(STRUCT(track,
      artist,
      album,
      count)
  ORDER BY
    count DESC
  LIMIT
    10) AS top
FROM (
  SELECT
    COUNT(track) AS count,
    track,
    artist,
    MAX(album) as album,
    month,
    pretty,
    MAX(album_cover) AS artwork,
    MAX(spotify_id) AS spotify
  FROM (
    SELECT
      *,
      FORMAT_DATE('%s', DATE(timestamp)) AS month,
      FORMAT_DATE('%s', DATE(timestamp)) AS pretty
    FROM
      %s)
  GROUP BY
    track,
    artist,
    month,
    pretty
  ORDER BY
    count DESC )
GROUP BY
  month
ORDER BY
  month desc
`, monthFormat, monthFormatPretty, tableName)

		q := bigqueryClient.Query(queryString)

		it, err := q.Read(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		monthsTopTracks := []monthTopTracks{}
		for {
			var r monthTopTracks
			err := it.Next(&r)
			if err == iterator.Done {
				break
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			for i := range r.TopTracks {
				t := &r.TopTracks[i]

				t.Artists = strings.Split(t.Artist, ", ")

				t.Artwork = fmt.Sprintf(
					"/artworks/%s/%s.jpg",
					utils.CRC32Hash(t.Artist),
					utils.CRC32Hash(t.Album),
				)
			}

			monthsTopTracks = append(monthsTopTracks, r)
		}

		err = gv.Render(
			w,
			http.StatusOK,
			"months",
			goview.M{
				"Months": monthsTopTracks,
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}

type monthTopTracks struct {
	Month     string
	Pretty    string
	TopTracks []monthTopTrack `bigquery:"top"`
}

type monthTopTrack struct {
	Track  string
	Artist string
	Album  string
	Count  int

	Artwork string
	Artists []string
}
