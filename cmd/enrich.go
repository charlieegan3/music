package cmd

import (
	"log"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	"github.com/charlieegan3/music/internal/pkg/enrich"
	"github.com/spf13/cobra"
)

var enrichCommand = &cobra.Command{
	Use:   "enrich",
	Short: "enrich raw play data and reupload",
	Run: func(cmd *cobra.Command, args []string) {
		b := backoff.NewExponentialBackOff()
		b.MaxElapsedTime = 3 * time.Minute
		err := backoff.Retry(func() error {
			return enrich.Enrich(cfg)
		}, b)
		if err != nil {
			log.Fatalf("enrich failed: %v", err)
		}
	},
}

func init() {
	rootCommand.AddCommand(enrichCommand)
}
