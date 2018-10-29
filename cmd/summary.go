package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type resultPlayFromPeriod struct {
	Track   string
	Artist  string
	Album   string
	Artwork string
	// Duration int
	Spotify string

	Count int
}

type resultArtistFromPeriod struct {
	Artist string
	Count  int
}

type resultCountForMonth struct {
	Month  string
	Count  int
	Pretty string
}

// Summary gets a list of recently played tracks
func Summary() {
	// Gather env config values
	projectID := os.Getenv("GOOGLE_PROJECT")
	datasetName := os.Getenv("GOOGLE_DATASET")
	tableName := os.Getenv("GOOGLE_TABLE")
	accountJSON := os.Getenv("GOOGLE_JSON")
	bucketName := os.Getenv("GOOGLE_SUMMARY_BUCKET")
	objectName := "stats.json"

	// get the credentials from json
	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(ctx, []byte(accountJSON), bigquery.Scope, storage.ScopeReadWrite)
	if err != nil {
		log.Fatalf("Creds parse failed: %v", err)
		os.Exit(1)
	}

	// create a big query client to query for the music stats
	bigqueryClient, err := bigquery.NewClient(ctx, projectID, option.WithCredentials(creds))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
		os.Exit(1)
	}

	// fetch and format data
	output := struct {
		PlaysByMonth []resultCountForMonth

		PlaysYear  []resultPlayFromPeriod
		PlaysMonth []resultPlayFromPeriod
		PlaysWeek  []resultPlayFromPeriod

		ArtistsYear  []resultArtistFromPeriod
		ArtistsMonth []resultArtistFromPeriod
		ArtistsWeek  []resultArtistFromPeriod

		LastUpdated string
	}{
		countsForLastYear(ctx, bigqueryClient, projectID, datasetName, tableName),
		playsFromLastNDays(ctx, bigqueryClient, projectID, datasetName, tableName, 365),
		playsFromLastNDays(ctx, bigqueryClient, projectID, datasetName, tableName, 30),
		playsFromLastNDays(ctx, bigqueryClient, projectID, datasetName, tableName, 7),
		artistsForLastNDays(ctx, bigqueryClient, projectID, datasetName, tableName, 365),
		artistsForLastNDays(ctx, bigqueryClient, projectID, datasetName, tableName, 30),
		artistsForLastNDays(ctx, bigqueryClient, projectID, datasetName, tableName, 7),

		time.Now().UTC().Format(time.RFC3339),
	}

	// format data as json
	bytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	storageClient, err := storage.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalf("Client create Failed: %v", err)
		os.Exit(1)
	}

	bkt := storageClient.Bucket(bucketName)
	obj := bkt.Object(objectName)

	w := obj.NewWriter(ctx)
	w.ContentType = "application/json"
	w.ObjectAttrs.CacheControl = "max-age=3600"

	if _, err := fmt.Fprintf(w, string(bytes)); err != nil {
		log.Fatalf("Write Failed: %v", err)
		os.Exit(1)
	}
	if err := w.Close(); err != nil {
		log.Fatalf("Close Failed: %v", err)
		os.Exit(1)
	}
}

func countsForLastYear(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string) []resultCountForMonth {
	queryString :=
		"SELECT FORMAT_DATE(\"%Y-%m\", DATE(timestamp)) as month, FORMAT_DATE(\"%B %Y\", DATE(timestamp)) as pretty, count(track) as count\n" +
			"FROM " + fmt.Sprintf("`%s.%s.%s`\n", projectID, datasetName, tableName) +
			`GROUP BY month, pretty
			ORDER BY month DESC
			LIMIT 12`

	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var results []resultCountForMonth
	var result resultCountForMonth
	for {
		err := it.Next(&result)
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		results = append(results, result)
	}
	return results
}

func playsFromLastNDays(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string, days int) []resultPlayFromPeriod {
	queryString := fmt.Sprintf(
		`SELECT
		  track,
		  artist,
		  album,
		  count(track) as count,
		  STRING_AGG(album_cover, "" ORDER BY LENGTH(album_cover) DESC LIMIT 1) as artwork,
		  ANY_VALUE(duration) as duration,
		  ANY_VALUE(spotify_id) as spotify
		FROM `+"`"+"%s.%s.%s"+"`"+
			`WHERE timestamp BETWEEN TIMESTAMP_ADD(CURRENT_TIMESTAMP(), INTERVAL -%d DAY) AND CURRENT_TIMESTAMP()
		GROUP BY track, artist, album
		ORDER BY count DESC
		LIMIT 10`,
		projectID,
		datasetName,
		tableName,
		days,
	)

	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var results []resultPlayFromPeriod
	var result resultPlayFromPeriod
	for {
		err := it.Next(&result)
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		results = append(results, result)
	}
	return results
}

func artistsForLastNDays(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string, days int) []resultArtistFromPeriod {
	queryString := fmt.Sprintf(
		`SELECT
		  artist,
		  count(track) as count
		FROM `+"`"+"%s.%s.%s"+"`"+
			`WHERE timestamp BETWEEN TIMESTAMP_ADD(CURRENT_TIMESTAMP(), INTERVAL -%d DAY) AND CURRENT_TIMESTAMP()
		GROUP BY artist
		ORDER BY count DESC
		LIMIT 10`,
		projectID,
		datasetName,
		tableName,
		days,
	)

	q := client.Query(queryString)
	it, err := q.Read(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var results []resultArtistFromPeriod
	var result resultArtistFromPeriod
	for {
		err := it.Next(&result)
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		results = append(results, result)
	}
	return results
}
