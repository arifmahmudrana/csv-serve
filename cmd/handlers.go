package main

import (
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/arifmahmudrana/csv-serve/cassandra"
	"github.com/go-chi/chi/v5"
)

func (app *application) Ping(w http.ResponseWriter, r *http.Request) {
	app.infoLog.Println("[api][handlerss-api][Ping] =>")
	w.Write([]byte("PONG!"))
}

func (app *application) CreatePromotion(w http.ResponseWriter, r *http.Request) {
	app.infoLog.Println("[api][handlerss-api][CreatePromotion] =>")

	dec := json.NewDecoder(r.Body)
	var p Promotion
	if err := dec.Decode(&p); err != nil {
		app.errorLog.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return
	}

	expirationDate, err := time.Parse("2006-01-02 15:04:05 -0700 MST", p.ExpirationDate)
	if err != nil {
		app.errorLog.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return
	}

	d := cassandra.Promotion{
		ID:             p.ID,
		Price:          p.Price,
		ExpirationDate: expirationDate.UTC(),
	}
	if err := app.db.InsertPromotions([]cassandra.Promotion{d}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong"))
		return
	}

	w.Write([]byte("OK"))
}

type Promotion struct {
	ID             string  `json:"id"`
	Price          float64 `json:"price"`
	ExpirationDate string  `json:"expiration_date"`
}

const (
	dateFormat = "2006-01-02 15:04:05"
)

func (p *Promotion) transform(d *cassandra.Promotion) {
	p.ID = d.ID
	p.Price = math.Round(d.Price*100) / 100
	p.ExpirationDate = d.ExpirationDate.Format(dateFormat)
}

func (app *application) GetPromotion(w http.ResponseWriter, r *http.Request) {
	app.infoLog.Println("[api][handlerss-api][GetPromotion] =>")

	id := chi.URLParam(r, "promotionID")
	d, err := app.db.GetPromotionByID(id)
	if err != nil {
		app.errorLog.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong"))
		return
	}
	if d == nil {
		app.infoLog.Printf("[api][handlerss-api][GetPromotion] => promotion %s not found\n", id)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
		return
	}

	var res Promotion
	res.transform(d)
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(res); err != nil {
		app.errorLog.Println(err)
	}
}
