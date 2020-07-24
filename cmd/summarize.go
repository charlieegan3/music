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
		err := summary.Overview(cfg)
		if err != nil {
			log.Fatalf("summary failed: %v", err)
		}
	},
}

var summaryRecentCommand = &cobra.Command{
	Use:   "recent",
	Short: "save data about most recent plays",
	Run: func(cmd *cobra.Command, args []string) {
		err := summary.Recent(cfg)
		if err != nil {
			log.Fatalf("summary failed: %v", err)
		}
	},
}

var summaryMonthsCommand = &cobra.Command{
	Use:   "months",
	Short: "generate top lists for each month",
	Run: func(cmd *cobra.Command, args []string) {
		err := summary.Months(cfg)
		if err != nil {
			log.Fatalf("summary failed: %v", err)
		}
	},
}

var summaryTracksCommand = &cobra.Command{
	Use:   "tracks",
	Short: "generate and save the tracks summary for artists page",
	Run: func(cmd *cobra.Command, args []string) {
		err := summary.Tracks(cfg)
		if err != nil {
			log.Fatalf("summary failed: %v", err)
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
