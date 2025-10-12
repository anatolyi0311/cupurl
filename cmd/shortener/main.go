package main

import (
	"net/http"

	"github.com/anatolyi0311/cupurl/internal/router"
)

func main() {
	mux := router.Router()
	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
