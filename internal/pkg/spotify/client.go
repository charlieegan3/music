package spotify

import (
	"time"

	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

// creates a token to interact with the Spotify API
func buildClient(accessToken, refreshToken, clientID, clientSecret string) spotify.Client {
	token := &oauth2.Token{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		RefreshToken: refreshToken,
		Expiry:       time.Now(),
	}

	auth := spotify.NewAuthenticator("http://localhost:8080", spotify.ScopeUserReadPrivate)
	auth.SetAuthInfo(clientID, clientSecret)
	return auth.NewClient(token)
}
