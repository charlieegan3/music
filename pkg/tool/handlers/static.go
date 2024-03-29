package handlers

import (
	"embed"
	"net/http"
	"net/url"
)

//go:embed static/*
var staticContent embed.FS

func BuildStaticHandler() (handler func(http.ResponseWriter, *http.Request)) {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")

		rootedReq := http.Request{
			URL: &url.URL{
				Path: "./static" + req.URL.Path,
			},
		}
		http.FileServer(http.FS(staticContent)).ServeHTTP(w, &rootedReq)
	}
}
