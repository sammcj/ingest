package token

import (
	"fmt"

	"github.com/pkoukk/tiktoken-go"
)

func GetTokenizer(encoding string) *tiktoken.Tiktoken {
	var err error
	var tk *tiktoken.Tiktoken

	switch encoding {
	case "cl100k":
		tk, err = tiktoken.GetEncoding("cl100k_base")
	case "p50k":
		tk, err = tiktoken.GetEncoding("p50k_base")
	case "r50k", "gpt2":
		tk, err = tiktoken.GetEncoding("r50k_base")
	default:
		tk, err = tiktoken.GetEncoding("cl100k_base")
	}

	if err != nil {
		fmt.Printf("Failed to get tokenizer: %v\n", err)
		return nil
	}
	return tk
}

func GetModelInfo(encoding string) string {
	switch encoding {
	case "cl100k":
		return "ChatGPT models, text-embedding-ada-002"
	case "p50k":
		return "Code models, text-davinci-002, text-davinci-003"
	case "r50k", "gpt2":
		return "GPT-3 models like davinci"
	default:
		return "ChatGPT models, text-embedding-ada-002"
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
