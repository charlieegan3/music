package cmd

import (
	"github.com/charlieegan3/music/pkg/tool/bq"
	"log"

	"github.com/charlieegan3/music/internal/pkg/shazam"
	"github.com/charlieegan3/music/internal/pkg/soundcloud"
	"github.com/charlieegan3/music/internal/pkg/spotify"
	"github.com/charlieegan3/music/internal/pkg/youtube"
	"github.com/spf13/cobra"
)

var syncCommand = &cobra.Command{
	Use:   "sync",
	Short: "sync data from various play sources",
}

var spotifySyncCommand = &cobra.Command{
	Use:   "spotify",
	Short: "fetch latest plays and saves any new ones",
	Run: func(cmd *cobra.Command, args []string) {
		operation := func() error {
			err := spotify.Sync(
				cfg.Spotify.AccessToken,
				cfg.Spotify.RefreshToken,
				cfg.Spotify.ClientID,
				cfg.Spotify.ClientSecret,
				cfg.Google.SvcAccountJSON,
				cfg.Google.Project,
				cfg.Google.Dataset,
				cfg.Google.Table,
				bq.JSONSchema,
			)
			if err != nil {
				log.Printf("sync attempt failed: %v", err)
			}
			return err
		}

		err := retry(operation)
		if err != nil {
			log.Fatalf("sync failed: %v", err)
		}
	},
}

var youtubeSyncCommand = &cobra.Command{
	Use:   "youtube",
	Short: "upload latest plays from Youtube",
	Run: func(cmd *cobra.Command, args []string) {
		operation := func() error {
			err := youtube.Sync(cfg)
			if err != nil {
				log.Printf("sync attempt failed: %v", err)
			}
			return err
		}
		err := retry(operation)
		if err != nil {
			log.Fatalf("sync failed: %v", err)
		}
	},
}

var shazamSyncCommand = &cobra.Command{
	Use:   "shazam",
	Short: "upload latest plays from shazam",
	Run: func(cmd *cobra.Command, args []string) {
		operation := func() error {
			err := shazam.Sync(cfg)
			if err != nil {
				log.Printf("sync attempt failed: %v", err)
			}
			return err
		}
		err := retry(operation)
		if err != nil {
			log.Fatalf("sync failed: %v", err)
		}
	},
}

var soundcloudSyncCommand = &cobra.Command{
	Use:   "soundcloud",
	Short: "upload latest plays from Soundcloud",
	Run: func(cmd *cobra.Command, args []string) {
		operation := func() error {
			err := soundcloud.Sync(cfg)
			if err != nil {
				log.Printf("sync attempt failed: %v", err)
			}
			return err
		}
		err := retry(operation)
		if err != nil {
			log.Fatalf("sync failed: %v", err)
		}
	},
}

func init() {
	syncCommand.AddCommand(spotifySyncCommand)
	syncCommand.AddCommand(youtubeSyncCommand)
	syncCommand.AddCommand(shazamSyncCommand)
	syncCommand.AddCommand(soundcloudSyncCommand)
	rootCommand.AddCommand(syncCommand)
}
