package summarizer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

func SummarizeEOChunk(text string, section int) (string, error) {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	ctx := context.Background()

	prompt := fmt.Sprintf("Summarize section %d of the following executive order:\n\n%s", section, text)

	for attempt := 1; attempt <= 5; attempt++ {
		resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: `
You are a civic analyst. Return a **concise, plain-English bullet-point summary** of a U.S. executive order. 
Use clear, neutral language. Format your response as 4–6 **Markdown-style bullet points** using \'- \' at the beginning of each item.

Example:

- Declares coal a national energy priority  
- Removes barriers to domestic coal production  
- Promotes coal tech for AI and data centers

Avoid legal phrases, disclaimers, or any additional preambles.
`},

				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
		})

		if err == nil {
			return strings.TrimSpace(resp.Choices[0].Message.Content), nil
		}

		if respErr, ok := err.(*openai.APIError); ok && respErr.HTTPStatusCode == 429 {
			wait := time.Duration(5*attempt) * time.Second
			fmt.Printf("Rate limited. Waiting %v before retrying...\n", wait)
			time.Sleep(wait)
			continue
		}

		return "", fmt.Errorf("OpenAI chunk error: %w", err)
	}

	return "", fmt.Errorf("OpenAI chunk %d failed after retries", section)
}

func SummarizeFinalSummaryFromChunks(chunks []string) (string, error) {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	ctx := context.Background()

	var input strings.Builder
	input.WriteString("Summarize the executive order below using 4 to 6 short, plain-language bullet points. Avoid legal phrasing.\n\n")
	for i, chunk := range chunks {
		input.WriteString(fmt.Sprintf("Section %d:\n%s\n\n", i+1, chunk))
	}

	prompt := fmt.Sprintf(`Summarize the executive order below using 4 to 6 short, plain-language Markdown bullet points. Avoid legal phrasing and formatting outside of bullets.
		%s`, input.String())

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You are a civic analyst. Return a concise bullet-point summary of a U.S. executive order in plain English. Use clear, short points (4–6 max). Avoid filler or legal language."},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("OpenAI final summary error: %w", err)
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

func SocioeconomicImpact(summary string) (map[string]string, error) {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	ctx := context.Background()

	prompt := fmt.Sprintf(`Given the following executive order summary:

"%s"

Write ONE short sentence for each of the following groups, explaining how this executive order might affect them. Be clear, direct, and avoid policy jargon.

Respond in strict JSON format like this:

{
  "average": "One sentence here.",
  "poorest": "One sentence here.",
  "richest": "One sentence here."
}`, summary)

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You are a civic analyst who explains policy impacts."},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("impact summary failed: %w", err)
	}

	raw := strings.TrimSpace(resp.Choices[0].Message.Content)
	return parseImpactResponse(raw), nil
}

func parseImpactResponse(raw string) map[string]string {
	var result map[string]string
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		result = map[string]string{
			"average": "Unable to parse structured response. Raw output:\n" + raw,
			"poorest": "",
			"richest": "",
		}
	}
	return result
}
