package main

import (
	"net/http"
)

func (app *application) Ping(w http.ResponseWriter, r *http.Request) {
	app.infoLog.Println("[api][handlerss-api][Ping] =>")
	w.Write([]byte("PONG!"))
}
