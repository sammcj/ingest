package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
	"github.com/sammcj/gollama/vramestimator"
	"github.com/sammcj/ingest/config"
	"github.com/sammcj/ingest/filesystem"
	"github.com/sammcj/ingest/git"
	"github.com/sammcj/ingest/template"
	"github.com/sammcj/ingest/token"
	"github.com/sammcj/ingest/utils"
	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

var (
	includePriority      bool
	excludeFromTree      bool
	tokens               bool
	encoding             string
	output               string
	diff                 bool
	gitDiffBranch        string
	gitLogBranch         string
	lineNumber           bool
	noCodeblock          bool
	relativePaths        bool
	noClipboard          bool
	templatePath         string
	jsonOutput           bool
	patternExclude       string
	printDefaultExcludes bool
	printDefaultTemplate bool
	promptPrefix         string
	promptSuffix         string
	report               bool
	vramFlag      bool
	modelIDFlag   string
	quantFlag     string
	contextFlag   int
	kvCacheFlag   string
	memoryFlag    float64
	quantTypeFlag string
	verbose              bool
	Version              string // This will be set by the linker at build time
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ingest [flags] [paths...]",
		Short: "Generate a markdown LLM prompt from files and directories",
		Long:  `ingest is a command-line tool to generate an LLM prompt from files and directories.`,
		RunE:  run,
	}

	rootCmd.Flags().Bool("llm", false, "Send output to any OpenAI compatible API for inference")
	rootCmd.Flags().BoolP("version", "V", false, "Print the version number")
	rootCmd.Flags().BoolVar(&excludeFromTree, "exclude-from-tree", false, "Exclude files/folders from the source tree based on exclude patterns")
	rootCmd.Flags().BoolVar(&includePriority, "include-priority", false, "Include files in case of conflict between include and exclude patterns")
	rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "Print output as JSON")
	rootCmd.Flags().BoolVar(&noCodeblock, "no-codeblock", false, "Disable wrapping code inside markdown code blocks")
	rootCmd.Flags().BoolVar(&printDefaultExcludes, "print-default-excludes", false, "Print the default exclude patterns")
	rootCmd.Flags().BoolVar(&printDefaultTemplate, "print-default-template", false, "Print the default template")
	rootCmd.Flags().BoolVar(&relativePaths, "relative-paths", false, "Use relative paths instead of absolute paths, including the parent directory")
	rootCmd.Flags().BoolVar(&report, "report", true, "Report the top 5 largest files included in the output")
	rootCmd.Flags().BoolVar(&tokens, "tokens", true, "Display the token count of the generated prompt")
	rootCmd.Flags().BoolVarP(&diff, "diff", "d", false, "Include git diff")
	rootCmd.Flags().BoolVarP(&lineNumber, "line-number", "l", false, "Add line numbers to the source code")
	rootCmd.Flags().BoolVarP(&noClipboard, "no-clipboard", "n", false, "Disable copying to clipboard")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().StringSliceP("exclude", "e", nil, "Patterns to exclude")
	rootCmd.Flags().StringSliceP("include", "i", nil, "Patterns to include")
	rootCmd.Flags().StringVar(&gitDiffBranch, "git-diff-branch", "", "Generate git diff between two branches")
	rootCmd.Flags().StringVar(&gitLogBranch, "git-log-branch", "", "Retrieve git log between two branches")
	rootCmd.Flags().StringVar(&patternExclude, "pattern-exclude", "", "Path to a specific .glob file for exclude patterns")
	rootCmd.Flags().StringVarP(&encoding, "encoding", "c", "", "Optional tokenizer to use for token count")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "Optional output file path")
	rootCmd.Flags().StringArrayP("prompt", "p", nil, "Prompt suffix to append to the generated content")
	rootCmd.Flags().StringVarP(&templatePath, "template", "t", "", "Optional Path to a custom Handlebars template")

	// VRAM estimation flags
	rootCmd.Flags().BoolVar(&vramFlag, "vram", false, "Estimate vRAM usage")
	rootCmd.Flags().StringVar(&modelIDFlag, "model", "", "vRAM Estimation - Model ID")
	rootCmd.Flags().StringVar(&quantFlag, "quant", "", "vRAM Estimation - Quantization type (e.g., q4_k_m) or bits per weight (e.g., 5.0)")
	rootCmd.Flags().IntVar(&contextFlag, "context", 0, "vRAM Estimation - Context length for vRAM estimation")
	rootCmd.Flags().StringVar(&kvCacheFlag, "kvcache", "fp16", "vRAM Estimation - KV cache quantization: fp16, q8_0, or q4_0")
	rootCmd.Flags().Float64Var(&memoryFlag, "memory", 0, "vRAM Estimation - Available memory in GB for context calculation")
	rootCmd.Flags().StringVar(&quantTypeFlag, "quanttype", "gguf", "vRAM Estimation - Quantization type: gguf or exl2")


	rootCmd.ParseFlags(os.Args[1:])

	// if run with no arguments, assume the current directory
	if len(rootCmd.Flags().Args()) == 0 {
		rootCmd.SetArgs([]string{"."})
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if version, _ := cmd.Flags().GetBool("version"); version {
		fmt.Printf("ingest version %s\n", Version)
		return nil
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := utils.EnsureConfigDirectories(); err != nil {
		return fmt.Errorf("failed to ensure config directories: %w", err)
	}

	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Handle the prompt flag
	promptArray, _ := cmd.Flags().GetStringArray("prompt")
	promptSuffix = strings.Join(promptArray, " ")

	if printDefaultExcludes {
		filesystem.PrintDefaultExcludes()
		return nil
	}

	if printDefaultTemplate {
		template.PrintDefaultTemplate()
		return nil
	}

	path := args[0]
	includePatterns, _ := cmd.Flags().GetStringSlice("include")
	excludePatterns, _ := cmd.Flags().GetStringSlice("exclude")

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Setup template
	tmpl, err := template.SetupTemplate(templatePath)
	if err != nil {
		return fmt.Errorf("failed to set up template: %w", err)
	}

	// Setup progress spinner
	spinner := utils.SetupSpinner("Traversing directory and building tree...")
	defer spinner.Finish()

	// If verbose, print active excludes
	if verbose {
		activeExcludes, err := filesystem.ReadExcludePatterns(patternExclude)
		if err != nil {
			return fmt.Errorf("failed to read exclude patterns: %w", err)
		}
		printExcludePatterns(activeExcludes)
	}

	// Process all provided paths
	var allFiles []filesystem.FileInfo
	var allTrees []string

	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", path, err)
		}

		// Traverse directory with parallel processing
		tree, files, err := filesystem.WalkDirectory(absPath, includePatterns, excludePatterns, patternExclude, includePriority, lineNumber, relativePaths, excludeFromTree, noCodeblock)
		if err != nil {
			return fmt.Errorf("failed to process %s: %w", path, err)
		}

		allFiles = append(allFiles, files...)
		allTrees = append(allTrees, fmt.Sprintf("%s:\n%s", absPath, tree))
	}

	// Handle git operations
	gitDiff := ""
	if diff {
		spinner.Describe("Generating git diff...")
		gitDiff, err = git.GetGitDiff(absPath)
		if err != nil {
			return fmt.Errorf("failed to get git diff: %w", err)
		}
	}

	gitDiffBranchContent := ""
	if gitDiffBranch != "" {
		spinner.Describe("Generating git diff between two branches...")
		branches := strings.Split(gitDiffBranch, ",")
		if len(branches) != 2 {
			spinner.Finish()
			return fmt.Errorf("please provide exactly two branches separated by a comma")
		}
		gitDiffBranchContent, err = git.GetGitDiffBetweenBranches(absPath, branches[0], branches[1])
		if err != nil {
			return fmt.Errorf("failed to get git diff between branches: %w", err)
		}
	}

	gitLogBranchContent := ""
	if gitLogBranch != "" {
		spinner.Describe("Generating git log between two branches...")
		branches := strings.Split(gitLogBranch, ",")
		if len(branches) != 2 {
			spinner.Finish()
			return fmt.Errorf("please provide exactly two branches separated by a comma")
		}
		gitLogBranchContent, err = git.GetGitLog(absPath, branches[0], branches[1])
		if err != nil {
			return fmt.Errorf("failed to get git log: %w", err)
		}
	}

	spinner.Finish()

	// Prepare data for template
	data := map[string]interface{}{
		"source_trees":       strings.Join(allTrees, "\n\n"),
		"files":              allFiles,
		"absolute_code_path": absPath,
		"git_diff":           gitDiff,
		"git_diff_branch":    gitDiffBranchContent,
		"git_log_branch":     gitLogBranchContent,
	}

	// Render template

	rendered, err := template.RenderTemplate(tmpl, data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// VRAM estimation
	if vramFlag {
		fmt.Println()
		if err := performVRAMEstimation(rendered); err != nil {
			utils.PrintColouredMessage("❌", fmt.Sprintf("VRAM estimation error: %v", err), color.FgRed)
		}
	}
	useLLM, _ := cmd.Flags().GetBool("llm")

	// Handle output
	if useLLM {
		if err := handleLLMOutput(rendered, cfg.LLM, tokens, encoding); err != nil {
			utils.PrintColouredMessage("❌", fmt.Sprintf("LLM output error: %v", err), color.FgRed)
		}
	} else {
		if err := handleOutput(rendered, tokens, encoding, noClipboard, output, jsonOutput, report || verbose, allFiles); err != nil {
			return fmt.Errorf("failed to handle output: %w", err)
		}
	}

	return nil
}

