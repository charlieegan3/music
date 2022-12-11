package handlers

import (
	"net/http"
)

func BuildIndexHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("index"))
	}
}
