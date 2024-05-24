package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/chromedp/chromedp"
	claude "github.com/potproject/claude-sdk-go"
)

type TokenPrice struct {
	CostPerMillion float32 `json:"costPerMillion"`
	Currency       string  `json:"currency"`
}

type ModelPrice struct {
	ModelName        string     `json:"modelName"`
	InputTokenPrice  TokenPrice `json:"inputTokenPrice"`
	OutputTokenPrice TokenPrice `json:"outputTokenPrice"`
}

type PriceScraperResponse struct {
	ModelPrices []ModelPrice `json:"modelPrices"`
}

const DEFAULT_CLAUDE_MODEL = "claude-3-haiku-20240307"

func main() {
	ctx := context.Background()

	url := flag.String("openai", "https://openai.com/api/pricing/", "URL to fetch HTML from")
	flag.Parse()

	htmlContent, err := fetchHTML(*url)
	if err != nil {
		log.Fatalf("Error fetching HTML: %v", err)
	}

	fmt.Printf("HTML: %s\n", htmlContent)

	client := newClaudeClient()

	systemPrompt := generateSystemPrompt()
	fmt.Printf("System Prompt: %s\n", systemPrompt)
	priceData, err := extractPrices(ctx, client, systemPrompt, htmlContent)
	if err != nil {
		log.Fatalf("Error extracting prices: %v", err)
	}

	fmt.Printf("Extracted Prices: %+v\n", priceData)
}

func newClaudeClient() *claude.Client {
	apiKey, exists := os.LookupEnv("ANTHROPIC_API_KEY")
	if !exists {
		log.Fatal("ANTHROPIC_API_KEY environment variable is not set")
	}
	client := claude.NewClient(apiKey)
	return client
}

// func fetchHTML(url string) (string, error) {
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return "", err
// 	}

// 	return string(body), nil
// }

func fetchHTML(url string) (string, error) {
	// Create context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var htmlContent string

	// Run task
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.OuterHTML("html", &htmlContent),
	)
	if err != nil {
		return "", err
	}

	return htmlContent, nil
}

func extractPrices(ctx context.Context, client *claude.Client, prompt, htmlContent string) (*PriceScraperResponse, error) {
	req := claude.RequestBodyMessages{
		Model:       "claude-3-haiku-20240307",
		MaxTokens:   1000,
		System:      prompt,
		Temperature: 0.0,
		Messages: []claude.RequestBodyMessagesMessages{
			{
				Role:    claude.MessagesRoleUser,
				Content: htmlContent,
			},
		},
	}

	resp, err := client.CreateMessages(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.Content) == 0 {
		return nil, fmt.Errorf("no response from Claude")
	}

	content := resp.Content[0].Text
	fmt.Printf("Received: \n%s\n", content)
	var decodedMessage PriceScraperResponse
	err = json.Unmarshal([]byte(content), &decodedMessage)
	if err != nil {
		return nil, fmt.Errorf("JSON decoding error: %v", err)
	}

	return &decodedMessage, nil
}

func generateSystemPrompt() string {
	exampleResponse := PriceScraperResponse{
		ModelPrices: []ModelPrice{
			{
				ModelName: "GPT-3.5",
				InputTokenPrice: TokenPrice{
					CostPerMillion: 0.01,
					Currency:       "USD",
				},
				OutputTokenPrice: TokenPrice{
					CostPerMillion: 0.02,
					Currency:       "USD",
				},
			},
			{
				ModelName: "GPT-4",
				InputTokenPrice: TokenPrice{
					CostPerMillion: 0.03,
					Currency:       "EUR",
				},
				OutputTokenPrice: TokenPrice{
					CostPerMillion: 0.04,
					Currency:       "EUR",
				},
			},
		},
	}

	jsonExample, err := json.MarshalIndent(exampleResponse, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling example response: %v", err)
	}

	return fmt.Sprintf("You are a price extraction API that takes public data from HTML and extracts data formatted using the PriceScraperResponse schema, exemplified below:\n%s", string(jsonExample))
}
