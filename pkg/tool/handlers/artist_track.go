package handlers

import (
	"cloud.google.com/go/bigquery"
	"database/sql"
	"fmt"
	"github.com/charlieegan3/music/pkg/tool/utils"
	"github.com/doug-martin/goqu/v9"
	"github.com/dustin/go-humanize"
	"github.com/foolin/goview"
	"github.com/gorilla/mux"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"net/http"
	"strings"
	"time"
)

func BuildArtistTrackHandler(db *sql.DB, projectID, datasetName, tablename, googleJSON string) func(http.ResponseWriter, *http.Request) {

	goquDB := goqu.New("postgres", db)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		artistSlug, ok := mux.Vars(r)["artistSlug"]
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("invalid URL"))
			return
		}

		artistParts := strings.Split(artistSlug, "-")
		if len(artistParts) < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("invalid URL"))
			return
		}

		trackSlug, ok := mux.Vars(r)["trackSlug"]
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("invalid URL"))
			return
		}

		trackParts := strings.Split(trackSlug, "-")
		if len(trackParts) < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("invalid URL"))
			return
		}

		artistID := artistParts[0]
		trackID := trackParts[0]

		var artistName string
		_, err = goquDB.Select("name").
			From("music.name_index").
			Where(goqu.C("id").Eq(artistID)).ScanVal(&artistName)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		var trackName string
		_, err = goquDB.Select("name").
			From("music.name_index").
			Where(goqu.C("id").Eq(trackID)).ScanVal(&trackName)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

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

		queryString := fmt.Sprintf(`
SELECT
  artist,
  album,
  timestamp
FROM
  %s
WHERE
  (STARTS_WITH(artist, @artistName)
    OR CONTAINS_SUBSTR(artist, @artistNameWithComma))
  AND ( track = @trackName )
ORDER BY
  timestamp desc
`,
			fmt.Sprintf(
				"`%s.%s.%s`",
				projectID,
				datasetName,
				tablename,
			),
		)
		q := bigqueryClient.Query(queryString)
		q.Parameters = []bigquery.QueryParameter{
			{
				Name:  "artistName",
				Value: artistName,
			},
			{
				Name:  "artistNameWithComma",
				Value: fmt.Sprintf(", %s", artistName),
			},
			{
				Name:  "trackName",
				Value: trackName,
			},
		}

		it, err := q.Read(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		var rows []artistSingleTrackRow
		for {
			var r artistSingleTrackRow
			err := it.Next(&r)
			if err == iterator.Done {
				break
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			for _, a := range strings.Split(r.Artist, ", ") {
				if a != artistName {
					r.Artists = append(r.Artists, a)
				}
			}

			r.Artwork = fmt.Sprintf(
				"/artworks/%s/%s.jpg",
				utils.CRC32Hash(r.Artist),
				utils.CRC32Hash(r.Album),
			)

			r.TimestampString = humanize.Time(r.Timestamp)
			if r.Timestamp.Before(time.Now().Add(-24 * 30 * time.Hour)) {
				r.TimestampDetail = r.Timestamp.Format("2006-01-02")
			}

			rows = append(rows, r)
		}

		err = gv.Render(
			w,
			http.StatusOK,
			"artist_track",
			goview.M{
				"TrackName":  trackName,
				"ArtistName": artistName,
				"Plays":      rows,
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}

type artistSingleTrackRow struct {
	Artist          string
	Album           string
	Artists         []string
	Artwork         string
	Timestamp       time.Time
	TimestampString string
	TimestampDetail string
}