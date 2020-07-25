package uploader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"

	bq "github.com/charlieegan3/music/internal/pkg/bigquery"
	"github.com/charlieegan3/music/internal/pkg/config"
)

type submission struct {
	Key     string
	Track   string
	Artist  string
	Message string
}

// EnableUpload is set in flag from the command
var EnableUpload = false

// Serve runs a server capable of uploading play data
func Serve(cfg config.Config) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var s submission
		if err := decoder.Decode(&s); err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("failed to parse json"))
			return
		}

		if s.Key != cfg.Uploader.Token {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}

		s, err := parseMessage(s.Message)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("failed to parse message"))
			return
		}

		fmt.Printf("Saving %s by %s\n", s.Track, s.Artist)
		if EnableUpload {
			if uploadTrack(cfg, s) != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("failed to upload track"))
				return
			}
		}

		w.Write([]byte("completed"))
	})

	return http.ListenAndServe(":8080", nil)
}

func parseMessage(message string) (submission, error) {
	var s submission

	parts := strings.Split(message, " by ")

	if len(parts) > 2 {
		parts = []string{strings.Join(parts[0:len(parts)-1], " by "), parts[len(parts)-1]}
	}

	s.Artist, s.Track = strings.TrimSpace(parts[len(parts)-1]), strings.TrimSpace(parts[0])

	return s, nil
}

func uploadTrack(cfg config.Config, s submission) error {
	// Creates a bq client.
	ctx := context.Background()
	projectID := cfg.Google.Project
	datasetName := cfg.Google.Dataset
	tableName := cfg.Google.Table

	bigqueryClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("Failed to create client: %v", err)
	}
	// loads in the table schema from file
	jsonSchema, err := ioutil.ReadFile("schema.json")
	if err != nil {
		return fmt.Errorf("Failed to create schema: %v", err)
	}
	schema, err := bigquery.SchemaFromJSON(jsonSchema)
	if err != nil {
		return fmt.Errorf("Failed to parse schema: %v", err)
	}
	u := bigqueryClient.Dataset(datasetName).Table(tableName).Uploader()

	return bq.SavePlay(ctx,
		schema,
		*u,
		s.Track,
		s.Artist,
		"",
		fmt.Sprintf("%d", time.Now().UTC().Unix()),
		0,
		"", // spotify
		"", // image
		"now_playing",
		"", // youtube_id
		"", // youtube_category_id
		"", // soundcloud_id
		"", // soundcloud_permalink
		"", // shazam_id
		"", // shazam_permalink
	)
}
