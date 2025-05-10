// ingester/ingester.go
package main

import (
	"context"
	"log"
	"strings"
	"time"

	"executive-disorder.com/fetcher"
	"executive-disorder.com/scraper"
	storage "executive-disorder.com/handlers"
	"executive-disorder.com/summarizer"
	"github.com/joho/godotenv"

	"cloud.google.com/go/firestore"
)

const workerCount = 1
const maxRetries = 3

var retryCount = make(map[string]int)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()
	client, err := firestore.NewClientWithDatabase(ctx, "executive-disorder", "eo-summary-db")
	if err != nil {
		log.Fatalf("Failed to connect to Firestore: %v", err)
	}
	defer client.Close()

	eos, err := fetcher.GetAllExecutiveOrders(ctx)
	if err != nil {
		log.Fatalf("Failed to fetch executive orders: %v", err)
	}

	log.Printf("Processing %d executive orders with %d workers...", len(eos), workerCount)

	jobs := make(chan fetcher.ExecutiveOrder, len(eos)*2) // allow for retries
	results := make(chan string, len(eos))

	for w := 1; w <= workerCount; w++ {
		go worker(ctx, w, client, jobs, results)
	}

	for _, eo := range eos {
		jobs <- eo
	}
	close(jobs)

	for i := 0; i < len(eos); i++ {
		<-results
	}

	log.Println("✅ Batch processing complete.")
}

func worker(ctx context.Context, id int, client *firestore.Client, jobs chan fetcher.ExecutiveOrder, results chan<- string) {
	for eo := range jobs {
		log.Printf("[Worker %d] Processing EO: %s", id, eo.EOID)

		exists, err := storage.Exists(ctx, client, eo.EOID)
		if err != nil {
			log.Printf("[Worker %d] Exists check failed: %v", id, err)
			results <- eo.EOID
			continue
		}
		if exists {
			log.Printf("[Worker %d] ✅ Already exists, skipping.", id)
			results <- eo.EOID
			continue
		}

		text, err := scraper.ExtractTextFromPDF(eo.PDFURL)
		if err != nil {
			log.Printf("[Worker %d] ❌ PDF extraction failed: %v", id, err)
			results <- eo.EOID
			continue
		}

		chunks := summarizer.SplitTextIntoChunks(text, 3000)
		var chunkSummaries []string
		for j, chunk := range chunks {
			log.Printf("[Worker %d] Summarizing chunk %d/%d...", id, j+1, len(chunks))
			summary, err := summarizer.SummarizeEOChunk(chunk, j+1)
			if err != nil {
				log.Printf("[Worker %d] Chunk %d failed: %v", id, j+1, err)
				continue
			}
			chunkSummaries = append(chunkSummaries, summary)
			time.Sleep(2 * time.Second)
		}

		finalSummary, err := summarizer.SummarizeFinalSummaryFromChunks(chunkSummaries)
		if err != nil {
			log.Printf("[Worker %d] Final summary failed: %v", id, err)
			results <- eo.EOID
			continue
		}

		impact, err := summarizer.SocioeconomicImpact(finalSummary)
		if err != nil {
			log.Printf("[Worker %d] Impact analysis failed: %v", id, err)
			results <- eo.EOID
			continue
		}

		primary := summarizer.InferPrimaryBeneficiary(finalSummary, impact)

		record := storage.EORecord{
			EOID:       eo.EOID,
			Title:      eo.Title,
			DateIssued: eo.DateIssued,
			President:  eo.President,
			HTMLURL:    eo.HTMLURL,
			PDFURL:     eo.PDFURL,
			Summary:    splitMarkdownBullets(finalSummary),
			Impact: map[string]string{
				"average": impact["average"],
				"poorest": impact["poorest"],
				"richest": impact["richest"],
			},
			PrimaryBeneficiary: primary,
		}

		if record.EOID == "" || record.Title == "" || record.DateIssued == "" ||
			record.HTMLURL == "" || record.PDFURL == "" || len(record.Summary) == 0 ||
			record.Impact["average"] == "" || record.Impact["poorest"] == "" || record.Impact["richest"] == "" {

			retryCount[eo.EOID]++
			if retryCount[eo.EOID] <= maxRetries {
				log.Printf("[Worker %d] ❌ Missing fields, re-queuing EO %s (retry %d)", id, eo.EOID, retryCount[eo.EOID])
				go func(e fetcher.ExecutiveOrder) { jobs <- e }(eo)
			} else {
				log.Printf("[Worker %d] ❌ EO %s failed too many times, skipping.", id, eo.EOID)
			}
			results <- eo.EOID
			continue
		}

		err = storage.SaveSummary(ctx, client, record)
		if err != nil {
			log.Printf("[Worker %d] Failed to save EO: %v", id, err)
		} else {
			log.Printf("[Worker %d] ✅ Saved EO %s", id, eo.EOID)
		}

		results <- eo.EOID
	}
}

func splitMarkdownBullets(markdown string) []string {
	var bullets []string
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			bullets = append(bullets, strings.TrimPrefix(line, "- "))
		}
	}
	return bullets
}
