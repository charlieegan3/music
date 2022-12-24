package handlers

import (
	"cloud.google.com/go/bigquery"
	"fmt"
	"github.com/charlieegan3/music/pkg/tool/utils"
	"github.com/dustin/go-humanize"
	"github.com/foolin/goview"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"net/http"
	"strings"
	"time"
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
			var r recentPlayRow
			err := it.Next(&r)
			if err == iterator.Done {
				break
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			r.Artists = strings.Split(r.Artist, ",")

			r.Artwork = fmt.Sprintf(
				"/artworks/%s/%s.jpg",
				utils.CRC32Hash(r.Artist),
				utils.CRC32Hash(r.Album),
			)

			r.AgoTime = humanize.Time(r.Timestamp)

			rows = append(rows, r)
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
