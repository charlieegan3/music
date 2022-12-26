package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/dustin/go-humanize"
	"github.com/foolin/goview"
	"github.com/gorilla/mux"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/charlieegan3/music/pkg/tool/utils"
)

func BuildRecentHandler(projectID, datasetName, tablename, googleJSON string) func(http.ResponseWriter, *http.Request) {
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

		queryString := fmt.Sprintf(`
select track, artist, album, timestamp from %s
order by timestamp desc
limit 50
`, fmt.Sprintf("`%s.%s.%s`", projectID, datasetName, tablename))

		q := bigqueryClient.Query(queryString)
		it, err := q.Read(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		var rows []recentPlayRow
		for {
			var row recentPlayRow
			err := it.Next(&row)
			if err == iterator.Done {
				break
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			row.Artists = strings.Split(row.Artist, ", ")

			row.Artwork = fmt.Sprintf(
				"/artworks/%s/%s.jpg",
				utils.CRC32Hash(row.Artist),
				utils.CRC32Hash(row.Album),
			)

			row.AgoTime = humanize.Time(row.Timestamp)

			rows = append(rows, row)
		}

		format, _ := mux.Vars(r)["format"]
		if format == ".json" {
			d, err := json.MarshalIndent(struct {
				RecentPlays []recentPlayRow `json:"RecentPlays"`
			}{
				RecentPlays: rows,
			}, "", "  ")
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(d)
			return
		}

		err = gv.Render(
			w,
			http.StatusOK,
			"recent",
			goview.M{"Plays": rows},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}

type recentPlayRow struct {
	Track     string
	Artist    string
	Artists   []string
	Album     string
	Artwork   string
	AgoTime   string
	Timestamp time.Time
}
