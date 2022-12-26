package handlers

import (
	"net/http"

	"github.com/foolin/goview"
)

func BuildMenuHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := gv.Render(
			w,
			http.StatusOK,
			"menu",
			goview.M{},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}
