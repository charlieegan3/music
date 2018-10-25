package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/bigquery"
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
	Month string
	Count int
}

// Summary gets a list of recently played tracks
func Summary() {
	// Creates a bq client.
	ctx := context.Background()
	projectID := os.Getenv("GOOGLE_PROJECT")
	datasetName := os.Getenv("GOOGLE_DATASET")
	tableName := os.Getenv("GOOGLE_TABLE")
	accountJSON := os.Getenv("GOOGLE_JSON")

	creds, err := google.CredentialsFromJSON(ctx, []byte(accountJSON), bigquery.Scope)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	bigqueryClient, err := bigquery.NewClient(ctx, projectID, option.WithCredentials(creds))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
		os.Exit(1)
	}

	output := struct {
		PlaysYear  []resultPlayFromPeriod
		PlaysMonth []resultPlayFromPeriod
		PlaysWeek  []resultPlayFromPeriod

		ArtistsYear  []resultArtistFromPeriod
		ArtistsMonth []resultArtistFromPeriod
		ArtistsWeek  []resultArtistFromPeriod

		PlaysByMonth []resultCountForMonth
	}{
		playsFromLastNDays(ctx, bigqueryClient, projectID, datasetName, tableName, 365),
		playsFromLastNDays(ctx, bigqueryClient, projectID, datasetName, tableName, 30),
		playsFromLastNDays(ctx, bigqueryClient, projectID, datasetName, tableName, 7),
		artistsForLastNDays(ctx, bigqueryClient, projectID, datasetName, tableName, 365),
		artistsForLastNDays(ctx, bigqueryClient, projectID, datasetName, tableName, 30),
		artistsForLastNDays(ctx, bigqueryClient, projectID, datasetName, tableName, 7),
		countsForLastYear(ctx, bigqueryClient, projectID, datasetName, tableName),
	}

	bytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(string(bytes))
}

func countsForLastYear(ctx context.Context, client *bigquery.Client, projectID string, datasetName string, tableName string) []resultCountForMonth {
	queryString :=
		"SELECT FORMAT_DATE(\"%Y-%m\", DATE(timestamp)) as month, count(track) as count\n" +
			"FROM " + fmt.Sprintf("`%s.%s.%s`\n", projectID, datasetName, tableName) +
			`GROUP BY month
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
