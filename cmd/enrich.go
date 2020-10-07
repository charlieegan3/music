package cmd

import (
	"log"

	"github.com/charlieegan3/music/internal/pkg/enrich"
	"github.com/spf13/cobra"
)

var enrichCommand = &cobra.Command{
	Use:   "enrich",
	Short: "enrich raw play data and reupload",
	Run: func(cmd *cobra.Command, args []string) {
		err := retry(func() error {
			return enrich.Enrich(cfg)
		})
		if err != nil {
			log.Fatalf("enrich failed: %v", err)
		}
	},
}

func init() {
	rootCommand.AddCommand(enrichCommand)
}
