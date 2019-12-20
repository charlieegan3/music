package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
)

type rowBQ = map[string]bigquery.Value

type row struct {
	Album               string    `json:"album"`
	AlbumCover          string    `json:"album_cover"`
	Artist              string    `json:"artist"`
	CreatedAt           time.Time `json:"created_at"`
	Duration            int64     `json:"duration"`
	ShazamID            string    `json:"shazam_id"`
	ShazamPermalink     string    `json:"shazam_permalink"`
	SoundcloudID        string    `json:"soundcloud_id"`
	SoundcloudPermalink string    `json:"soundcloud_permalink"`
	Source              string    `json:"source"`
	SpotifyID           string    `json:"spotify_id"`
	Timestamp           time.Time `json:"timestamp"`
	Track               string    `json:"track"`
	YoutubeCategoryID   string    `json:"youtube_category_id"`
	YoutubeID           string    `json:"youtube_id"`

	TrackHash  string `json:"-"`
	ArtistHash string `json:"-"`
}

func (r *row) Hash() string {
	return r.TrackHash + r.ArtistHash
}

// Enrich downloads, formats/cleans and reuploads as a new table
func Enrich() error {
	// Creates a bq client.
	ctx := context.Background()
	projectID := os.Getenv("GOOGLE_PROJECT")
	datasetName := os.Getenv("GOOGLE_DATASET")
	sourceTableName := os.Getenv("GOOGLE_TABLE")
	enrichedTableName := os.Getenv("GOOGLE_TABLE_ENRICHED")

	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}
	dataset := client.Dataset(datasetName)
	// loads in the table schema from file
	jsonSchema, err := ioutil.ReadFile("schema.json")
	if err != nil {
		return fmt.Errorf("Failed to create schema: %v", err)
	}
	schema, err := bigquery.SchemaFromJSON(jsonSchema)
	if err != nil {
		return fmt.Errorf("Failed to parse schema: %v", err)
	}

	rows, err := downloadCurrentRawData(ctx, *client, projectID, datasetName, sourceTableName)
	if err != nil {
		return fmt.Errorf("failed to download and parse existing data: %v", err)
	}

	rows = setHashes(rows)
	rows = setNewDates(rows)
	rows = setMissingCreatedAtTimes(rows)
	rows = setOldRowSources(rows)
	rows = skipDupeNowPlaying(rows)
	rows = coalesceMetadata(rows)

	var lines []string
	for _, v := range rows {
		json, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed generate json %v, %+v", err, v)
		}
		lines = append(lines, string(json))
	}

	dataString := strings.Join(lines, "\n")

	rs := bigquery.NewReaderSource(strings.NewReader(dataString))
	rs.FileConfig.SourceFormat = bigquery.JSON
	rs.FileConfig.Schema = schema
	loader := dataset.Table(enrichedTableName).LoaderFrom(rs)
	loader.CreateDisposition = bigquery.CreateIfNeeded
	loader.WriteDisposition = bigquery.WriteTruncate
	job, err := loader.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to upload data with load job %v", err)
	}
	_, err = job.Wait(ctx)
	if err != nil {
		return fmt.Errorf("copy job failed to complete %v", err)
	}

	return nil
}

func downloadCurrentRawData(ctx context.Context, client bigquery.Client, projectID, datasetName, tableName string) ([]row, error) {
	var rows []row
	queryString := fmt.Sprintf(
		"SELECT * FROM `%s.%s.%s` order by timestamp desc",
		projectID,
		datasetName,
		tableName,
	)
	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		return rows, fmt.Errorf("Failed query for recent timestamps: %v", err)
	}
	var rowsbq []rowBQ
	for {
		var r rowBQ
		err := it.Next(&r)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return rows, fmt.Errorf("Failed reading results for time: %v", err)
		}
		rowsbq = append(rowsbq, r)
	}

	for _, v := range rowsbq {
		var r row

		r.Artist = v["artist"].(string)
		r.Track = v["track"].(string)

		r.CreatedAt = time.Unix(0, 0)
		if v["created_at"] != nil {
			r.CreatedAt = v["created_at"].(time.Time)
		}

		r.Timestamp = time.Unix(0, 0)
		if v["timestamp"] != nil {
			r.Timestamp = v["timestamp"].(time.Time)
		}

		if v["duration"] != nil {
			r.Duration = v["duration"].(int64)
		} else {
			r.Duration = 0
		}

		if v["album"] != nil {
			r.Album = v["album"].(string)
		}
		if v["album_cover"] != nil {
			r.AlbumCover = v["album_cover"].(string)
		}
		if v["source"] != nil {
			r.Source = v["source"].(string)
		}
		if v["shazam_id"] != nil {
			r.ShazamID = v["shazam_id"].(string)
		}
		if v["shazam_permalink"] != nil {
			r.ShazamPermalink = v["shazam_permalink"].(string)
		}
		if v["soundcloud_id"] != nil {
			r.SoundcloudID = v["soundcloud_id"].(string)
		}
		if v["soundcloud_permalink"] != nil {
			r.SoundcloudPermalink = v["soundcloud_permalink"].(string)
		}
		if v["spotify_id"] != nil {
			r.SpotifyID = v["spotify_id"].(string)
		}
		if v["youtube_id"] != nil {
			r.YoutubeID = v["youtube_id"].(string)
		}
		if v["youtube_category_id"] != nil {
			r.YoutubeCategoryID = v["youtube_category_id"].(string)
		}

		rows = append(rows, r)
	}

	return rows, nil
}

