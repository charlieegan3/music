package cmd

import (
	"log"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	"github.com/charlieegan3/music/internal/pkg/backup"
	"github.com/spf13/cobra"
)

var backupCommand = &cobra.Command{
	Use:   "backup",
	Short: "commands relating to backing up of play data",
}

var backupPlaysCommand = &cobra.Command{
	Use:   "plays",
	Short: "save data in bigquery table to GCS",
	Run: func(cmd *cobra.Command, args []string) {
		b := backoff.NewExponentialBackOff()
		b.MaxElapsedTime = 3 * time.Minute
		err := backoff.Retry(func() error {
			return backup.Plays(cfg)
		}, b)
		if err != nil {
			log.Fatalf("backup failed: %v", err)
		}
	},
}

func init() {
	rootCommand.AddCommand(backupCommand)
	backupCommand.AddCommand(backupPlaysCommand)
}
