package cmd

import (
	"log"

	"github.com/charlieegan3/music/internal/pkg/monitoring"
	"github.com/spf13/cobra"
)

var monitoringCommand = &cobra.Command{
	Use:   "monitoring",
	Short: "commands to test the system is working",
}

var monitoringLatestProbeCommand = &cobra.Command{
	Use:   "latest",
	Short: "fails is last play is over 1 day ago",
	Run: func(cmd *cobra.Command, args []string) {
		err := retry(func() error {
			return monitoring.LatestProbe(cfg)
		})
		if err != nil {
			log.Fatalf("latest probe failed: %v", err)
		}
	},
}

func init() {
	monitoringCommand.AddCommand(monitoringLatestProbeCommand)
	rootCommand.AddCommand(monitoringCommand)
}
