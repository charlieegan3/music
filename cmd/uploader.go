package cmd

import (
	"log"

	"github.com/charlieegan3/music/internal/pkg/uploader"
	"github.com/spf13/cobra"
)

var uploaderServeCommand = &cobra.Command{
	Use:   "uploader",
	Short: "start the uploader server process",
	Run: func(cmd *cobra.Command, args []string) {
		err := uploader.Serve(cfg)
		if err != nil {
			log.Fatalf("serve failed: %v", err)
		}
	},
}

func init() {
	rootCommand.AddCommand(uploaderServeCommand)
}
