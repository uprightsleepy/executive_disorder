package main

import (
	"log"
	"net/http"

	"executive-disorder.com/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/api/eos/{eo_id}", handlers.GetEOByID)
	r.Get("/api/eos", handlers.GetAllEOs)

	log.Println("✅ EO API running on :8080")
	http.ListenAndServe(":8080", r)
}