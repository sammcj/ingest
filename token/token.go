package token

import (
	"fmt"

	"github.com/pkoukk/tiktoken-go"
)

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

func CountTokens(rendered string, encoding string) int {
	tk := GetTokenizer(encoding)
	if tk == nil {
		return 0
	}

	tokens := tk.Encode(rendered, nil, nil)
	return len(tokens)
}
