package cmd

import (
	"log"

	"github.com/charlieegan3/music/internal/pkg/summary"
	"github.com/spf13/cobra"
)

var summarizeCommand = &cobra.Command{
	Use:   "summarize",
	Short: "commands related to generating summaries of music data",
}

var summaryOverviewCommand = &cobra.Command{
	Use:   "overview",
	Short: "generate and save stats for homepage",
	Run: func(cmd *cobra.Command, args []string) {
		err := retry(func() error {
			return summary.Overview(cfg)
		})

		if err != nil {
			log.Fatalf("summary overview failed: %v", err)
		}
	},
}

var summaryRecentCommand = &cobra.Command{
	Use:   "recent",
	Short: "save data about most recent plays",
	Run: func(cmd *cobra.Command, args []string) {
		err := retry(func() error {
			return summary.Recent(cfg)
		})

		if err != nil {
			log.Fatalf("summary recent failed: %v", err)
		}
	},
}

var summaryMonthsCommand = &cobra.Command{
	Use:   "months",
	Short: "generate top lists for each month",
	Run: func(cmd *cobra.Command, args []string) {
		err := retry(func() error {
			return summary.Months(cfg)
		})

		if err != nil {
			log.Fatalf("summary months failed: %v", err)
		}
	},
}

var summaryTracksCommand = &cobra.Command{
	Use:   "tracks",
	Short: "generate and save the tracks summary for artists page",
	Run: func(cmd *cobra.Command, args []string) {
		err := retry(func() error {
			return summary.Tracks(cfg)
		})

		if err != nil {
			log.Fatalf("summary tracks failed: %v", err)
		}
	},
}

func init() {
	rootCommand.AddCommand(summarizeCommand)
	summarizeCommand.AddCommand(summaryOverviewCommand)
	summarizeCommand.AddCommand(summaryRecentCommand)
	summarizeCommand.AddCommand(summaryMonthsCommand)
	summarizeCommand.AddCommand(summaryTracksCommand)
}
