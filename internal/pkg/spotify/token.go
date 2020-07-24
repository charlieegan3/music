package spotify

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/zmb3/spotify"
)

// Token gives a URL to get a new token and listens for the callback from Spotify
func Token(clientID, clientSecret, authState string) {
	var auth spotify.Authenticator
	auth = spotify.NewAuthenticator("http://localhost:8080", spotify.ScopeUserReadPrivate, spotify.ScopePlaylistReadPrivate, spotify.ScopeUserReadRecentlyPlayed)
	auth.SetAuthInfo(clientID, clientSecret)

	fmt.Println(auth.AuthURL(authState))

	http.HandleFunc("/", newRedirectHandler(auth, authState))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func newRedirectHandler(auth spotify.Authenticator, authState string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.Token(authState, r)
		if err != nil {
			http.Error(w, "Couldn't get token", http.StatusNotFound)
			return
		}
		fmt.Printf("\nAccess:\n%v\n\nRefresh:\n%v\n", token.AccessToken, token.RefreshToken)
		os.Exit(0)
	}
}
