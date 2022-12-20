package jobs

import (
	"bytes"
	"cloud.google.com/go/storage"
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"google.golang.org/api/option"
	"hash/crc32"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
)

var coverSizePerferenceOrder = []string{
	"mega",
	"extralarge",
	"large",
	"medium",
	"small",
}

// CoversStore is a job that takes the covers in the database and makes sure that
// they are all stored in the covers bucket
type CoversStore struct {
	DB               *sql.DB
	ScheduleOverride string
	LastFMAPIKey     string

	GoogleCredentialsJSON string
	GoogleBucketName      string
}

func (s *CoversStore) Name() string {
	return "covers-store"
}

func (s *CoversStore) Run(ctx context.Context) error {
	doneCh := make(chan bool)
	errCh := make(chan error)

	storageClient, err := storage.NewClient(ctx, option.WithCredentialsJSON([]byte(s.GoogleCredentialsJSON)))
	if err != nil {
		return fmt.Errorf("failed to create google storage client: %v", err)
	}
	defer storageClient.Close()

	go func() {

		goquDB := goqu.New("postgres", s.DB)
		exe := goquDB.From("music.covers").
			Select("id", "artist", "album", "url").
			Where(
				goqu.And(
					goqu.C("completed").IsFalse(),
					goqu.C("error_count").Lt(3),
				),
			).
			Order(goqu.I("artist").Asc()).
			Executor()

		var rows []struct {
			ID     int64  `db:"id"`
			Artist string `db:"artist"`
			Album  string `db:"album"`
			URL    string `db:"url"`
		}

		err := exe.ScanStructs(&rows)
		if err != nil {
			errCh <- fmt.Errorf("failed to select: %v", err)
		}

		for _, row := range rows {
			var err error
			var resp io.Reader
			var contentType string

			if row.URL != "" {
				contentType, resp, err = loadRowArt(row.URL)
				if !strings.HasPrefix(contentType, "image/") {
					err = fmt.Errorf("content type is not an image: %s", contentType)
				}
				if err != nil {
					log.Printf("failed to load row art (searching instead): %v\n", err)
					contentType, resp, err = findCoverArt(s.LastFMAPIKey, row.Artist, row.Album)
					if err != nil {
						log.Printf("failed to find cover art: %v\n", err)
						updateAsError(goquDB, row.ID, false)
						continue
					}
				}
			}

			switch contentType {
			case "image/jpeg", "image/jpg":
			// do nothing
			case "image/png":
				img, err := png.Decode(resp)
				if err != nil {
					updateAsError(goquDB, row.ID, false)
					log.Printf("failed to read png for artist: %s, album: %s: %v\n", row.Artist, row.Album, err)
					continue
				}

				buf := new(bytes.Buffer)
				err = jpeg.Encode(buf, img, nil)
				if err != nil {
					updateAsError(goquDB, row.ID, false)
					log.Printf("failed to encode jpg for artist: %s, album: %s: %v\n", row.Artist, row.Album, err)
					continue
				}
				resp = bytes.NewReader(buf.Bytes())
			default:
				updateAsError(goquDB, row.ID, false)
				log.Printf("unknown content type '%s' for artist: %s, album: %s\n", contentType, row.Artist, row.Album)
				continue
			}

			bkt := storageClient.Bucket(s.GoogleBucketName)
			obj := bkt.Object(fmt.Sprintf("%s/%s.jpg", crc32Hash(row.Artist), crc32Hash(row.Album)))
			w := obj.NewWriter(ctx)
			_, err = io.Copy(w, resp)
			if err != nil {
				updateAsError(goquDB, row.ID, false)
				log.Printf("failed to copy cover for %s/%s: %v\n", row.Artist, row.Album, err)
				continue
			}
			err = w.Close()
			if err != nil {
				updateAsError(goquDB, row.ID, false)
				log.Printf("failed to close cover for %s/%s: %v\n", row.Artist, row.Album, err)
				continue
			}

			updateAsCompleted(goquDB, row.ID)

			log.Println("stored", row.Artist, row.Album)
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

func (s *CoversStore) Timeout() time.Duration {
	return 30 * time.Second
}

func (s *CoversStore) Schedule() string {
	if s.ScheduleOverride != "" {
		return s.ScheduleOverride
	}
	return "0 0 6 * * *"
}

func crc32Hash(input string) string {
	return fmt.Sprintf("%d", crc32.ChecksumIEEE([]byte(input)))
}

func updateAsCompleted(goquDB *goqu.Database, rowID int64) {
	upd := goquDB.Update("music.covers").
		Set(goqu.Record{
			"completed": true,
		}).
		Where(goqu.C("id").Eq(rowID)).
		Executor()

	_, err := upd.Exec()
	if err != nil {
		log.Printf("failed to update row %d as completed: %v\n", rowID, err)
	}
}

func updateAsError(goquDB *goqu.Database, rowID int64, fatal bool) {
	errorCount := 1
	if fatal {
		errorCount = 5
	}
	upd := goquDB.Update("music.covers").
		Set(goqu.Record{
			"error_count": goqu.L(fmt.Sprintf("error_count + %d", errorCount)),
		}).
		Where(goqu.C("id").Eq(rowID)).
		Executor()

	_, err := upd.Exec()
	if err != nil {
		log.Printf("failed to update row %d with error: %v\n", rowID, err)
	}
}

func findCoverArt(lastFMAPIKey, artist, album string) (string, io.Reader, error) {
	client := &http.Client{}

	urlParams := url.Values{
		"api_key": []string{lastFMAPIKey},
		"method":  []string{"album.getinfo"},
		"artist":  []string{artist},
		"album":   []string{album},
		"format":  []string{"json"},
	}
	formattedURL, err := url.Parse("https://ws.audioscrobbler.com/2.0/?" + urlParams.Encode())
	if err != nil {
		return "", nil, fmt.Errorf("failed to url params: %v", err)
	}

	req, err := http.NewRequest("GET", formattedURL.String(), nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to do request: %v", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var response struct {
		Album struct {
			Image []struct {
				Text string `json:"#text"`
				Size string `json:"size"`
			} `json:"image"`
		} `json:"album"`
	}

	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return "", nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	var selectedImage string
	for _, image := range response.Album.Image {
		for _, size := range coverSizePerferenceOrder {
			if image.Size == size {
				selectedImage = image.Text
				break
			}
		}
	}

	if selectedImage == "" {
		return "", nil, fmt.Errorf("failed to find image suitable image")
	}

	req, err = http.NewRequest("GET", selectedImage, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err = client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to do request: %v", err)
	}

	contentType := "unknown"
	if resp.Header.Get("Content-Type") != "" {
		contentType = resp.Header.Get("Content-Type")
	}

	return contentType, resp.Body, nil
}

func loadRowArt(url string) (string, io.Reader, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to do request: %v", err)
	}

	contentType := "unknown"
	if resp.Header.Get("Content-Type") != "" {
		contentType = resp.Header.Get("Content-Type")
	}

	return contentType, resp.Body, nil
}
