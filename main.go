// This example demonstrates how to authenticate with Spotify.
// In order to run this example yourself, you'll need to:
//
//  1. Register an application at: https://developer.spotify.com/my-applications/
//       - Use "http://localhost:8080/callback" as the redirect URI
//  2. Set the SPOTIFY_ID environment variable to the client ID you got in step 1.
//  3. Set the SPOTIFY_SECRET environment variable to the client secret from step 1.
//  4. Override the CALLBACK_URL environment variable if not using localhost callback
package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

// redirectURI is the OAuth redirect URI for the application.
// You must register an application at Spotify's developer portal
// and enter this value.
const defaultRedirectURI = "http://localhost:8080/callback"

var (
	auth  spotify.Authenticator
	ch    = make(chan *spotify.Client)
	state = "abc123"
)

func main() {

	if os.Getenv("CALLBACK_URL") != "" {
		auth = spotify.NewAuthenticator(os.Getenv("CALLBACK_URL"), spotify.ScopeUserReadPrivate)
	} else {
		auth = spotify.NewAuthenticator(defaultRedirectURI, spotify.ScopeUserReadPrivate)
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/callback", authHandler)
	http.HandleFunc("/search", searchHandler)

	go http.ListenAndServe(":8080", nil)

	log.Println("listening on :8080")

	<-ch
}

type pageData struct {
	Title   string
	Warning string
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeTemplate(w, http.StatusNotFound, "not_found.html", pageData{
			Title: "404 - not found",
		})
		return
	}

	writeTemplate(w, http.StatusOK, "index.html", pageData{
		Title: "Track lister",
	})
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	oauthToken, err := r.Cookie("sp_token")
	if err != nil && err != http.ErrNoCookie {
		serverErrorHandler(w, err)
		return
	}

	if oauthToken == nil || oauthToken.Value == "" {
		url := auth.AuthURL(state)
		http.Redirect(w, r, url, http.StatusFound)
		return
	}

	tok := &oauth2.Token{
		AccessToken: oauthToken.Value,
		TokenType:   "bearer",
	}

	type searchData struct {
		pageData
		PlaylistID string
		Tracks     []spotify.FullTrack
	}

	playlistID := r.FormValue("playlist")
	playlistID = strings.TrimSpace(playlistID)
	pd := searchData{
		pageData:   pageData{Title: "Search"},
		PlaylistID: playlistID,
	}

	playlist := spotify.ID("")
	if playlistID != "" {
		parts := strings.Split(playlistID, ":")
		if len(parts) != 3 {
			pd.Warning = "That did not looks like a valid search term"
		} else {
			playlist = spotify.ID(parts[2])
		}
	}

	if playlist == "" {
		writeTemplate(w, 401, "search.html", pd)
		return
	}

	log.Printf("Received search request for playlist id %s", playlistID)

	client := auth.NewClient(tok)
	pl, err := client.GetPlaylist(playlist)
	if err != nil {
		pd.Warning = err.Error()
	} else {
		log.Printf("Found %d tunes\n", len(pl.Tracks.Tracks))

		for _, track := range pl.Tracks.Tracks {
			pd.Tracks = append(pd.Tracks, track.Track)
		}
	}

	writeTemplate(w, 200, "search.html", pd)
}

func authHandler(w http.ResponseWriter, r *http.Request) {
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

	http.Redirect(w, r, "/search", http.StatusFound)
}

func writeTemplate(w http.ResponseWriter, statusCode int, tmplFile string, data interface{}) {
	t, err := template.ParseGlob("./templates/*")
	if err != nil {
		serverErrorHandler(w, err)
		return
	}

	w.WriteHeader(statusCode)
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	err = t.ExecuteTemplate(w, tmplFile, data)
	if err != nil {
		serverErrorHandler(w, err)
		return
	}
}

func serverErrorHandler(w http.ResponseWriter, err error) {
	t, tErr := template.ParseGlob("./templates/*")
	if tErr != nil {
		fmt.Fprintf(w, "<p>500 Server error</p><p>%s</p>", err)
		log.Printf("server error: %s\n", err)
		return
	}

	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(500)
	sErr := t.ExecuteTemplate(w, "error.html", pageData{
		Title:   "Server error",
		Warning: err.Error(),
	})
	if sErr != nil {
		fmt.Fprintf(w, "<p>500 Server error</p><p>%s</p>", err)
		log.Printf("server error: %s\n", err)
		return
	}
}