func reportLargestFiles(files []filesystem.FileInfo) {
	sort.Slice(files, func(i, j int) bool {
		return len(files[i].Code) > len(files[j].Code)
	})

	fmt.Println("\nTop 5 largest files (by estimated token count):")
	for i, file := range files[:min(5, len(files))] {
		tokenCount := token.CountTokens(file.Code, encoding)
		fmt.Printf("%d. %s (%s tokens)\n", i+1, file.Path, utils.FormatNumber(tokenCount))
	}
}

func handleOutput(rendered string, countTokens bool, encoding string, noClipboard bool, output string, jsonOutput bool, report bool, files []filesystem.FileInfo) error {
	if countTokens {
		tokenCount := token.CountTokens(rendered, encoding)
		utils.PrintColouredMessage("ℹ️", fmt.Sprintf("Tokens (Approximate): %v", utils.FormatNumber(tokenCount)), color.FgYellow)
	}

	if report {
		reportLargestFiles(files)
	}

	if jsonOutput {
		jsonData := map[string]interface{}{
			"prompt":      rendered,
			"token_count": token.CountTokens(rendered, encoding),
			"model_info":  token.GetModelInfo(encoding),
		}
		jsonBytes, err := json.MarshalIndent(jsonData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		if !noClipboard {
			err := utils.CopyToClipboard(rendered)
			if err == nil {
				utils.PrintColouredMessage("✅", "Copied to clipboard successfully.", color.FgGreen)
				return nil
			}
			// If clipboard copy failed, fall back to console output
			utils.PrintColouredMessage("i", fmt.Sprintf("Failed to copy to clipboard: %v. Falling back to console output.", err), color.FgYellow)
		}

		if output != "" {
			err := utils.WriteToFile(output, rendered)
			if err != nil {
				return fmt.Errorf("failed to write to file: %w", err)
			}
			utils.PrintColouredMessage("✅", fmt.Sprintf("Written to file: %s", output), color.FgGreen)
		} else {
			// If no output file is specified, print to console
			fmt.Print(rendered)
		}
	}

	return nil
}

func printExcludePatterns(patterns []string) {
	utils.PrintColouredMessage("i", "Active exclude patterns:", color.FgCyan)

	// Define colours for syntax highlighting
	starColour := color.New(color.FgHiGreen).SprintFunc()
	slashColour := color.New(color.FgGreen).SprintFunc()
	dotColour := color.New(color.FgBlue).SprintFunc()

	// Calculate the maximum width of patterns for alignment
	maxWidth := 0
	for _, pattern := range patterns {
		if len(pattern) > maxWidth {
			maxWidth = len(pattern)
		}
	}

	// Print patterns in a horizontal list
	lineWidth := 0

	// get the width of the terminal
	w := utils.GetTerminalWidth()

	for i, pattern := range patterns {
		highlighted := pattern
		highlighted = strings.ReplaceAll(highlighted, "*", starColour("*"))
		highlighted = strings.ReplaceAll(highlighted, "/", slashColour("/"))
		highlighted = strings.ReplaceAll(highlighted, ".", dotColour("."))

		// Add padding to align patterns
		padding := strings.Repeat(" ", maxWidth-len(pattern)+2)

		if lineWidth+len(pattern)+2 > w && i > 0 {
			fmt.Println()
			lineWidth = 0
		}

		if lineWidth == 0 {
			fmt.Print("  ")
		}

		fmt.Print(highlighted + padding)
		lineWidth += len(pattern) + len(padding)

		if i < len(patterns)-1 {
			fmt.Print("| ")
			lineWidth += 2
		}
	}
}

func handleLLMOutput(rendered string, llmConfig config.LLMConfig, countTokens bool, encoding string) error {
	if countTokens {
		tokenCount := token.CountTokens(rendered, encoding)
		utils.PrintColouredMessage("ℹ️", fmt.Sprintf("Tokens (Approximate): %v", utils.FormatNumber(tokenCount)), color.FgYellow)
	}

	if promptPrefix != "" {
		rendered = promptPrefix + "\n" + rendered
	}

	if promptSuffix != "" {
		rendered += "\n" + promptSuffix
	}

	if llmConfig.AuthToken == "" {
		return fmt.Errorf("LLM auth token is empty")
	}

	clientConfig := openai.DefaultConfig(llmConfig.AuthToken)
	clientConfig.BaseURL = llmConfig.BaseURL
	clientConfig.APIType = openai.APIType(llmConfig.APIType)

	c := openai.NewClientWithConfig(clientConfig)
	ctx := context.Background()

	req := openai.CompletionRequest{
		Model:     llmConfig.Model,
		MaxTokens: llmConfig.MaxTokens,
		Prompt:    rendered,
		Stream:    true,
	}

	if llmConfig.Temperature != nil {
		req.Temperature = *llmConfig.Temperature
	}
	if llmConfig.TopP != nil {
		req.TopP = *llmConfig.TopP
	}
	if llmConfig.PresencePenalty != nil {
		req.PresencePenalty = *llmConfig.PresencePenalty
	}
	if llmConfig.FrequencyPenalty != nil {
		req.FrequencyPenalty = *llmConfig.FrequencyPenalty
	}

	stream, err := c.CreateCompletionStream(ctx, req)
	if err != nil {
		return fmt.Errorf("LLM CompletionStream error: %w", err)
	}
	defer stream.Close()

	termWidth := utils.GetTerminalWidth()
	// if the term width is over 160, set it to 160
	if termWidth > 160 {
		termWidth = 160
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dracula"),
		glamour.WithWordWrap(termWidth-10),
	)
	if err != nil {
		return fmt.Errorf("failed to create renderer: %w", err)
	}

	var buffer strings.Builder
	var output strings.Builder
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		buffer.WriteString(response.Choices[0].Text)

		// Process complete lines
		for {
			line, rest, found := strings.Cut(buffer.String(), "\n")
			if !found {
				break
			}
			output.WriteString(line + "\n")
			buffer.Reset()
			buffer.WriteString(rest)

			// Render and print if we have a complete code block or enough non-code content
			//  if there are any headers, render the markdown chuck before each header
			isHeading := regexp.MustCompile(`^#+`).MatchString(output.String())
			if isHeading && output.Len() > 0 && !strings.Contains(output.String(), "```") &&
				(strings.HasSuffix(output.String(), "```\n") || output.Len() > 200) {
				contentToRender := strings.TrimSpace(output.String()) + "\n"
				renderedContent, err := r.Render(contentToRender)
				if err != nil {
					return fmt.Errorf("rendering error: %w", err)
				}
				fmt.Print(renderedContent)
				output.Reset()
			} else {
				if !strings.Contains(output.String(), "```") &&
					(strings.HasSuffix(output.String(), "```\n") || output.Len() > 200) {
					// Trim excess newlines before rendering
					contentToRender := strings.TrimSpace(output.String()) + "\n"
					renderedContent, err := r.Render(contentToRender)
					if err != nil {
						return fmt.Errorf("rendering error: %w", err)
					}
					fmt.Print(renderedContent)
					output.Reset()
				}
			}
		}
	}

	// Render and print any remaining content
	if buffer.Len() > 0 {
		output.WriteString(buffer.String())
	}
	if output.Len() > 0 {
		renderedContent, err := r.Render(output.String())
		if err != nil {
			return fmt.Errorf("rendering error: %w", err)
		}
		fmt.Print(renderedContent)
	}

	return nil
}



