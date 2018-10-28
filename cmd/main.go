package main

import (
	"fmt"
	"os"
	"time"

	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

func main() {
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
		LastFM()
	case "latest":
		Latest()
	case "summary":
		Summary()
	case "summary_recent":
		SummaryRecent()
	case "backup_plays_table":
		BackupPlaysTable()
	default:
		fmt.Println("enter a valid command")
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
