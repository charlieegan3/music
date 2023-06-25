package jobs

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"

	"github.com/charlieegan3/music/pkg/tool/bq"
	"github.com/charlieegan3/music/pkg/tool/utils"
)

const lastFMSourceName = "lastfm"

// LastFMSync is a job that syncs last fm plays to bigquery
type LastFMSync struct {
	ScheduleOverride string
	Endpoint         string

	APIKey   string
	Username string

	GoogleCredentialsJSON string
	ProjectID             string
	DatasetName           string
	TableName             string
}

func (s *LastFMSync) Name() string {
	return "lastfm-sync"
}

func (s *LastFMSync) Run(ctx context.Context) error {
	doneCh := make(chan bool)
	errCh := make(chan error)

	go func() {

		bigqueryClient, err := bigquery.NewClient(
			ctx,
			s.ProjectID,
			option.WithCredentialsJSON([]byte(s.GoogleCredentialsJSON)),
		)
		if err != nil {
			errCh <- fmt.Errorf("failed to create bq client: %v", err)
			return
		}
		schema, err := bigquery.SchemaFromJSON([]byte(bq.JSONSchema))
		if err != nil {
			errCh <- fmt.Errorf("failed to parse schema: %v", err)
			return
		}

		client := &http.Client{}

		req, err := http.NewRequest(
			"GET",
			fmt.Sprintf(
				"http://ws.audioscrobbler.com/2.0/?method=user.getrecenttracks&user=%s&api_key=%s&format=json",
				s.Username,
				s.APIKey,
			),
			nil,
		)

		resp, err := client.Do(req)
		if err != nil {
			errCh <- fmt.Errorf("failed to get last fm plays: %w", err)
			return
		}

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			errCh <- fmt.Errorf("failed to read response body: %w", err)
			return
		}

		var results lastFMResponse
		err = json.Unmarshal(respBody, &results)
		if err != nil {
			errCh <- fmt.Errorf("failed to unmarshal response body: %w", err)
			return
		}

		mostRecentPlays, err := utils.MostRecentTimestamps(ctx, bigqueryClient, s.ProjectID, s.DatasetName, s.TableName, lastFMSourceName, 1)
		if err != nil {
			errCh <- fmt.Errorf("failed to get most recent timestamp: %w", err)
			return
		}

		if len(mostRecentPlays) == 0 {
			errCh <- fmt.Errorf("no most recent timestamp found")
			return
		}

		mostRecentPlayTime := mostRecentPlays[0].UTC()

		var newCompletedPlays []play
		for _, p := range results.RecentTracks.Items {
			if p.Date.Timestamp == "" {
				continue
			}

			i, err := strconv.ParseInt(p.Date.Timestamp, 10, 64)
			if err != nil {
				errCh <- fmt.Errorf("failed to parse timestamp: %w", err)
				return
			}
			if time.Unix(i, 0).Before(mostRecentPlayTime) {
				continue
			}

			if len(newCompletedPlays) > 0 {
				lastPlay := newCompletedPlays[len(newCompletedPlays)-1]

				j, err := strconv.ParseInt(lastPlay.Date.Timestamp, 10, 64)
				if err != nil {
					errCh <- fmt.Errorf("failed to parse timestamp: %w", err)
					return
				}

				if math.Abs(float64(i-j)) < 5 {
					fmt.Printf("skipping %v as it's too close to the previous play\n", p.Name)
					continue
				}
			}

			newCompletedPlays = append(newCompletedPlays, p)
		}

		var vss []*bigquery.ValuesSaver
		for _, p := range newCompletedPlays {
			var image string
			if len(p.Image) > 0 {
				image = p.Image[len(p.Image)-1].Text
			}
			vss = append(vss, &bigquery.ValuesSaver{
				Schema:   schema,
				InsertID: fmt.Sprintf("%v", p.Date.Timestamp),
				Row: []bigquery.Value{
					p.Name,
					p.Artist.Name,
					p.Album.Name,
					p.Date.Timestamp,
					bigquery.NullInt64{Int64: 0, Valid: false},
					"",
					image,
					time.Now().Unix(),
					lastFMSourceName,
					"",
					"",
					"",
					"",
					"",
					"",
				},
			})
		}

		// upload the items
		inserter := bigqueryClient.Dataset(s.DatasetName).Table(s.TableName).Inserter()
		err = inserter.Put(ctx, vss)
		if err != nil {
			if pmErr, ok := err.(bigquery.PutMultiError); ok {
				for _, rowInsertionError := range pmErr {
					log.Println(rowInsertionError.Errors)
				}
			}

			errCh <- fmt.Errorf("failed to insert plays: %w", err)
		}

		for _, play := range newCompletedPlays {
			fmt.Fprintf(os.Stdout, "Inserted %s %s %s\n", play.Name, play.Artist.Name, play.Date.Timestamp)
		}

		doneCh <- true
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-errCh:
		return fmt.Errorf("job failed with error: %s", e)
	case <-doneCh:
		return nil
	}
}

func (s *LastFMSync) Timeout() time.Duration {
	return 30 * time.Second
}

func (s *LastFMSync) Schedule() string {
	if s.ScheduleOverride != "" {
		return s.ScheduleOverride
	}
	return "0 0 6 * * *"
}

type play struct {
	Album struct {
		Name string `json:"#text"`
		MBID string `json:"mbid"`
	} `json:"album"`
	Artist struct {
		Name string `json:"#text"`
		MBID string `json:"mbid"`
	} `json:"artist"`
	Date struct {
		Text      string `json:"#text"`
		Timestamp string `json:"uts"`
	} `json:"date"`
	Image []struct {
		Text string `json:"#text"`
		Size string `json:"size"`
	} `json:"image"`
	MBID       string `json:"mbid"`
	Name       string `json:"name"`
	Streamable string `json:"streamable"`
	URL        string `json:"url"`
}

type lastFMResponse struct {
	RecentTracks struct {
		Items []play `json:"track"`
	} `json:"recenttracks"`
}
