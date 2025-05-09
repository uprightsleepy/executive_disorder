package scraper

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/ledongthuc/pdf"
)

func ExtractTextFromPDF(pdfURL string) (string, error) {
	tmpFile, err := os.CreateTemp("", "eo_*.pdf")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	resp, err := http.Get(pdfURL)
	if err != nil {
		return "", fmt.Errorf("failed to download PDF: %w", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save PDF: %w", err)
	}
	tmpFile.Close()

	f, r, err := pdf.Open(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var text string
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		for _, block := range page.Content().Text {
			text += block.S + "\n"
		}
	}

	if len(text) < 300 {
		return "", fmt.Errorf("PDF text too short")
	}

	return text, nil
}