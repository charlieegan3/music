package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/doug-martin/goqu/v9"
	"github.com/foolin/goview"
	"github.com/gorilla/mux"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/charlieegan3/music/pkg/tool/utils"
)

func BuildArtistAlbumHandler(db *sql.DB, projectID, datasetName, tablename, googleJSON string) func(http.ResponseWriter, *http.Request) {

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

		artistID := artistParts[0]
		albumID := albumParts[0]

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
  track,
  COUNT(track) AS count
FROM
  %s
WHERE
  (STARTS_WITH(artist, @artistName)
    OR CONTAINS_SUBSTR(artist, @artistNameWithComma))
  AND ( album = @albumName )
GROUP BY
  artist,
  album,
  track
ORDER BY
  count DESC
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
		}

		it, err := q.Read(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		var rows []artistAlbumTrackRow
		var total int64
		for {
			var r artistAlbumTrackRow
			err := it.Next(&r)
			if err == iterator.Done {
				break
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			r.Artwork = fmt.Sprintf(
				"/artworks/%s/%s.jpg",
				utils.CRC32Hash(r.Artist),
				utils.CRC32Hash(r.Album),
			)

			for _, a := range strings.Split(r.Artist, ", ") {
				if a != artistName {
					r.Artists = append(r.Artists, a)
				}
			}

			total += r.Count

			rows = append(rows, r)
		}

		err = gv.Render(
			w,
			http.StatusOK,
			"album",
			goview.M{
				"ArtistName": artistName,
				"AlbumName":  albumName,
				"Tracks":     rows,
				"Total":      total,
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}

type artistAlbumTrackRow struct {
	Album   string
	Artist  string
	Artists []string
	Track   string
	Artwork string
	Count   int64
}
