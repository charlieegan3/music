package main

import (
	"context"
	"log"
	"os"

	"github.com/spf13/viper"

	"github.com/charlieegan3/music/pkg/tool"
)

func main() {
	viper.SetConfigName("config-tools")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Fatal error config file: %s \n", err)
	}

	toolCfg, ok := viper.Get("tools.music").(map[string]interface{})
	if !ok {
		log.Fatalf("failed to read tools config in map[string]interface{} format")
	}
	t := &tool.Music{}
	t.SetConfig(toolCfg)

	j, err := t.Jobs()
	if err != nil {
		log.Fatalf("failed to get jobs: %s", err)
	}

	switch os.Args[1] {
	case "lastfm":
		err = j[0].Run(context.Background())
	case "spotify":
		err = j[1].Run(context.Background())
	default:
		log.Fatalf("unknown job: %s", os.Args[1])
	}

	if err != nil {
		log.Fatalf("failed to run job: %s", err)
	}
}
