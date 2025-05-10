package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-resty/resty/v2"
)

type ExecutiveOrder struct {
	EOID        string `json:"eo_id"`
	Title       string `json:"title"`
	President   string `json:"president"`
	DateIssued  string `json:"date_issued"`
	HTMLURL     string `json:"html_url"`
	PDFURL      string `json:"pdf_url"`
}

func derivePresident(dateStr string) string {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "Unknown"
	}
	switch {
	case date.After(time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)):
		return "Donald Trump"
	case date.After(time.Date(2021, 1, 20, 0, 0, 0, 0, time.UTC)):
		return "Joe Biden"
	case date.After(time.Date(2017, 1, 20, 0, 0, 0, 0, time.UTC)):
		return "Donald Trump"
	case date.After(time.Date(2009, 1, 20, 0, 0, 0, 0, time.UTC)):
		return "Barack Obama"
	case date.After(time.Date(2001, 1, 20, 0, 0, 0, 0, time.UTC)):
		return "George W. Bush"
	case date.After(time.Date(1993, 1, 20, 0, 0, 0, 0, time.UTC)):
		return "Bill Clinton"
	default:
		return "Unknown"
	}
}

func GetAllExecutiveOrders(ctx context.Context) ([]ExecutiveOrder, error) {
	client := resty.New()
	baseURL := "https://www.federalregister.gov/api/v1/documents.json"
	var allOrders []ExecutiveOrder
	page := 1

	for {
		resp, err := client.R().SetContext(ctx).SetQueryParams(map[string]string{
			"conditions[term]": "Executive Order",
			"order":            "desc",
			"per_page":         "100",
			"page":             fmt.Sprintf("%d", page),
		}).Get(baseURL)
		if err != nil {
			return nil, fmt.Errorf("API request failed on page %d: %w", page, err)
		}

		type apiResponse struct {
			Results []struct {
				Title           string `json:"title"`
				DocumentNumber  string `json:"document_number"`
				HTMLURL         string `json:"html_url"`
				PDFURL          string `json:"pdf_url"`
				PublicationDate string `json:"publication_date"`
				Type            string `json:"type"`
			} `json:"results"`
		}

		var data apiResponse
		if err := json.Unmarshal(resp.Body(), &data); err != nil {
			return nil, fmt.Errorf("unmarshal error on page %d: %w", page, err)
		}
		if len(data.Results) == 0 {
			break
		}

		for _, item := range data.Results {
			if item.Type != "Presidential Document" {
				continue
			}
			eo := ExecutiveOrder{
				EOID:        item.DocumentNumber,
				Title:       item.Title,
				DateIssued:  item.PublicationDate,
				HTMLURL:     item.HTMLURL,
				PDFURL:      item.PDFURL,
				President:   derivePresident(item.PublicationDate),
			}
			allOrders = append(allOrders, eo)
		}
		page++
		if page > 5 {
			break
		}
	}

	sort.Slice(allOrders, func(i, j int) bool {
		d1, _ := time.Parse("2006-01-02", allOrders[i].DateIssued)
		d2, _ := time.Parse("2006-01-02", allOrders[j].DateIssued)
		return d1.After(d2)
	})

	return allOrders, nil
}