func uploadRows(ctx context.Context, uploader bigquery.Uploader, schema bigquery.Schema, rows []row) error {
	var vss []*bigquery.ValuesSaver
	for i, v := range rows {
		vss = append(vss, &bigquery.ValuesSaver{
			Schema:   schema,
			InsertID: fmt.Sprintf("%v-%v-%v", i, v.Hash(), time.Now().Unix()),
			Row: []bigquery.Value{
				fmt.Sprintf("%s", v.Track),
				fmt.Sprintf("%s", v.Artist),
				fmt.Sprintf("%s", v.Album),
				v.Timestamp,
				bigquery.NullInt64{Int64: v.Duration, Valid: true},
				fmt.Sprintf("%s", v.SpotifyID),
				fmt.Sprintf("%s", v.AlbumCover),
				v.CreatedAt,
				fmt.Sprintf("%s", v.Source),
				fmt.Sprintf("%s", v.YoutubeID),
				fmt.Sprintf("%s", v.YoutubeCategoryID),
				fmt.Sprintf("%s", v.SoundcloudID),
				fmt.Sprintf("%s", v.SoundcloudPermalink),
				fmt.Sprintf("%s", v.ShazamID),
				fmt.Sprintf("%s", v.ShazamPermalink),
			},
		})

		if len(vss) >= 10000 || i+1 == len(rows) { // upload in chunks
			err := uploader.Put(ctx, vss)
			if err != nil {
				if pmErr, ok := err.(bigquery.PutMultiError); ok {
					for _, rowInsertionError := range pmErr {
						log.Println(rowInsertionError.Errors)
					}
				}

				return fmt.Errorf("failed to insert data: %v", err)
			}
			vss = []*bigquery.ValuesSaver{}
		}
	}

	return nil
}

func setHashes(rows []row) []row {
	var newRows []row
	for _, v := range rows {
		v.Track = strings.Trim(v.Track, " ")
		v.TrackHash = fmt.Sprintf("%v", md5.Sum([]byte(strings.ToLower(v.Track))))
		v.ArtistHash = fmt.Sprintf("%v", md5.Sum([]byte(strings.ToLower(v.Artist))))
		newRows = append(newRows, v)
	}
	return newRows
}

func setNewDates(rows []row) []row {
	var newRows []row
	for _, v := range rows {
		if v.Timestamp.Unix() == 0 {
			v.Timestamp = time.Unix(1358816400, 0)
		}
		newRows = append(newRows, v)
	}
	return newRows
}

func setMissingCreatedAtTimes(rows []row) []row {
	var newRows []row
	for _, v := range rows {
		if v.CreatedAt.Unix() == 0 && v.Timestamp.Unix() != 0 {
			v.CreatedAt = v.Timestamp
		}
		newRows = append(newRows, v)
	}
	return newRows
}

func setOldRowSources(rows []row) []row {
	var newRows []row
	lastfmCutOff, _ := time.Parse(time.RFC3339, "2018-10-11T21:00:00Z")

	for _, v := range rows {
		if v.Source == "" {
			if v.SpotifyID == "" {
				if v.Timestamp.Before(lastfmCutOff) {
					v.Source = "lastfm"
				} else {
					v.Source = "unknown"
				}
			} else if len(v.SpotifyID) > 1 {
				v.Source = "spotify"
			} else {
				v.Source = "unknown"
			}
		}
		newRows = append(newRows, v)
	}
	return newRows
}

func skipDupeNowPlaying(rows []row) []row {
	var newRows []row
	var lastHash = ""

	for _, v := range rows {
		if v.Source == "now_playing" {
			if v.Hash() != lastHash {
				newRows = append(newRows, v)
			}
			lastHash = v.Hash()
		} else {
			newRows = append(newRows, v)
		}
	}
	return newRows
}

