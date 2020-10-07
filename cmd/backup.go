package cmd

import (
	"log"

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
		err := retry(func() error {
			return backup.Plays(cfg)
		})
		if err != nil {
			log.Fatalf("backup failed: %v", err)
		}
	},
}

func init() {
	rootCommand.AddCommand(backupCommand)
	backupCommand.AddCommand(backupPlaysCommand)
}
