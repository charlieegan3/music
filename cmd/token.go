package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/zmb3/spotify"
)

var auth spotify.Authenticator

// Token gives a URL to get a new token and listens for the callback from spotify
func Token() {
	auth = spotify.NewAuthenticator("http://localhost:8080", spotify.ScopeUserReadPrivate, spotify.ScopePlaylistReadPrivate)
	auth.SetAuthInfo(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"))

	fmt.Println(auth.AuthURL(os.Getenv("SPOTIFY_AUTH_STATE")))

	http.HandleFunc("/", redirectHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.Token(os.Getenv("SPOTIFY_AUTH_STATE"), r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusNotFound)
		return
	}
	fmt.Printf("Access:\n%v\n\nRefresh:\n%v\n", token.AccessToken, token.RefreshToken)
	os.Exit(0)
}
