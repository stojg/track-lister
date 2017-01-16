// This example demonstrates how to authenticate with Spotify.
// In order to run this example yourself, you'll need to:
//
//  1. Register an application at: https://developer.spotify.com/my-applications/
//       - Use "http://localhost:8080/callback" as the redirect URI
//  2. Set the SPOTIFY_ID environment variable to the client ID you got in step 1.
//  3. Set the SPOTIFY_SECRET environment variable to the client secret from step 1.
package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
	"strings"
)

// redirectURI is the OAuth redirect URI for the application.
// You must register an application at Spotify's developer portal
// and enter this value.
const redirectURI = "http://localhost:8080/callback"

var (
	auth  = spotify.NewAuthenticator(redirectURI, spotify.ScopeUserReadPrivate)
	ch    = make(chan *spotify.Client)
	state = "abc123"
)

func main() {
	// first start an HTTP server
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html; charset=utf-8")

		sp_token, err := r.Cookie("sp_token")

		if err != nil {
			fmt.Println(err)
		}

		if sp_token == nil || sp_token.Value == "" {
			url := auth.AuthURL(state)
			http.Redirect(w, r, url, http.StatusFound)
			return
		}

		tok := &oauth2.Token{
			AccessToken: sp_token.Value,
			TokenType:   "bearer",
		}

		user := ""
		playlist := spotify.ID("")
		playlistID := r.FormValue("playlist")
		if playlistID != "" {
			parts := strings.Split(playlistID, ":")
			user = parts[2]
			playlist = spotify.ID(parts[4])
		}

		fmt.Fprintf(w, "<form><input name=\"playlist\" value=\"%s\" size=%d /><input type=\"submit\"></form>", playlistID, len(playlistID))
		if user == "" || playlist == "" {
			return
		}

		client := auth.NewClient(tok)
		pl, err := client.GetPlaylist(user, playlist)
		if err != nil {
			log.Fatal(err)
		}

		for _, track := range pl.Tracks.Tracks {
			artist := track.Track.Artists[0].Name
			trackName := track.Track.Name
			view := fmt.Sprintf("%s - %s", artist, trackName)
			query := strings.Replace(view, " ", "+", -1)
			fmt.Fprintf(w, "%s <a href=\"https://www.beatport.com/search/tracks?q=%s\">beatport search</a><br>", view, query)
		}

	})
	go http.ListenAndServe(":8080", nil)

	<-ch
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}

	// use the client to make calls that require authorization

	fmt.Printf("%s", tok.Expiry)
	cookie := http.Cookie{Name: "sp_token", Value: tok.AccessToken, Expires: tok.Expiry}
	http.SetCookie(w, &cookie)

	http.Redirect(w, r, "/", http.StatusFound)
}
