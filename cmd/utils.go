package main

import (
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
)

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
