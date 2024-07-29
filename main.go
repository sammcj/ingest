package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/sammcj/ingest/config"
	"github.com/sammcj/ingest/filesystem"
	"github.com/sammcj/ingest/git"
	"github.com/sammcj/ingest/template"
	"github.com/sammcj/ingest/token"
	"github.com/sammcj/ingest/utils"
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
	report               bool
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

	rootCmd.Flags().StringSliceP("include", "i", nil, "Patterns to include")
	rootCmd.Flags().StringSliceP("exclude", "e", nil, "Patterns to exclude")
	rootCmd.Flags().BoolVar(&includePriority, "include-priority", false, "Include files in case of conflict between include and exclude patterns")
	rootCmd.Flags().BoolVar(&excludeFromTree, "exclude-from-tree", false, "Exclude files/folders from the source tree based on exclude patterns")
	rootCmd.Flags().BoolVar(&tokens, "tokens", true, "Display the token count of the generated prompt")
	rootCmd.Flags().StringVarP(&encoding, "encoding", "c", "", "Optional tokenizer to use for token count")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "Optional output file path")
	rootCmd.Flags().BoolVarP(&diff, "diff", "d", false, "Include git diff")
	rootCmd.Flags().StringVar(&gitDiffBranch, "git-diff-branch", "", "Generate git diff between two branches")
	rootCmd.Flags().StringVar(&gitLogBranch, "git-log-branch", "", "Retrieve git log between two branches")
	rootCmd.Flags().BoolVarP(&lineNumber, "line-number", "l", false, "Add line numbers to the source code")
	rootCmd.Flags().BoolVar(&noCodeblock, "no-codeblock", false, "Disable wrapping code inside markdown code blocks")
	rootCmd.Flags().BoolVar(&relativePaths, "relative-paths", false, "Use relative paths instead of absolute paths, including the parent directory")
	rootCmd.Flags().BoolVarP(&noClipboard, "no-clipboard", "n", false, "Disable copying to clipboard")
	rootCmd.Flags().StringVarP(&templatePath, "template", "t", "", "Optional Path to a custom Handlebars template")
	rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "Print output as JSON")
	rootCmd.Flags().StringVar(&patternExclude, "pattern-exclude", "", "Path to a specific .glob file for exclude patterns")
	rootCmd.Flags().BoolP("ollama", "p", false, "Pass output to Ollama for processing")
	rootCmd.Flags().BoolVar(&printDefaultExcludes, "print-default-excludes", false, "Print the default exclude patterns")
	rootCmd.Flags().BoolVar(&printDefaultTemplate, "print-default-template", false, "Print the default template")
	rootCmd.Flags().BoolP("version", "V", false, "Print the version number")
	rootCmd.Flags().BoolVar(&report, "report", false, "Report the top 5 largest files included in the output")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

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

	useOllama, _ := cmd.Flags().GetBool("ollama")
	// Handle output
	if useOllama {
			if err := handleOllamaOutput(rendered, cfg.Ollama[0]); err != nil {
					return fmt.Errorf("failed to handle Ollama output: %w", err)
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
		utils.PrintColouredMessage("i", fmt.Sprintf("%s Tokens (Approximate)", utils.FormatNumber(tokenCount)), color.FgYellow)
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
				utils.PrintColouredMessage("✓", "Copied to clipboard successfully.", color.FgGreen)
				return nil
			}
			// If clipboard copy failed, fall back to console output
			utils.PrintColouredMessage("!", fmt.Sprintf("Failed to copy to clipboard: %v. Falling back to console output.", err), color.FgYellow)
		}

		if output != "" {
			err := utils.WriteToFile(output, rendered)
			if err != nil {
				return fmt.Errorf("failed to write to file: %w", err)
			}
			utils.PrintColouredMessage("✓", fmt.Sprintf("Written to file: %s", output), color.FgGreen)
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
	starColour := color.New(color.FgYellow).SprintFunc()
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

func handleOllamaOutput(rendered string, ollamaConfig config.OllamaConfig) error {
	if ollamaConfig.PromptPrefix != "" {
		rendered = ollamaConfig.PromptPrefix + "\n" + rendered
	}
	if ollamaConfig.PromptSuffix != "" {
		rendered += ollamaConfig.PromptSuffix
	}

	if ollamaConfig.AutoRun {
		return runOllamaCommand(ollamaConfig.Model, rendered)
	}

	fmt.Println()
	utils.PrintColouredMessage("!", "Enter Ollama prompt:", color.FgRed)
	reader := bufio.NewReader(os.Stdin)
	prompt, _ := reader.ReadString('\n')
	prompt = strings.TrimSpace(prompt)

	return runOllamaCommand(ollamaConfig.Model, rendered+"\n"+prompt)
}

func runOllamaCommand(model, input string) error {
	cmd := exec.Command("ollama", "run", model)
	cmd.Stdin = strings.NewReader(input)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
