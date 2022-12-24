package handlers

import (
	"cloud.google.com/go/bigquery"
	"database/sql"
	"fmt"
	"github.com/charlieegan3/music/pkg/tool/utils"
	"github.com/doug-martin/goqu/v9"
	"github.com/foolin/goview"
	"github.com/gorilla/mux"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"net/http"
	"strings"
)

func BuildArtistHandler(db *sql.DB, projectID, datasetName, tablename, googleJSON string) func(http.ResponseWriter, *http.Request) {

	goquDB := goqu.New("postgres", db)

	return func(w http.ResponseWriter, r *http.Request) {
		artistSlug, ok := mux.Vars(r)["artistSlug"]
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("invalid URL"))
			return
		}

		parts := strings.Split(artistSlug, "-")
		if len(parts) < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("invalid URL"))
			return
		}

		artistID := parts[0]

		var artistName string
		_, err := goquDB.Select("name").
			From("music.artists").
			Where(goqu.C("id").Eq(artistID)).ScanVal(&artistName)
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
select artist, album, track, count(track) as count from %s
where artist = %q
group by artist, album, track
order by count desc
`,
			fmt.Sprintf(
				"`%s.%s.%s`",
				projectID,
				datasetName,
				tablename,
			),
			artistName)

		q := bigqueryClient.Query(queryString)
		it, err := q.Read(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		var rows []artistTrackRow
		var total int64
		for {
			var r artistTrackRow
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
				utils.CRC32Hash(artistName),
				utils.CRC32Hash(r.Album),
			)

			total += r.Count

			rows = append(rows, r)
		}

		queryString = fmt.Sprintf(`
WITH
  artists AS (
  SELECT
    artist,
    COUNT(track) AS count
  FROM
	%s
  GROUP BY
    artist
  ORDER BY
    count DESC ),
  ranks AS (
  SELECT
    ROW_NUMBER() OVER (ORDER BY count DESC) AS rank,
    artist,
    count
  FROM
    artists
  ORDER BY
    count DESC)
SELECT
  rank
FROM
  ranks
WHERE
  artist = %q
`,
			fmt.Sprintf(
				"`%s.%s.%s`",
				projectID,
				datasetName,
				tablename,
			),
			artistName)

		q = bigqueryClient.Query(queryString)
		it, err = q.Read(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		var rank int64
		var row struct {
			Rank int64
		}
		err = it.Next(&row)
		if err != nil && err != iterator.Done {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		rank = row.Rank

		var rankString string
		if rank > 0 {
			rankString = fmt.Sprintf("Artist Rank #%d", rank)
		}

		err = gv.Render(
			w,
			http.StatusOK,
			"artist",
			goview.M{
				"ArtistName": artistName,
				"Tracks":     rows,
				"Total":      total,
				"Rank":       rankString,
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}

type artistTrackRow struct {
	Album   string
	Track   string
	Artwork string
	Count   int64
}
