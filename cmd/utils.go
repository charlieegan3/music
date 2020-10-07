package cmd

import (
	"time"

	backoff "github.com/cenkalti/backoff/v4"
)

func retry(operation backoff.Operation) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 3 * time.Minute

	return backoff.Retry(operation, b)
}
