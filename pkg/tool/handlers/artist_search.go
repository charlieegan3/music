package handlers

import (
	"fmt"
	"net/http"

	"cloud.google.com/go/bigquery"
	"github.com/foolin/goview"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/charlieegan3/music/pkg/tool/utils"
)

func BuildArtistSearchHandler(projectID, datasetName, tablename, googleJSON string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		if r.URL.Query().Get("q") == "" {
			err = gv.Render(
				w,
				http.StatusOK,
				"artist_search_index",
				goview.M{},
			)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
			}
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
  DISTINCT artist
FROM
  %s
WHERE
  CONTAINS_SUBSTR(LOWER(artist), @query)
ORDER BY
  LENGTH(artist) asc
`, fmt.Sprintf("`%s.%s.%s`", projectID, datasetName, tablename))

		q := bigqueryClient.Query(queryString)
		q.Parameters = []bigquery.QueryParameter{
			{
				Name:  "query",
				Value: r.URL.Query().Get("q"),
			},
		}

		it, err := q.Read(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		var artists []string
		for {
			var r struct {
				Artist string
			}
			err := it.Next(&r)
			if err == iterator.Done {
				break
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			artists = append(artists, r.Artist)
		}

		if len(artists) == 1 {
			http.Redirect(w, r, fmt.Sprintf("/artists/%s", utils.NameSlug(artists[0])), http.StatusFound)
			return
		}

		err = gv.Render(
			w,
			http.StatusOK,
			"artist_search_results",
			goview.M{"Artists": artists},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}
