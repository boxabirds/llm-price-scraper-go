package main

import (
	"encoding/json"
	"fmt"
	"log"
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

func main() {
	// Example usage
	resp := PriceScraperResponse{
		ModelPrices: []ModelPrice{
			{
				ModelName: "gpt-3.5-turbo-0125",
				InputTokenPrice: TokenPrice{
					CostPerMillion: 0.01,
					Currency:       "USD",
				},
				OutputTokenPrice: TokenPrice{
					CostPerMillion: 0.02,
					Currency:       "USD",
				},
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
	}
	fmt.Println("Marshaled JSON:", string(data))

	var newResp PriceScraperResponse
	err = json.Unmarshal(data, &newResp)
	if err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
	}
	fmt.Printf("Unmarshaled JSON: %+v\n", newResp)
}
