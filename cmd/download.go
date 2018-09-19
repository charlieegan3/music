package main

import (
	"fmt"
	"os"
	"time"

	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

// Download saves all playlists from Spotify
func Download() {
	token := &oauth2.Token{
		AccessToken:  os.Getenv("SPOTIFY_ACCESS_TOKEN"),
		TokenType:    "Bearer",
		RefreshToken: os.Getenv("SPOTIFY_REFRESH_TOKEN"),
		Expiry:       time.Now(),
	}

	auth := spotify.NewAuthenticator("http://localhost:8080", spotify.ScopeUserReadPrivate)
	auth.SetAuthInfo(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"))
	client := auth.NewClient(token)

	limit := 50
	offset := 0

	var playlists []spotify.SimplePlaylist

	for {
		playlistsPage, err := client.GetPlaylistsForUserOpt("charlieegan3", &spotify.Options{Limit: &limit, Offset: &offset})
		if err != nil {
			fmt.Printf("There was an error getting the playlists: %s", err)
			return
		}

		playlists = append(playlists, playlistsPage.Playlists...)
		fmt.Printf("Fetched %v playlists\n", len(playlists))

		if len(playlistsPage.Playlists) != limit {
			break
		}

		offset += limit
	}

	for _, v := range playlists {
		fmt.Printf("%v %v\n", v.Name, v.Tracks.Total)
	}
	fmt.Println(len(playlists))
}
