package main

import (
	"log"
	"net/http"
	"os"

	"executive-disorder.com/handlers"
	"github.com/go-chi/chi/v5"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stderr)
	log.Println("Logging explicitly to stderr...")

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("➡️ %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	r.Get("/api/eos/{eo_id}", handlers.GetEOByID)
	r.Get("/api/eos", handlers.GetAllEOs)

	log.Println("✅ EO API running on :8080")
	http.ListenAndServe(":8080", r)
}
