package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/doug-martin/goqu/v9"
	"github.com/dustin/go-humanize"
	"github.com/foolin/goview"
	"github.com/gorilla/mux"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/charlieegan3/music/pkg/tool/utils"
)

func BuildArtistAlbumTrackHandler(db *sql.DB, projectID, datasetName, tablename, googleJSON string) func(http.ResponseWriter, *http.Request) {

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

		albumSlug, ok := mux.Vars(r)["albumSlug"]
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("invalid URL"))
			return
		}

		albumParts := strings.Split(albumSlug, "-")
		if len(albumParts) < 2 {
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
		albumID := albumParts[0]
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

		var albumName string
		_, err = goquDB.Select("name").
			From("music.name_index").
			Where(goqu.C("id").Eq(albumID)).ScanVal(&albumName)
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
  timestamp
FROM
  %s
WHERE
  (STARTS_WITH(artist, @artistName)
    OR CONTAINS_SUBSTR(artist, @artistNameWithComma))
  AND ( album = @albumName )
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
				Name:  "albumName",
				Value: albumName,
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
		var rows []artistAlbumSingleTrackRow
		for {
			var r artistAlbumSingleTrackRow
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

			r.TimestampString = humanize.Time(r.Timestamp)
			if r.Timestamp.Before(time.Now().Add(-24 * 30 * time.Hour)) {
				r.TimestampDetail = r.Timestamp.Format("2006-01-02")
			}

			rows = append(rows, r)
		}

		err = gv.Render(
			w,
			http.StatusOK,
			"album_track",
			goview.M{
				"TrackName":  trackName,
				"ArtistName": artistName,
				"AlbumName":  albumName,
				"Plays":      rows,
				"Artwork": fmt.Sprintf(
					"/artworks/%s/%s.jpg",
					utils.CRC32Hash(artistName),
					utils.CRC32Hash(albumName),
				),
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}

type artistAlbumSingleTrackRow struct {
	Artist          string
	Artists         []string
	Timestamp       time.Time
	TimestampString string
	TimestampDetail string
}
