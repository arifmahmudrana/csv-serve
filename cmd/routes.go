package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()

	mux.Get("/api/ping", app.Ping)
	mux.With(app.Memcache).
		Get("/api/promotions/{promotionID}", app.GetPromotion)
	mux.Post("/api/promotions", app.CreatePromotion)
	mux.Post("/api/promotions/truncate", app.TruncatePromotion)

	return mux
}
