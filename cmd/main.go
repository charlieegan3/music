package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

func main() {
	var err error
	err = godotenv.Load(os.Getenv("ENV_PATH"))
	if err != nil {
		log.Fatalf("Error loading .env file %v", err)
		os.Exit(1)
	}

	if len(os.Args) != 2 {
		fmt.Println("pass only one argument")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "token":
		Token()
	case "download":
		Download()
	case "lastfm":
		err = LastFM()
	case "spotify":
		err = Spotify()
	case "summary":
		err = Summary()
	case "summary_recent":
		err = SummaryRecent()
	case "summary_months":
		err = SummaryMonths()
	case "summary_tracks":
		err = SummaryTracks()
	case "backup_plays_table":
		err = BackupPlaysTable()
	case "youtube":
		err = Youtube()
	case "soundcloud":
		err = Soundcloud()
	case "shazam":
		err = Shazam()
	case "uploader":
		err = Uploader()
	case "enrich":
		err = Enrich()
	default:
		err = errors.New("enter a valid command")
	}

	if err != nil {
		log.Fatalf("Failed due to error: %v", err)
		os.Exit(1)
	}
}

func buildClient() spotify.Client {
	token := &oauth2.Token{
		AccessToken:  os.Getenv("SPOTIFY_ACCESS_TOKEN"),
		TokenType:    "Bearer",
		RefreshToken: os.Getenv("SPOTIFY_REFRESH_TOKEN"),
		Expiry:       time.Now(),
	}

	auth := spotify.NewAuthenticator("http://localhost:8080", spotify.ScopeUserReadPrivate)
	auth.SetAuthInfo(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"))
	return auth.NewClient(token)
}
