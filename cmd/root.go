package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/charlieegan3/music/internal/pkg/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var rootCommand = &cobra.Command{
	Use:   "music",
	Short: "Operations to sync plays and generate state for music.charlieegan3.com",
	// Load and validate config before all commands
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// load application config
		file, err := os.Open("config.yaml")
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		// parse config for use in command configs
		b, err := ioutil.ReadAll(file)
		err = yaml.Unmarshal(b, &cfg)
		if err != nil {
			fmt.Println("error:", err)
		}

		// defer to clean up the set env vars
		defer cfg.ValidateAndInit()
	},
}

var cfg config.Config

// Execute is the main entrypoint and manages all child commands
func Execute() {
	if err := rootCommand.Execute(); err != nil {
		log.Fatalf("%v", err)
	}
}
