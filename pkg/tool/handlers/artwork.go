package handlers

import (
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"net/http"
)

func BuildArtworkHandler(bucketName string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=604800")
		w.Header().Set("Content-Type", "image/jpeg")

		artist, ok := mux.Vars(r)["artist"]
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("artist name is required"))
			return
		}

		album, ok := mux.Vars(r)["album"]
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("album name is required"))
			return
		}

		sourcePath := fmt.Sprintf(
			"https://storage.googleapis.com/%s/%s/%s.jpg",
			bucketName,
			artist,
			album,
		)

		resp, err := http.Get(sourcePath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("ETag", resp.Header.Get("ETag"))

		_, err = io.Copy(w, resp.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}
}
