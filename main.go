package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
	"github.com/sammcj/ingest/config"
	"github.com/sammcj/ingest/filesystem"
	"github.com/sammcj/ingest/git"
	"github.com/sammcj/ingest/internal/compressor" // Added compressor import
	"github.com/sammcj/ingest/template"
	"github.com/sammcj/ingest/token"
	"github.com/sammcj/ingest/utils"
	"github.com/sammcj/ingest/web"
	"github.com/sammcj/quantest"
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
	vramFlag             bool
	modelIDFlag          string
	quantFlag            string
	contextFlag          int
	kvCacheFlag          string
	memoryFlag           float64
	fitsFlag             float64
	quantTypeFlag        string
	verbose              bool
	noDefaultExcludes    bool
	followSymlinks       bool
	Version              string // This will be set by the linker at build time
	rootCmd              *cobra.Command
	webCrawl             bool
	webMaxDepth          int
	webAllowedDomains    []string
	webTimeout           int
	webConcurrentJobs    int
	compressFlag         bool // Added compress flag
)

type GitData struct {
	Path          string
	GitDiff       string
	GitDiffBranch string
	GitLogBranch  string
}

func init() {
	rootCmd = &cobra.Command{
		Use:   "ingest [flags] [path ...]",
		Short: "Generate a markdown LLM prompt from files and directories",
		Long:  `ingest is a command-line tool to generate an LLM prompt from files and directories.`,
		RunE:  run,
		Args:  cobra.ArbitraryArgs,
	}

	// Define flags
	rootCmd.Flags().Bool("llm", false, "Send output to any OpenAI compatible API for inference")
	rootCmd.Flags().BoolP("version", "V", false, "Print the version number")
	rootCmd.Flags().BoolVar(&excludeFromTree, "exclude-from-tree", false, "Exclude files/folders from the source tree based on exclude patterns")
	rootCmd.Flags().BoolVar(&includePriority, "include-priority", false, "Include files in case of conflict between include and exclude patterns")
	rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "Print output as JSON")
	rootCmd.Flags().BoolVar(&noCodeblock, "no-codeblock", false, "Disable wrapping code inside markdown code blocks")
	rootCmd.Flags().BoolVar(&printDefaultExcludes, "print-default-excludes", false, "Print the default exclude patterns")
	rootCmd.Flags().BoolVar(&printDefaultTemplate, "print-default-template", false, "Print the default template")
	rootCmd.Flags().BoolVar(&relativePaths, "relative-paths", false, "Use relative paths instead of absolute paths, including the parent directory")
	rootCmd.Flags().BoolVar(&report, "report", true, "Report the top 10 largest files included in the output")
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
	rootCmd.Flags().StringVarP(&encoding, "encoding", "c", "o200k", "Tokeniser to use for token count (o200k, cl100k, p50k, r50k)")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "Optional output file path")
	rootCmd.Flags().StringArrayP("prompt", "p", nil, "Prompt suffix to append to the generated content")
	rootCmd.Flags().StringVarP(&templatePath, "template", "t", "", "Optional Path to a custom Handlebars template")
	rootCmd.Flags().BoolP("save", "s", false, "Automatically save the generated markdown to ~/ingest/<dirname>.md")
	rootCmd.Flags().Bool("config", false, "Open the config file in the default editor")
	rootCmd.Flags().BoolVar(&noDefaultExcludes, "no-default-excludes", false, "Disable default exclude patterns")
	rootCmd.Flags().BoolVar(&followSymlinks, "follow-symlinks", false, "Follow symlinked files and directories")
	rootCmd.Flags().BoolVar(&compressFlag, "compress", false, "Enable code compression using Tree-sitter") // Added compress flag

	// Web Crawler flags
	rootCmd.Flags().BoolVar(&webCrawl, "web", false, "Enable web crawling mode")
	rootCmd.Flags().IntVar(&webMaxDepth, "web-depth", 1, "Maximum crawling depth for web pages")
	rootCmd.Flags().StringSliceVar(&webAllowedDomains, "web-domains", nil, "Allowed domains for web crawling")
	rootCmd.Flags().IntVar(&webTimeout, "web-timeout", 120, "Timeout in seconds for web requests")
	rootCmd.Flags().IntVar(&webConcurrentJobs, "web-concurrent", 6, "Number of concurrent crawling jobs")

	// VRAM estimation flags
	rootCmd.Flags().BoolVar(&vramFlag, "vram", false, "Estimate vRAM usage")
	rootCmd.Flags().StringVarP(&modelIDFlag, "model", "m", "", "vRAM Estimation - Model ID")
	rootCmd.Flags().StringVarP(&quantFlag, "quant", "q", "", "vRAM Estimation - Quantization type (e.g., q4_k_m) or bits per weight (e.g., 5.0)")
	rootCmd.Flags().IntVar(&contextFlag, "context", 0, "vRAM Estimation - Context length for vRAM estimation")
	rootCmd.Flags().StringVar(&kvCacheFlag, "kvcache", "fp16", "vRAM Estimation - KV cache quantization: fp16, q8_0, or q4_0")
	rootCmd.Flags().StringVar(&quantTypeFlag, "quanttype", "gguf", "vRAM Estimation - Quantization type: gguf or exl2")
	rootCmd.Flags().Float64Var(&memoryFlag, "memory", 0, "vRAM Estimation - Available memory in GB for context calculation")
	// if --fits is set, set memory to it's value
	rootCmd.Flags().Float64VarP(&fitsFlag, "fits", "f", 0, "(alias for --memory)")
	if err := rootCmd.Flags().SetAnnotation("fits", cobra.BashCompOneRequiredFlag, []string{"--memory"}); err != nil {
		fmt.Printf("Error setting annotation for fits flag: %v\n", err)
	}

	// Add completion command
	rootCmd.AddCommand(&cobra.Command{
		Use:                   "completion [bash|zsh|fish]",
		Short:                 "Generate completion script",
		Long:                  `To load completions: Bash: $ source <(ingest completion bash) Zsh: $ source <(ingest completion zsh) Fish: $ ingest completion`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run:                   runCompletion,
	})
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// If no arguments are provided, use the current directory
	if len(args) == 0 {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		args = []string{currentDir}
	}

	if version, _ := cmd.Flags().GetBool("version"); version {
		fmt.Printf("ingest version %s\n", Version)
		return nil
	}

	if configFlag, _ := cmd.Flags().GetBool("config"); configFlag {
		if err := config.OpenConfig(); err != nil {
			return fmt.Errorf("failed to open config: %w", err)
		}
		os.Exit(0)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := utils.EnsureConfigDirectories(); err != nil {
		return fmt.Errorf("failed to ensure config directories: %w", err)
	}
	// If no arguments are provided, use the current directory
	if len(args) == 0 {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		args = []string{currentDir}
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

	includePatterns, _ := cmd.Flags().GetStringSlice("include")
	excludePatterns, _ := cmd.Flags().GetStringSlice("exclude")

	// Setup template
	tmpl, err := template.SetupTemplate(templatePath)
	if err != nil {
		return fmt.Errorf("failed to set up template: %w", err)
	}

	// Setup progress spinner
	spinner := utils.SetupSpinner("Traversing directory and building tree..")
	defer func() {
		if err := spinner.Finish(); err != nil {
			fmt.Printf("Error finishing spinner: %v\n", err)
		}
	}()

	// If verbose, print active excludes
	if verbose {
		activeExcludes, err := filesystem.ReadExcludePatterns(patternExclude, false)
		if err != nil {
			return fmt.Errorf("failed to read exclude patterns: %w", err)
		}
		printExcludePatterns(activeExcludes)
	}

	// Process all provided paths
	var allFiles []filesystem.FileInfo
	var allTrees []string
	var gitData []GitData
	var allExcluded []*filesystem.ExcludedInfo

	remainingArgs := make([]string, len(args))
	copy(remainingArgs, args)

	for i := range remainingArgs {
		arg := remainingArgs[i]

		// Check if this is a URL (either with --web flag or auto-detected)
		if webCrawl || isURL(arg) {
			if !isURL(arg) {
				return fmt.Errorf("web crawling is enabled but the argument '%s' is not a URL", arg)
			}

			utils.PrintColouredMessage("ℹ️", fmt.Sprintf("Processing URL: %s", arg), color.FgBlue)

			// Process as web URL - now passing excludePatterns
			result, err := processWebInput(arg, excludePatterns)
			if err != nil {
				return fmt.Errorf("failed to process web URL %s: %w", arg, err)
			}

			allFiles = append(allFiles, result.Files...)
			allTrees = append(allTrees, result.TreeString)
			continue
		}

		// Process as local file/directory
		absPath, err := filepath.Abs(arg)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", arg, err)
		}

		fileInfo, err := os.Stat(absPath)
		if err != nil {
			return fmt.Errorf("failed to get file info for %s: %w", arg, err)
		}

		var files []filesystem.FileInfo
		var tree string
		var excluded *filesystem.ExcludedInfo

		// Initialize the compressor if the flag is set
		var comp *compressor.GenericCompressor
		if compressFlag {
			comp = compressor.NewGenericCompressor()
		}

		if fileInfo.IsDir() {
			// Existing directory processing logic
			tree, files, excluded, err = filesystem.WalkDirectory(absPath, includePatterns, excludePatterns, patternExclude, includePriority, lineNumber, relativePaths, excludeFromTree, noCodeblock, noDefaultExcludes, followSymlinks, comp) // Pass compressor
			if err != nil {
				return fmt.Errorf("failed to process directory %s: %w", arg, err)
			}
			tree = fmt.Sprintf("%s:\n%s", absPath, tree)
		} else {
			// New file processing logic
			file, err := filesystem.ProcessSingleFile(absPath, lineNumber, relativePaths, noCodeblock, followSymlinks, comp) // Pass compressor
			if err != nil {
				return fmt.Errorf("failed to process file %s: %w", arg, err)
			}
			files = []filesystem.FileInfo{file}
			tree = fmt.Sprintf("File: %s", absPath)
		}

		allFiles = append(allFiles, files...)
		allTrees = append(allTrees, tree)
		if excluded != nil {
			allExcluded = append(allExcluded, excluded)
		}

		// Handle git operations for each path
		gitDiffContent := ""
		gitDiffBranchContent := ""
		gitLogBranchContent := ""

		if diff {
			gitDiffContent, err = git.GetGitDiff(absPath)
			if err != nil {
				// Log the error but continue processing
				fmt.Printf("Warning: failed to get git diff for %s: %v\n", absPath, err)
			}
		}

		if gitDiffBranch != "" {
			branches := strings.Split(gitDiffBranch, ",")
			if len(branches) == 2 {
				gitDiffBranchContent, err = git.GetGitDiffBetweenBranches(absPath, branches[0], branches[1])
				if err != nil {
					fmt.Printf("Warning: failed to get git diff between branches for %s: %v\n", absPath, err)
				}
			}
		}

		if gitLogBranch != "" {
			branches := strings.Split(gitLogBranch, ",")
			if len(branches) == 2 {
				gitLogBranchContent, err = git.GetGitLog(absPath, branches[0], branches[1])
				if err != nil {
					fmt.Printf("Warning: failed to get git log for %s: %v\n", absPath, err)
				}
			}
		}

		gitData = append(gitData, GitData{
			Path:          absPath,
			GitDiff:       gitDiffContent,
			GitDiffBranch: gitDiffBranchContent,
			GitLogBranch:  gitLogBranchContent,
		})
	}

	// Prepare data for template
	var excludedInfo any
	if len(allExcluded) > 0 {
		excludedInfo = allExcluded[0] // Use the first excluded info if available
	}

	data := map[string]any{
		"source_trees": strings.Join(allTrees, "\n\n"),
		"files":        allFiles,
		"git_data":     gitData,
		"excluded":     excludedInfo,
	}

	if err := spinner.Finish(); err != nil {
		return fmt.Errorf("failed to finish spinner: %w", err)
	}

	// Render template
	rendered, err := template.RenderTemplate(tmpl, data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Check if save is set in config or flag
	autoSave, _ := cmd.Flags().GetBool("save")
	if cfg.AutoSave || autoSave {
		// Pass the output flag value to autoSaveOutput
		if err := autoSaveOutput(rendered, output, args[0]); err != nil { // Assuming args[0] is a representative source path
			utils.PrintColouredMessage("❌", fmt.Sprintf("Error auto-saving file: %v", err), color.FgRed)
		}
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
		// If both --save and --output are used, we don't want handleOutput to write the file
		// as autoSaveOutput will handle it
		outputForHandleOutput := output
		if (cfg.AutoSave || autoSave) && output != "" {
			outputForHandleOutput = ""
		}
		if err := handleOutput(rendered, tokens, encoding, noClipboard, outputForHandleOutput, jsonOutput, report || verbose, allFiles); err != nil {
			return fmt.Errorf("failed to handle output: %w", err)
		}
	}

	// Print all collected messages at the end
	utils.PrintMessages()

	return nil
}

func reportLargestFiles(files []filesystem.FileInfo) {
	sort.Slice(files, func(i, j int) bool {
		return len(files[i].Code) > len(files[j].Code)
	})

	// fmt.Println("Top 15 largest files (by estimated token count):")
	// print this in colour
	utils.PrintColouredMessage("ℹ️", "Top 15 largest files (by estimated token count):", color.FgCyan)
	colourRange := []*color.Color{
		color.New(color.FgRed),
		color.New(color.FgRed),
		color.New(color.FgRed),
		color.New(color.FgRed),
		color.New(color.FgRed),
		color.New(color.FgYellow),
		color.New(color.FgYellow),
		color.New(color.FgYellow),
		color.New(color.FgYellow),
		color.New(color.FgYellow),
		color.New(color.FgGreen),
		color.New(color.FgGreen),
		color.New(color.FgGreen),
		color.New(color.FgGreen),
		color.New(color.FgGreen),
	}

	// print the files
	for i, file := range files {
		tokenCount := token.CountTokens(file.Code, encoding)
		// get the colour
		colour := colourRange[i]
		fmt.Printf("- %d. %s (%s tokens)\n", i+1, file.Path, colour.Sprint(utils.FormatNumber(tokenCount)))
		// break after 14
		if i == 14 {
			break
		}
	}

	fmt.Println()
}

func handleOutput(rendered string, countTokens bool, encoding string, noClipboard bool, output string, jsonOutput bool, report bool, files []filesystem.FileInfo) error {
	if countTokens {
		tokenCount := token.CountTokens(rendered, encoding)
		println()
		utils.AddMessage("ℹ️", fmt.Sprintf("Tokens (Approximate): %v", utils.FormatNumber(tokenCount)), color.FgYellow, 1)
	}

	if report {
		reportLargestFiles(files)
	}

	if jsonOutput {
		jsonData := map[string]any{
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
		outputWritten := false
		if output != "" {
			err := utils.WriteToFile(output, rendered)
			if err != nil {
				// Report the error but continue to potentially copy to clipboard or print
				utils.PrintColouredMessage("❌", fmt.Sprintf("Failed to write to file %s: %v", output, err), color.FgRed)
			} else {
				utils.AddMessage("✅", fmt.Sprintf("Written to file: %s", output), color.FgGreen, 20)
				outputWritten = true
			}
		}

		clipboardCopied := false
		if !noClipboard {
			err := utils.CopyToClipboard(rendered)
			if err == nil {
				utils.AddMessage("✅", "Copied to clipboard successfully.", color.FgGreen, 5)
				clipboardCopied = true
			} else {
				// Only show clipboard error if we didn't write to a file
				if !outputWritten {
					utils.PrintColouredMessage("⚠️", fmt.Sprintf("Failed to copy to clipboard: %v.", err), color.FgYellow)
				}
			}
		}

		// If neither output file was written nor clipboard was copied, print to console
		if !outputWritten && !clipboardCopied {
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
		utils.AddMessage("ℹ️", fmt.Sprintf("Tokens (Approximate): %v", utils.FormatNumber(tokenCount)), color.FgYellow, 40)
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

	termWidth := min(
		// if the term width is over 160, set it to 160
		utils.GetTerminalWidth(), 160)

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
		return fmt.Errorf("model ID is required for vRAM estimation")
	}

	tokenCount := token.CountTokens(content, encoding)

	// TODO: fix this:
	// quant, err := quantest.GetOllamaQuantLevel(modelIDFlag)
	// if err != nil {
	// 	return fmt.Errorf("error getting quantisation level: %w", err)
	// }

	quant := "q4_k_m"

	estimation, err := quantest.EstimateVRAMForModel(modelIDFlag, memoryFlag, tokenCount, quant, kvCacheFlag)
	if err != nil {
		return fmt.Errorf("error estimating vRAM: %w", err)
	}

	utils.AddMessage("ℹ️", fmt.Sprintf("Model: %s", estimation.ModelName), color.FgCyan, 10)
	utils.AddMessage("ℹ️", fmt.Sprintf("Estimated vRAM Required: %.2f GB", estimation.EstimatedVRAM), color.FgCyan, 3)
	// print the vram available
	utils.AddMessage("ℹ️", fmt.Sprintf("Available vRAM: %.2f GB", memoryFlag), color.FgCyan, 10)
	if estimation.FitsAvailable {
		utils.AddMessage("✅", "Fits Available vRAM", color.FgGreen, 2)
	} else {
		utils.AddMessage("❌", "Does Not Fit Available vRAM", color.FgYellow, 2)
	}
	utils.AddMessage("ℹ️", fmt.Sprintf("Max Context Size: %d", estimation.MaxContextSize), color.FgCyan, 8)
	// utils.AddMessage("ℹ️", fmt.Sprintf("Maximum Quantisation: %s", estimation.MaximumQuant), color.FgCyan, 10)
	// TODO: - this isn't that useful, come up with something smarter

	// Generate and print the quant table
	table, err := quantest.GenerateQuantTable(estimation.ModelConfig, memoryFlag)
	if err != nil {
		return fmt.Errorf("error generating quant table: %w", err)
	}
	fmt.Println(quantest.PrintFormattedTable(table))

	// Check if the content fits within the specified constraints
	if memoryFlag > 0 {
		if tokenCount > estimation.MaxContextSize {
			utils.AddMessage("❗️", fmt.Sprintf("Generated content (%d tokens) exceeds maximum context (%d tokens).", tokenCount, estimation.MaxContextSize), color.FgYellow, 2)
		} else {
			utils.AddMessage("✅", fmt.Sprintf("Generated content (%d tokens) fits within maximum context (%d tokens).", tokenCount, estimation.MaxContextSize), color.FgGreen, 2)
		}
	}

	return nil
}

// autoSaveOutput saves the content based on the combination of --save and --output flags:
// - If only --save is used, save to ~/ingest/<dirname>.md
// - If --save and --output ./somefile.md, only save to ./somefile.md
// - If --save and --output somefile.md, save to ~/ingest/somefile.md
func autoSaveOutput(content string, outputPath string, sourcePath string) error {
	var finalPath string

	if outputPath != "" {
		if strings.HasPrefix(outputPath, "./") || strings.HasPrefix(outputPath, "../") || filepath.IsAbs(outputPath) {
			// Case: --output starts with ./ or ../ or is absolute path
			// Save only to the specified output path
			absOutputPath, err := filepath.Abs(outputPath)
			if err != nil {
				return fmt.Errorf("failed to get absolute path for output file %s: %w", outputPath, err)
			}
			finalPath = absOutputPath
		} else {
			// Case: --output is just a filename
			// Save to ~/ingest/ with the specified filename
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get user home directory: %w", err)
			}
			ingestDir := filepath.Join(homeDir, "ingest")
			finalPath = filepath.Join(ingestDir, outputPath)
		}
	} else {
		// Default --save behavior
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		ingestDir := filepath.Join(homeDir, "ingest")
		fileName := filepath.Base(sourcePath) + ".md"
		finalPath = filepath.Join(ingestDir, fileName)
	}

	// Ensure the directory for the final path exists
	finalDir := filepath.Dir(finalPath)
	if err := os.MkdirAll(finalDir, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s for auto-save file: %w", finalDir, err)
	}

	// Write the file using os.WriteFile
	if err := os.WriteFile(finalPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write auto-save file to %s: %w", finalPath, err)
	}

	utils.AddMessage("💾", fmt.Sprintf("Auto-saved to: %s", finalPath), color.FgMagenta, 15) // Changed icon and message slightly
	return nil
}

func runCompletion(cmd *cobra.Command, args []string) {
	switch args[0] {
	case "bash":
		if err := cmd.Root().GenBashCompletion(os.Stdout); err != nil {
			fmt.Printf("Error generating bash completion: %v\n", err)
		}
	case "zsh":
		if err := cmd.Root().GenZshCompletion(os.Stdout); err != nil {
			fmt.Printf("Error generating zsh completion: %v\n", err)
		}
	case "fish":
		if err := cmd.Root().GenFishCompletion(os.Stdout, true); err != nil {
			fmt.Printf("Error generating fish completion: %v\n", err)
		}
	}
}

func processWebInput(urlStr string, excludePatterns []string) (*web.CrawlResult, error) {
	options := web.CrawlOptions{
		MaxDepth:       webMaxDepth,
		AllowedDomains: webAllowedDomains,
		Timeout:        webTimeout,
		ConcurrentJobs: webConcurrentJobs,
	}

	return web.ProcessWebURL(urlStr, options, excludePatterns)
}

func isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
