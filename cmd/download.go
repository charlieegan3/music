package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/gosimple/slug"
	"github.com/zmb3/spotify"
)

// Download saves all playlists from Spotify
func Download() {
	client := buildClient()
	playlists := fetchAllPLaylists(client)

	for _, p := range playlists {
		if strings.Contains(p.Name, "Liked") {
			fmt.Printf("Skipping %v\n", p.Name)
			continue
		}

		path := createPlaylistFolder(p.Name)
		name := slug.Make(strings.Replace(p.Name, path, "", 1))
		filename := fmt.Sprintf("spotify/%v/%v.json", path, name)

		if _, err := os.Stat(filename); os.IsNotExist(err) {
			fmt.Printf("New playlist %v\n", name)
		} else {
			jsonFile, err := os.Open(filename)
			if err != nil {
				fmt.Println(err)
				continue
			}
			defer jsonFile.Close()

			bytes, err := ioutil.ReadAll(jsonFile)
			if err != nil {
				fmt.Println(err)
				continue
			}

			var playlist spotify.FullPlaylist
			json.Unmarshal(bytes, &playlist)

			if playlist.SnapshotID == p.SnapshotID {
				continue
			}
		}

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
		file, err := os.Create(filename)
		if err != nil {
			log.Fatal("Failed to create file", err)
		}
		defer file.Close()
		fmt.Fprintf(file, string(json))
		fmt.Printf("Saved %v\n", filename)
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
