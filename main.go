package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
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

func ExtractMainContent(htmlContent string) (string, error) {
	// Create a new document from the HTML string
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("error creating document from HTML: %w", err)
	}

	// Find the <main> tag and extract its contents
	mainContent, err := doc.Find("main").Html()
	if err != nil {
		return "", fmt.Errorf("error extracting <main> content: %w", err)
	}

	return mainContent, nil
}

func main() {
	ctx := context.Background()

	// Set default URL to local file path
	localHtml := flag.String("local-html", "data/claude.html", "Path html page containing Claude pricing")
	scraperModel := flag.String("scraper-model", "gemini-1.5-flash-latest", "Gemini model used for scraping prices")
	flag.Parse()

	htmlContent, err := getHTML(*localHtml)
	if err != nil {
		log.Fatalf("Error fetching HTML: %v", err)
	}

	client, model := newGeminiClientModel(ctx, *scraperModel)
	defer client.Close()
	if err != nil {
		log.Fatalf("Error creating Gemini client: %v", err)
	}

	start := time.Now()
	priceData, resp, err := extractPrices(ctx, model, htmlContent)
	if err != nil {
		log.Fatalf("Error extracting prices: %v", err)
	}
	elapsed := time.Since(start)

	fmt.Printf("Extracted Prices: %+v\n", priceData)
	fmt.Printf("\nTokens generated: %d\n", resp.UsageMetadata.CandidatesTokenCount)
	fmt.Printf("Input token count: %d\n", resp.UsageMetadata.PromptTokenCount)
	fmt.Printf("Output tokens per Second: %.2f\n", float64(resp.UsageMetadata.CandidatesTokenCount)/elapsed.Seconds())
	fmt.Printf("Total Execution Time: %s\n", elapsed)

}

func newGeminiClientModel(ctx context.Context, modelString string) (*genai.Client, *genai.GenerativeModel) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		log.Fatal(err)
	}

	model := client.GenerativeModel(modelString)

	return client, model
}

func getHTML(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	mainContent, err := ExtractMainContent(string(content))
	if err != nil {
		return "", err
	}

	return string(mainContent), nil
}

func extractPrices(ctx context.Context, model *genai.GenerativeModel, htmlContent string) (*PriceScraperResponse, *genai.GenerateContentResponse, error) {
	systemPrompt := generateTypedHtmlScraperSystemPrompt()
	fmt.Printf("System Prompt: %s\n", systemPrompt)

	resp, err := model.GenerateContent(ctx, genai.Text(systemPrompt+htmlContent))
	if err != nil {
		log.Fatal(err)
	}

	// Parts can be Text, FunctionResponse or a Blob but we know it's Text in this case so the cast is good
	content := string(resp.Candidates[0].Content.Parts[0].(genai.Text))
	fmt.Printf("--- Received \n%s\n---", content)
	var decodedMessage PriceScraperResponse
	err = json.Unmarshal([]byte(content), &decodedMessage)
	if err != nil {
		return nil, resp, fmt.Errorf("JSON decoding error: %v", err)
	}

	return &decodedMessage, resp, nil
}

func generateTypedHtmlScraperSystemPrompt() string {
	exampleResponse := PriceScraperResponse{
		ModelPrices: []ModelPrice{
			{
				ModelName: "gpt-3.5-turbo",
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
				ModelName: "gpt-4-32k",
				InputTokenPrice: TokenPrice{
					CostPerMillion: 0.03,
					Currency:       "USD",
				},
				OutputTokenPrice: TokenPrice{
					CostPerMillion: 0.04,
					Currency:       "USD",
				},
			},
		},
	}

	jsonExample, err := json.MarshalIndent(exampleResponse, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling example response: %v", err)
	}

	return fmt.Sprintf("You are a price extraction API that takes public data from HTML and extracts data, returning it EXACTLY using the PriceScraperResponse schema so it can be read by a JSON parser, with no escape quotes before or after, exemplified as:\n%s", string(jsonExample))
}
