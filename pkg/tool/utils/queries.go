package utils

import (
	"cloud.google.com/go/bigquery"
	"context"
	"fmt"
	"google.golang.org/api/iterator"
	"log"
	"time"
)

// MostRecentTimestamps returns the N most recent timestamps for a given source
func MostRecentTimestamps(
	ctx context.Context,
	client *bigquery.Client,
	projectID string,
	datasetName string,
	tableName string,
	source string,
	count int,
) ([]time.Time, error) {
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
			log.Println(err)
			return t, fmt.Errorf("Failed reading results for time: %v", err)
		}
		t = append(t, l.Timestamp)
	}

	return t, nil
}
