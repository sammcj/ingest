package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

type OllamaConfig struct {
	Model        string `json:"ollama_model"`
	PromptPrefix string `json:"ollama_prompt_prefix"`
	PromptSuffix string `json:"ollama_prompt_suffix"`
	AutoRun      bool   `json:"ollama_auto_run"`
}

type Config struct {
	Ollama []OllamaConfig `json:"ollama"`
}

func LoadConfig() (*Config, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".config", "ingest", "ingest.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return createDefaultConfig(configPath)
	}

	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func createDefaultConfig(configPath string) (*Config, error) {
	defaultConfig := Config{
		Ollama: []OllamaConfig{
			{
				Model:        "llama3.1:8b-instruct-q6_K",
				PromptPrefix: "Code: ",
				PromptSuffix: "",
				AutoRun:      false,
			},
		},
	}

	err := os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal default config: %w", err)
	}

	if err := os.WriteFile(configPath, file, 0644); err != nil {
		return nil, fmt.Errorf("failed to write default config file: %w", err)
	}

	return &defaultConfig, nil
}
