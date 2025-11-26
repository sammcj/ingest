package token

import (
	"fmt"
	"sync"

	"github.com/fatih/color"
	"github.com/pkoukk/tiktoken-go"
	"github.com/sammcj/ingest/utils"
)

var (
	apiUsedOnce     sync.Once
	apiMessageShown bool
)

// CorrectionMultiplier is applied to offline token counts for better accuracy.
// Based on empirical analysis comparing offline tokeniser with Anthropic API,
// the offline tokeniser underestimates by approximately 16.61%.
// A 1.18x multiplier reduces average error from ~17% to ~2%.
const CorrectionMultiplier = 1.18

func GetTokenizer(encoding string) *tiktoken.Tiktoken {
	var err error
	var tk *tiktoken.Tiktoken

	switch encoding {
	case "o200k", "gpt-4o", "gpt-4.1", "gpt-4.5":
		tk, err = tiktoken.GetEncoding("o200k_base")
	case "cl100k", "llama3", "llama-3", "gpt-4", "gpt-3.5-turbo", "text-embedding-3-large", "text-embedding-3-small", "text-embedding-ada-002", "text-ada-002":
		tk, err = tiktoken.GetEncoding("cl100k_base")
	case "p50k":
		tk, err = tiktoken.GetEncoding("p50k_base")
	case "r50k", "gpt2", "text-ada-001", "text-curie-001", "text-babbage-001":
		tk, err = tiktoken.GetEncoding("r50k_base")
	default:
		// Default to o200k_base for modern Anthropic Claude and OpenAI models
		tk, err = tiktoken.GetEncoding("o200k_base")
	}

	if err != nil {
		fmt.Printf("Failed to get tokenizer: %v\n", err)
		return nil
	}
	return tk
}

func GetModelInfo(encoding string) string {
	switch encoding {
	case "o200k", "gpt-4o", "gpt-4.1", "gpt-4.5":
		return "OpenAI gpt-4+, Anthropic Claude Haiku/Sonnet/Opus 3+ models"
	case "cl100k":
		return "Llama3, OpenAI <4o models, text-embedding-ada-002, gpt-4 etc..."
	case "p50k":
		return "OpenAI code models, text-davinci-002, text-davinci-003 etc..."
	case "r50k", "gpt2", "llama2", "llama-2":
		return "Legacy models like llama2, GPT-3, davinci etc..."
	default:
		return "OpenAI gpt-4+, Anthropic Claude Haiku/Sonnet/Opus 3+ models"
	}
}

func CountTokens(rendered string, encoding string, useAnthropicAPI bool, noCorrection bool) int {
	if useAnthropicAPI {
		count, err := CountTokensAPI(rendered)
		if err != nil {
			if !apiMessageShown {
				fmt.Printf("Warning: Failed to count tokens using Anthropic API: %v\nFalling back to offline tokeniser\n", err)
				apiMessageShown = true
			}
			// Fall back to offline tokenizer
		} else {
			apiUsedOnce.Do(func() {
				fmt.Println() // Add blank line for visibility
				utils.PrintColouredMessage("✓", fmt.Sprintf("Using Anthropic API (%s) for token counting", AnthropicModel), color.FgYellow)
			})
			return count
		}
	}

	tk := GetTokenizer(encoding)
	if tk == nil {
		return 0
	}

	tokens := tk.Encode(rendered, nil, nil)
	rawCount := len(tokens)

	// Apply correction multiplier for better accuracy unless disabled
	if noCorrection {
		return rawCount
	}

	correctedCount := float64(rawCount) * CorrectionMultiplier
	return int(correctedCount)
}

// CountTokensBatch counts tokens for multiple strings, using parallel API calls if Anthropic API is enabled.
// Processes up to 4 items concurrently when using the API.
func CountTokensBatch(contents []string, encoding string, useAnthropicAPI bool, noCorrection bool) []int {
	if len(contents) == 0 {
		return []int{}
	}

	if useAnthropicAPI {
		counts, err := CountTokensBatchAPI(contents, 4)
		if err != nil {
			if !apiMessageShown {
				fmt.Printf("Warning: Failed to count tokens using Anthropic API: %v\nFalling back to offline tokeniser\n", err)
				apiMessageShown = true
			}
			// Fall back to offline tokeniser for all items
		} else {
			apiUsedOnce.Do(func() {
				fmt.Println() // Add blank line for visibility
				utils.PrintColouredMessage("✓", fmt.Sprintf("Using Anthropic API (%s) for token counting", AnthropicModel), color.FgYellow)
			})
			return counts
		}
	}

	// Use offline tokeniser for all items
	results := make([]int, len(contents))
	tk := GetTokenizer(encoding)
	if tk == nil {
		return results
	}

	for i, content := range contents {
		tokens := tk.Encode(content, nil, nil)
		rawCount := len(tokens)

		if noCorrection {
			results[i] = rawCount
		} else {
			correctedCount := float64(rawCount) * CorrectionMultiplier
			results[i] = int(correctedCount)
		}
	}

	return results
}
