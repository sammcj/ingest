package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

type OllamaConfig struct {
	Model        string `json:"llm_model"`
	PromptPrefix string `json:"llm_prompt_prefix"`
	PromptSuffix string `json:"llm_prompt_suffix"`
	AutoRun      bool   `json:"llm_auto_run"`
}

type Config struct {
	Ollama []OllamaConfig `json:"ollama"`
	LLM    LLMConfig      `json:"llm"`
	AutoSave bool         `json:"auto_save"`
}

type LLMConfig struct {
	AuthToken        string   `json:"llm_auth_token"`
	BaseURL          string   `json:"llm_base_url"`
	Model            string   `json:"llm_model"`
	MaxTokens        int      `json:"llm_max_tokens"`
	Temperature      *float32 `json:"llm_temperature,omitempty"`
	TopP             *float32 `json:"llm_top_p,omitempty"`
	PresencePenalty  *float32 `json:"llm_presence_penalty,omitempty"`
	FrequencyPenalty *float32 `json:"llm_frequency_penalty,omitempty"`
	APIType          string   `json:"llm_api_type"`
}

// loads the config file
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

	// Set default values for LLM config
	if config.LLM.AuthToken == "" {
		config.LLM.AuthToken = os.Getenv("OPENAI_API_KEY")
	}
	if config.LLM.BaseURL == "" {
		config.LLM.BaseURL = getDefaultBaseURL()
	}
	if config.LLM.Model == "" {
		config.LLM.Model = "llama3.1:8b-instruct-q6_K"
	}
	if config.LLM.MaxTokens == 0 {
		config.LLM.MaxTokens = 2048
	}
	if config.LLM.APIType == "" {
		config.LLM.APIType = "OPEN_AI"
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
		LLM: LLMConfig{
			BaseURL:   getDefaultBaseURL(),
			Model:     "llama3.1:8b-instruct-q6_K",
			MaxTokens: 2048,
		},
		AutoSave: false,
	}

	err := os.MkdirAll(filepath.Dir(configPath), 0750)
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

// returns the default base URL for the LLM API
func getDefaultBaseURL() string {
	if url := os.Getenv("OPENAI_API_BASE"); url != "" {
		return url
	}
	if url := os.Getenv("llm_HOST"); url != "" {
		return url + "/v1"
	}
	return "http://localhost:11434/v1"
}

// opens the config file in the default editor
func OpenConfig() error {
	home, err := homedir.Dir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".config", "ingest", "ingest.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist")
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	return runCommand(editor, configPath)
}

// runs a command in the shell
func runCommand(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
