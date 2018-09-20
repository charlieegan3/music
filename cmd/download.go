package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

// Download saves all playlists from Spotify
func Download() {
	client := buildClient()
	playlists := fetchAllPLaylists(client)

	//tracks, _ := client.GetPlaylistTracks(v.ID)
	//fmt.Printf("%+v\n", tracks.Tracks[0].Track.SimpleTrack.Name)
	//fmt.Printf("%+v\n", tracks.Tracks[0].Track.SimpleTrack.Artists)
	//fmt.Printf("%+v\n", tracks.Tracks[0].Track.Album.Name)
	//os.Exit(0)

	for _, p := range playlists {
		if strings.Contains(p.Name, "Liked") {
			fmt.Printf("Skipping %v\n", p.Name)
			continue
		}

		path := createPlaylistFolder(p.Name)
		name := slug.Make(strings.Replace(p.Name, path, "", 1))

		fullPlaylist, err := client.GetPlaylist(p.ID)
		if err != nil {
			fmt.Printf("Failed to download complete playlist %v\n", p.Name)
			continue
		}
		json, err := json.MarshalIndent(fullPlaylist, "", "  ")
		if err != nil {
			fmt.Printf("Failed to marshal JSON for playlist %v\n", p.Name)
			continue
		}
		file, err := os.Create(fmt.Sprintf("spotify/%v/%v.json", path, name))
		if err != nil {
			log.Fatal("Failed to create file", err)
		}
		defer file.Close()
		fmt.Fprintf(file, string(json))
		fmt.Printf("Saved %v\n", name)
		os.Exit(0)
	}
	fmt.Println(len(playlists))
}

func createPlaylistFolder(name string) string {
	words := strings.Split(name, " ")
	if len(words) < 2 {
		fmt.Printf("Incompatible playlist name %v, skipping", name)
		return ""
	}
	os.MkdirAll(fmt.Sprintf("spotify/%v", words[0]), os.ModePerm)
	return words[0]
}

func fetchAllPLaylists(client spotify.Client) []spotify.SimplePlaylist {
	var playlists []spotify.SimplePlaylist

	count := 0
	limit := 50
	offset := 0
	for {
		if count > 15 {
			break
		}

		playlistsPage, err := client.GetPlaylistsForUserOpt("charlieegan3", &spotify.Options{Limit: &limit, Offset: &offset})
		if err != nil {
			fmt.Printf("There was an error getting the playlists: %s", err)
			count++
			continue
		}

		playlists = append(playlists, playlistsPage.Playlists...)
		fmt.Printf("Fetched %v playlists\n", len(playlists))

		if len(playlistsPage.Playlists) != limit {
			break
		}

		offset += limit
		count++
	}

	return playlists
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
