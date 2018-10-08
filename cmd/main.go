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
	if os.Args[1] == "token" {
		Token()
		os.Exit(0)
	}
	if os.Args[1] == "download" {
		Download()
		os.Exit(0)
	}
	if os.Args[1] == "lastfm" {
		LastFM()
		os.Exit(0)
	}
	if os.Args[1] == "latest" {
		Latest()
		os.Exit(0)
	}

	fmt.Println("please pass either token or download")
	os.Exit(1)
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
