package cmd

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/charlieegan3/music/internal/pkg/lastfm"
	"github.com/charlieegan3/music/internal/pkg/spotify"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

var servicesCommand = &cobra.Command{
	Use:   "services",
	Short: "commands relating to service specific functions",
}

var spotifyCommand = &cobra.Command{
	Use:   "spotify",
	Short: "commands relating to spotify access and data",
}

var spotifyTokenCommand = &cobra.Command{
	Use:   "token",
	Short: "command to manually get new Spotify tokens for config",
	Run: func(cmd *cobra.Command, args []string) {
		spotify.Token(cfg.Spotify.ClientID, cfg.Spotify.ClientSecret, cfg.Spotify.AuthState)
	},
}
var spotifySavePlaylistsCommand = &cobra.Command{
	Use:   "save-playlists",
	Short: "command to save all playlists from Spotify",
	Run: func(cmd *cobra.Command, args []string) {
		err := spotify.SavePlaylists(
			cfg.Spotify.AccessToken,
			cfg.Spotify.RefreshToken,
			cfg.Spotify.ClientID,
			cfg.Spotify.ClientSecret,
		)
		if err != nil {
			log.Fatalf("sync failed: %v", err)
		}
	},
}

var lastFMCommand = &cobra.Command{
	Use:   "lastfm",
	Short: "commands relating to lastfm data",
}

var lastFMImportCommand = &cobra.Command{
	Use:       "import",
	Short:     "import data from lastfm dump between two timestamps",
	ValidArgs: []string{"start", "end"},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("expected two args, got %v", len(args))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var result *multierror.Error

		startSeconds, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to parse start time"))
		}
		endSeconds, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to parse end time"))
		}

		if result.ErrorOrNil() != nil {
			log.Fatalf("failed, invalid args: %v", result)
		}

		startTime := time.Unix(startSeconds, 0)
		endTime := time.Unix(endSeconds, 0)

		dryRun := true

		lastfm.Import(
			startTime,
			endTime,
			cfg.Google.Project,
			cfg.Google.Dataset,
			cfg.Google.Table,
			dryRun,
		)
	},
}

func init() {
	spotifyCommand.AddCommand(spotifyTokenCommand)
	spotifyCommand.AddCommand(spotifySyncCommand)
	spotifyCommand.AddCommand(spotifySavePlaylistsCommand)
	servicesCommand.AddCommand(spotifyCommand)

	lastFMImportCommand.PersistentFlags().BoolVarP(
		&lastfm.EnableUpload,
		"upload",
		"",
		false,
		"Uploads the imported data",
	)
	lastFMCommand.AddCommand(lastFMImportCommand)
	servicesCommand.AddCommand(lastFMCommand)

	rootCommand.AddCommand(servicesCommand)
}
