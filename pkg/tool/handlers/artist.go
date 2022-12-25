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
	"math"
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
where STARTS_WITH(artist, @artistName) or contains_substr(artist, @artistNameWithComma)
group by artist, album, track
order by count desc
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
		}

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
  artist,
  rank
FROM
  ranks
WHERE
  STARTS_WITH(artist, @artistName) 
  OR CONTAINS_SUBSTR(artist, @artistNameWithComma)
`,
			fmt.Sprintf(
				"`%s.%s.%s`",
				projectID,
				datasetName,
				tablename,
			),
		)

		q = bigqueryClient.Query(queryString)
		q.Parameters = []bigquery.QueryParameter{
			{
				Name:  "artistName",
				Value: artistName,
			},
			{
				Name:  "artistNameWithComma",
				Value: fmt.Sprintf(", %s", artistName),
			},
		}

		it, err = q.Read(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		var ranks []int64
		var best int64
		best = math.MaxInt64
		isPrimaryArtist := true
		for {
			var row struct {
				Rank   int64
				Artist string
			}
			err = it.Next(&row)
			if err == iterator.Done {
				break
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			ranks = append(ranks, row.Rank)

			if row.Artist != artistName {
				isPrimaryArtist = false
			}
			if row.Rank < best {
				best = row.Rank
			}
		}

		var rankString string
		if len(ranks) > 0 || !isPrimaryArtist {
			if len(ranks) > 1 || !isPrimaryArtist {
				rankString = fmt.Sprintf("Artist Rank #%d (including colabs)", best)
			} else {
				rankString = fmt.Sprintf("Artist Rank #%d", best)
			}
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
	Artist  string
	Artists []string
	Track   string
	Artwork string
	Count   int64
}
