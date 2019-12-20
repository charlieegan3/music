package main

import (
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
)

func savePlay(ctx context.Context,
	schema bigquery.Schema,
	uploader bigquery.Uploader,
	track, artists, album, timestamp string,
	duration int64,
	spotifyID, artwork, source, youtubeID, youtubeCategoryID, soundcloudID, soundcloudPermalink, shazamID, shazamPermalink string) error {

	var vss []*bigquery.ValuesSaver
	vss = append(vss, &bigquery.ValuesSaver{
		Schema:   schema,
		InsertID: fmt.Sprintf("%v", time.Now().Unix()),
		Row: []bigquery.Value{
			track,
			artists,
			album,
			timestamp,
			bigquery.NullInt64{Int64: duration, Valid: true},
			spotifyID,
			artwork,
			time.Now().Unix(),
			source,
			youtubeID,
			youtubeCategoryID,
			soundcloudID,
			soundcloudPermalink,
			shazamID,
			shazamPermalink,
		},
	})

	// upload the items
	err := uploader.Put(ctx, vss)
	if err != nil {
		if pmErr, ok := err.(bigquery.PutMultiError); ok {
			for _, rowInsertionError := range pmErr {
				log.Println(rowInsertionError.Errors)
			}
		}

		return fmt.Errorf("Failed to insert row: %v", err)
	}

	return nil
}

func mostRecentTimestamp(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string, source string) (time.Time, error) {
	var t time.Time
	queryString := fmt.Sprintf(
		"SELECT timestamp FROM `%s.%s.%s` WHERE source = \"%s\" ORDER BY timestamp DESC LIMIT 1",
		projectID,
		datasetName,
		tableName,
		source,
	)
	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		return t, fmt.Errorf("Failed query for recent timestamp: %v", err)
	}
	var l struct {
		Timestamp time.Time
	}
	for {
		err := it.Next(&l)
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			return t, fmt.Errorf("Failed reading results for time: %v", err)
		}
		break
	}

	return l.Timestamp, nil
}

func nMostRecentTimestamps(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string, source string, count int) ([]time.Time, error) {
	var t []time.Time
	queryString := fmt.Sprintf(
		"SELECT timestamp FROM `%s.%s.%s` WHERE source = \"%s\" ORDER BY timestamp DESC LIMIT %d",
		projectID,
		datasetName,
		tableName,
		source,
		count,
	)
	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		return t, fmt.Errorf("Failed query for recent timestamps: %v", err)
	}
	var l struct {
		Timestamp time.Time
	}
	for {
		err := it.Next(&l)
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			return t, fmt.Errorf("Failed reading results for time: %v", err)
		}
		t = append(t, l.Timestamp)
	}

	return t, nil
}