func coalesceMetadata(rows []row) []row {
	var newRows []row

	byTrack := make(map[string][]row)

	for _, v := range rows {
		byTrack[v.Hash()] = append(byTrack[v.Hash()], v)
	}

	for _, v := range byTrack {
		defaultRow := createDefaultRow(v)
		for _, e := range v {
			e.Artist = defaultRow.Artist
			e.Track = defaultRow.Track

			if e.Duration == 0 && defaultRow.Duration > 0 {
				e.Album = defaultRow.Album
			}
			if e.Album == "" && defaultRow.Album != "" {
				e.Album = defaultRow.Album
			}
			if e.AlbumCover == "" && defaultRow.AlbumCover != "" {
				e.AlbumCover = defaultRow.AlbumCover
			}
			if e.ShazamID == "" && defaultRow.ShazamID != "" {
				e.ShazamID = defaultRow.ShazamID
			}
			if e.ShazamPermalink == "" && defaultRow.ShazamPermalink != "" {
				e.ShazamPermalink = defaultRow.ShazamPermalink
			}
			if e.SoundcloudID == "" && defaultRow.SoundcloudID != "" {
				e.SoundcloudID = defaultRow.SoundcloudID
			}
			if e.SoundcloudPermalink == "" && defaultRow.SoundcloudPermalink != "" {
				e.SoundcloudPermalink = defaultRow.SoundcloudPermalink
			}
			if e.SpotifyID == "" && defaultRow.SpotifyID != "" {
				e.SpotifyID = defaultRow.SpotifyID
			}
			if e.YoutubeID == "" && defaultRow.YoutubeID != "" {
				e.YoutubeID = defaultRow.YoutubeID
			}
			if e.YoutubeCategoryID == "" && defaultRow.YoutubeID != "" {
				e.YoutubeCategoryID = defaultRow.YoutubeCategoryID
			}
			newRows = append(newRows, e)
		}
	}

	return newRows
}

func createDefaultRow(rows []row) row {
	var durationSum int64
	var durationCount int64
	counts := make(map[string]map[string]int)
	counts["Album"] = make(map[string]int)
	counts["AlbumCover"] = make(map[string]int)
	counts["Artist"] = make(map[string]int)
	counts["ShazamID"] = make(map[string]int)
	counts["ShazamPermalink"] = make(map[string]int)
	counts["SoundcloudID"] = make(map[string]int)
	counts["SoundcloudPermalink"] = make(map[string]int)
	counts["SpotifyID"] = make(map[string]int)
	counts["Track"] = make(map[string]int)
	counts["YoutubeID"] = make(map[string]int)
	counts["YoutubeCategoryID"] = make(map[string]int)
	for _, v := range rows {
		if v.Duration > 0 {
			durationSum += v.Duration
			durationCount++
		}
		counts["Album"][v.Album]++
		counts["AlbumCover"][v.AlbumCover]++
		counts["Artist"][v.Artist]++
		counts["ShazamID"][v.ShazamID]++
		counts["ShazamPermalink"][v.ShazamPermalink]++
		counts["SoundcloudID"][v.SoundcloudID]++
		counts["SoundcloudPermalink"][v.SoundcloudPermalink]++
		counts["SpotifyID"][v.SpotifyID]++
		counts["Track"][v.Track]++
		counts["YoutubeID"][v.YoutubeID]++
		counts["YoutubeCategoryID"][v.YoutubeCategoryID]++
	}

	var defaultRow row
	if durationCount > 0 {
		defaultRow.Duration = durationSum / durationCount
	}
	defaultRow.Album = returnMostCommon(counts["Album"])
	defaultRow.AlbumCover = returnMostCommon(counts["AlbumCover"])
	defaultRow.Artist = returnMostCommon(counts["Artist"])
	defaultRow.ShazamID = returnMostCommon(counts["ShazamID"])
	defaultRow.ShazamPermalink = returnMostCommon(counts["ShazamPermalink"])
	defaultRow.SoundcloudID = returnMostCommon(counts["SoundcloudID"])
	defaultRow.SoundcloudPermalink = returnMostCommon(counts["SoundcloudPermalink"])
	defaultRow.SpotifyID = returnMostCommon(counts["SpotifyID"])
	defaultRow.Track = returnMostCommon(counts["Track"])
	defaultRow.YoutubeCategoryID = returnMostCommon(counts["YoutubeCategoryID"])
	defaultRow.YoutubeID = returnMostCommon(counts["YoutubeID"])

	return defaultRow
}

func returnMostCommon(stringCounts map[string]int) string {
	var max int
	var chosen string
	for k, v := range stringCounts {
		if len(k) > 0 {
			if v > max {
				max = v
				chosen = k
			}
		}
	}
	return chosen
}
