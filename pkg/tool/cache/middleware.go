package cache

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"
)

func Middleware(duration string, storage *Storage, handler func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		content := storage.Get(r.RequestURI)
		if content != nil {
			fmt.Printf("Page served from cache: %s\n", r.RequestURI)
			w.Write(content)
		} else {
			c := httptest.NewRecorder()
			handler(c, r)

			for k, v := range c.Result().Header {
				w.Header()[k] = v
			}

			w.WriteHeader(c.Code)
			content := c.Body.Bytes()

			if d, err := time.ParseDuration(duration); err == nil {
				fmt.Printf("New page cached: %s for %s\n", r.RequestURI, duration)
				storage.Set(r.RequestURI, content, d)
			} else {
				fmt.Printf("Page not cached. err: %s\n", err)
			}

			w.Write(content)
		}

	})
}
