// handlers/firestore.go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/go-chi/chi/v5"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EORecord struct {
	EOID               string            `firestore:"EOID" json:"eo_id"`
	Title              string            `firestore:"Title" json:"title"`
	DateIssued         string            `firestore:"DateIssued" json:"date_issued"`
	President          string            `firestore:"President" json:"president"`
	HTMLURL            string            `firestore:"HTMLURL" json:"html_url"`
	PDFURL             string            `firestore:"PDFURL" json:"pdf_url"`
	Summary            []string          `firestore:"Summary" json:"summary"`
	Impact             map[string]string `firestore:"Impact" json:"impact"`
	// PrimaryBeneficiary represents the group that benefits most from the executive order.
	// Valid values: "average", "poorest", "richest".
	PrimaryBeneficiary string            `firestore:"PrimaryBeneficiary" json:"primary_beneficiary"`
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

	var record EORecord
	if err := doc.DataTo(&record); err != nil {
		log.Printf("❌ Failed to decode EO %s: %v", eoID, err)
		http.Error(w, "Failed to parse EO", http.StatusInternalServerError)
		return
	}

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
	monthRaw := strings.TrimSpace(r.URL.Query().Get("month"))
	monthFilter := fmt.Sprintf("%02s", strings.TrimLeft(monthRaw, "0"))
	dayRaw := strings.TrimSpace(r.URL.Query().Get("day"))
	dayFilter := fmt.Sprintf("%02s", strings.TrimLeft(dayRaw, "0"))

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

		var record EORecord
		if err := doc.DataTo(&record); err != nil {
			log.Printf("❌ Failed to decode doc: %v", err)
			continue
		}

		if presidentFilter != "" && !strings.Contains(strings.ToLower(record.President), presidentFilter) {
			continue
		}

		if yearFilter != "" {
			if !strings.HasPrefix(record.DateIssued, yearFilter) {
				continue
			}
			if monthFilter != "" {
				if len(record.DateIssued) < 7 || record.DateIssued[5:7] != monthFilter {
					continue
				}
				if dayFilter != "" && (len(record.DateIssued) < 10 || record.DateIssued[8:10] != dayFilter) {
					continue
				}
			}
		}
		results = append(results, record)
	}

	w.Header().Set("Content-Type", "application/json")
	if len(results) == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "No matching executive orders found.",
		})
		return
	}
	json.NewEncoder(w).Encode(results)
}

func Exists(ctx context.Context, client *firestore.Client, eoID string) (bool, error) {
	_, err := client.Collection("summaries").Doc(eoID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// SaveSummary saves the EORecord into Firestore. Ensure PrimaryBeneficiary is set before calling.
func SaveSummary(ctx context.Context, client *firestore.Client, record EORecord) error {
	_, err := client.Collection("summaries").Doc(record.EOID).Set(ctx, record)
	return err
}