func performVRAMEstimation(content string) error {
	if modelIDFlag == "" {
		return fmt.Errorf("model ID is required for VRAM estimation")
	}

	var kvCacheQuant vramestimator.KVCacheQuantisation
	switch kvCacheFlag {
	case "fp16":
		kvCacheQuant = vramestimator.KVCacheFP16
	case "q8_0":
		kvCacheQuant = vramestimator.KVCacheQ8_0
	case "q4_0":
		kvCacheQuant = vramestimator.KVCacheQ4_0
	default:
		utils.PrintColouredMessage("❌", fmt.Sprintf("Invalid KV cache quantization: %s. Using default fp16.", kvCacheFlag), color.FgYellow)
		kvCacheQuant = vramestimator.KVCacheFP16
	}

	tokenCount := token.CountTokens(content, encoding)

	if memoryFlag > 0 && contextFlag == 0 && quantFlag == "" {
		// Calculate best BPW
		bestBPW, err := vramestimator.CalculateBPW(modelIDFlag, memoryFlag, 0, kvCacheQuant, quantTypeFlag, "")
		if err != nil {
				utils.PrintColouredMessage("❌", fmt.Sprintf("Error calculating BPW: %v", err), color.FgYellow)
		}
		// Check if bestBPW is a string
		if bpwStr, ok := bestBPW.(string); ok {
				utils.PrintColouredMessage("ℹ️", fmt.Sprintf("Best BPW for %.2f GB of memory: %s", memoryFlag, bpwStr), color.FgGreen)
		} else if bpwFloat, ok := bestBPW.(float64); ok {
				utils.PrintColouredMessage("ℹ️", fmt.Sprintf("Best BPW for %.2f GB of memory: %.2f", memoryFlag, bpwFloat), color.FgGreen)
		} else {
				utils.PrintColouredMessage("❌", fmt.Sprintf("Unexpected type for BPW: %T", bestBPW), color.FgYellow)
		}

	} else {
		// Parse the quant flag for other operations
		bpw, err := vramestimator.ParseBPWOrQuant(quantFlag)
		if err != nil {
			utils.PrintColouredMessage("❌", fmt.Sprintf("Error parsing quantization: %v", err), color.FgYellow)
		}

		if memoryFlag > 0 && contextFlag == 0 {
			// Calculate maximum context
			maxContext, err := vramestimator.CalculateContext(modelIDFlag, memoryFlag, bpw, kvCacheQuant, "")
			if err != nil {
				return fmt.Errorf("error calculating context: %w", err)
			}
			utils.PrintColouredMessage("ℹ️", fmt.Sprintf("Maximum context for %.2f GB of memory: %d", memoryFlag, maxContext), color.FgGreen)
			if tokenCount > maxContext {
				utils.PrintColouredMessage("❗️", fmt.Sprintf("Generated content (%d tokens) exceeds maximum context.", tokenCount), color.FgYellow)
			} else {
				utils.PrintColouredMessage("✅", fmt.Sprintf("Generated content (%d tokens) fits within maximum context.", tokenCount), color.FgGreen)
			}
		} else if contextFlag > 0 {
			// Calculate VRAM usage
			vram, err := vramestimator.CalculateVRAM(modelIDFlag, bpw, contextFlag, kvCacheQuant, "")
			if err != nil {
				utils.PrintColouredMessage("❌", fmt.Sprintf("Error calculating VRAM: %v", err), color.FgYellow)
			}
			utils.PrintColouredMessage("ℹ️", fmt.Sprintf("Estimated VRAM usage: %.2f GB", vram), color.FgGreen)
			if memoryFlag > 0 {
				if vram > memoryFlag {
					utils.PrintColouredMessage("❗️", fmt.Sprintf("Estimated VRAM usage (%.2f GB) exceeds available memory (%.2f GB).", vram, memoryFlag), color.FgYellow)
				} else {
					utils.PrintColouredMessage("✅", fmt.Sprintf("Estimated VRAM usage (%.2f GB) fits within available memory (%.2f GB).", vram, memoryFlag), color.FgGreen)
				}
			}
		} else {
			return fmt.Errorf("invalid combination of flags. Please specify either --memory, --context, or both")
		}
	}

	return nil
}
