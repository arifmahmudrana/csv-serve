package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (app *application) Memcache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.infoLog.Println("Memcache: Running middleware")

		if id := chi.URLParam(r, "promotionID"); id != "" {
			var res Promotion
			found, err := app.m.Get(id, &res)

			if err != nil {
				app.errorLog.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Something went wrong"))
				return
			}

			if found {
				app.infoLog.Println("Memcache: found serving from cache ", id)
				encoder := json.NewEncoder(w)
				if err := encoder.Encode(res); err != nil {
					app.errorLog.Println(err)
				}

				return
			}

			app.infoLog.Println("Memcache: miss ", id)
		}

		next.ServeHTTP(w, r)
	})
}
