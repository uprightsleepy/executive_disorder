// summarizer/summarizer.go
package summarizer

import (
	"context"
	"fmt"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

var client *openai.Client

func init() {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		panic("OPENAI_API_KEY not set")
	}
	client = openai.NewClient(key)
}

func SplitTextIntoChunks(text string, maxChars int) []string {
	runes := []rune(text)
	var chunks []string
	for start := 0; start < len(runes); start += maxChars {
		end := start + maxChars
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}

func SummarizeEOChunk(text string, chunkNum int) (string, error) {
	ctx := context.Background()
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a civic analyst. Return a concise bullet-point summary of a U.S. executive order in plain English. Use clear, short points (4–6 max). Avoid filler or legal language.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: text,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("OpenAI chunk error: %w", err)
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

func SummarizeFinalSummaryFromChunks(chunks []string) (string, error) {
	ctx := context.Background()
	joined := strings.Join(chunks, "\n")
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a civic analyst. Return a final, concise, plain-English bullet-point summary of the executive order, combining all prior chunks. Use clear Markdown-style bullets. Avoid jargon or legalese.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: joined,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("OpenAI summary error: %w", err)
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

func SocioeconomicImpact(summary string) (map[string]string, error) {
	ctx := context.Background()
	prompt := fmt.Sprintf(`Given the following executive order summary:

"%s"

Write ONE short sentence for each of the following groups, explaining how this executive order might affect them. Be clear, direct, and avoid policy jargon. Think critically about structural and long-term effects.

Respond in strict JSON format like this:
{
  "average": "One sentence here.",
  "poorest": "One sentence here.",
  "richest": "One sentence here."
}`, summary)

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a civic economist focused on equity. Consider not only explicit benefits but also which group accrues the most power or economic gain.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("OpenAI impact error: %w", err)
	}

	raw := resp.Choices[0].Message.Content
	cleaned := strings.TrimSpace(raw)
	impact := make(map[string]string)
	
	// naive parsing fallback (use JSON decode if time permits)
	for _, line := range strings.Split(cleaned, "\n") {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				k := strings.ToLower(strings.Trim(parts[0], `" {}`))
				v := strings.Trim(parts[1], `" {},`)
				impact[k] = v
			}
		}
	}

	return impact, nil
}

func InferPrimaryBeneficiary(summary string, impact map[string]string) string {
	// Smart rule-based fallback that tilts toward structural gain
	summaryLower := strings.ToLower(summary)
	if strings.Contains(summaryLower, "corporate") || strings.Contains(summaryLower, "investment") || strings.Contains(summaryLower, "capital") {
		return "richest"
	}
	if strings.Contains(summaryLower, "job training") || strings.Contains(summaryLower, "food") || strings.Contains(summaryLower, "housing") {
		return "poorest"
	}
	if len(impact) == 3 {
		lengths := map[string]int{
			"average": len(impact["average"]),
			"poorest": len(impact["poorest"]),
			"richest": len(impact["richest"]),
		}
		longest := "average"
		for k, v := range lengths {
			if v > lengths[longest] {
				longest = k
			}
		}
		return longest
	}
	return "average"
}
