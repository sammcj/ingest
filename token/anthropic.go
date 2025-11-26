package token

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

type AnthropicTokenCountRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicTokenCountResponse struct {
	InputTokens int `json:"input_tokens"`
}

// AnthropicModel is the Claude model used for token counting
const AnthropicModel = "claude-sonnet-4-5"

func getAnthropicAPIKey() (string, error) {
	// Check environment variables in order of preference
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		return key, nil
	}
	if key := os.Getenv("ANTHROPIC_TOKEN"); key != "" {
		return key, nil
	}
	if key := os.Getenv("ANTHROPIC_TOKEN_COUNT_KEY"); key != "" {
		return key, nil
	}
	return "", fmt.Errorf("no Anthropic API key found in environment variables (checked ANTHROPIC_API_KEY, ANTHROPIC_TOKEN, ANTHROPIC_TOKEN_COUNT_KEY)")
}

func CountTokensAPI(content string) (int, error) {
	apiKey, err := getAnthropicAPIKey()
	if err != nil {
		return 0, err
	}

	requestBody := AnthropicTokenCountRequest{
		Model: AnthropicModel,
		Messages: []Message{
			{
				Role:    "user",
				Content: content,
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages/count_tokens", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response AnthropicTokenCountResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.InputTokens, nil
}

// CountTokensBatchAPI counts tokens for multiple content strings in parallel batches.
// Processes up to batchSize items concurrently to avoid overwhelming the API.
func CountTokensBatchAPI(contents []string, batchSize int) ([]int, error) {
	if len(contents) == 0 {
		return []int{}, nil
	}

	results := make([]int, len(contents))
	errors := make([]error, len(contents))
	var wg sync.WaitGroup

	// Process in batches
	for i := 0; i < len(contents); i += batchSize {
		end := min(i+batchSize, len(contents))

		// Process this batch concurrently
		for j := i; j < end; j++ {
			wg.Add(1)
			go func(index int, content string) {
				defer wg.Done()
				count, err := CountTokensAPI(content)
				if err != nil {
					errors[index] = err
					results[index] = 0
				} else {
					results[index] = count
				}
			}(j, contents[j])
		}

		// Wait for this batch to complete before starting the next
		wg.Wait()
	}

	// Check if any errors occurred
	var firstError error
	for _, err := range errors {
		if err != nil {
			firstError = err
			break
		}
	}

	return results, firstError
}
