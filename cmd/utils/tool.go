package main

import (
	"context"
	"log"

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

	err = j[0].Run(context.Background())
	if err != nil {
		log.Fatalf("failed to run job: %s", err)
	}
}
