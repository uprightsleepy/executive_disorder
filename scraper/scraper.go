package scraper

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/ledongthuc/pdf"
)

func ExtractTextFromPDF(pdfURL string) (string, error) {
	resp, err := http.Get(pdfURL)
	if err != nil {
		return "", fmt.Errorf("failed to download PDF: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read PDF body: %w", err)
	}

	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	var buf bytes.Buffer
	n := r.NumPage()
	for i := 1; i <= n; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		content, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		buf.WriteString(content)
	}

	text := buf.String()
	if len(text) < 100 {
		return "", fmt.Errorf("scraped text too short — likely failed")
	}
	return text, nil
}
