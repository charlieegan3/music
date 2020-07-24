package bigquery

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/bigquery"
)

// SavePlay uploads a single track to bigquery
func SavePlay(ctx context.Context,
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
