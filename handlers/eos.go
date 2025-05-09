package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/go-chi/chi/v5"
	"google.golang.org/api/iterator"
)

type EORecord struct {
	EOID       string            `json:"eo_id"`
	Title      string            `json:"title"`
	DateIssued string            `json:"date_issued"`
	President  string            `json:"president"`
	HTMLURL    string            `json:"html_url"`
	PDFURL     string            `json:"pdf_url"`
	Summary    []string          `json:"summary"`
	Impact     map[string]string `json:"impact"`
}

func GetEOByID(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	eoID := chi.URLParam(r, "eo_id")
	log.Printf("🔍 Looking up EO: %s", eoID)

	client, err := firestore.NewClientWithDatabase(ctx, "executive-disorder", "eo-summary-db")
	if err != nil {
		http.Error(w, "Failed to init Firestore", http.StatusInternalServerError)
		log.Println("Firestore error:", err)
		return
	}
	defer client.Close()

	doc, err := client.Collection("summaries").Doc(eoID).Get(ctx)
	if err != nil {
		http.Error(w, "EO not found", http.StatusNotFound)
		return
	}

	record := buildEORecord(doc.Data())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

func GetAllEOs(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	client, err := firestore.NewClientWithDatabase(ctx, "executive-disorder", "eo-summary-db")
	if err != nil {
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		log.Println("Firestore error:", err)
		return
	}
	defer client.Close()

	presidentFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("president")))
yearFilter := strings.TrimSpace(r.URL.Query().Get("year"))
	iter := client.Collection("summaries").Documents(ctx)
	var results []EORecord

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			http.Error(w, "Failed to read documents", http.StatusInternalServerError)
			return
		}
		data := doc.Data()
		if presidentFilter != "" && !strings.Contains(strings.ToLower(getString(data, "president")), presidentFilter) {
			continue
		}
		if yearFilter != "" && !strings.HasPrefix(getString(data, "date_issued"), yearFilter) {
			continue
		}
		record := buildEORecord(data)
		results = append(results, record)
	}

	w.Header().Set("Content-Type", "application/json")
	if len(results) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "No matching executive orders found.",
		})
		return
	}
	json.NewEncoder(w).Encode(results)
}

func buildEORecord(data map[string]interface{}) EORecord {
	summary := []string{}
	if markdown, ok := data["markdown_summary"].(string); ok {
		summary = splitMarkdownBullets(markdown)
	}

	return EORecord{
		EOID:       getString(data, "eo_id"),
		Title:      getString(data, "title"),
		DateIssued: getString(data, "date_issued"),
		President:  getString(data, "president"),
		HTMLURL:    getString(data, "html_url"),
		PDFURL:     getString(data, "pdf_url"),
		Summary:    summary,
		Impact: map[string]string{
			"average": getString(data, "impact_average"),
			"poorest": getString(data, "impact_poorest"),
			"richest": getString(data, "impact_richest"),
		},
	}
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func splitMarkdownBullets(summary string) []string {
	lines := []string{}
	for _, line := range strings.Split(summary, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			lines = append(lines, strings.TrimPrefix(line, "- "))
		}
	}
	return lines
}
