package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()

	mux.Get("/api/ping", app.Ping)
	mux.Get("/api/promotions/{promotionID}", app.GetPromotion)
	mux.Post("/api/promotions", app.CreatePromotion)

	return mux
}
