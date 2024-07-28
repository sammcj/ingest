package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/sammcj/ingest/filesystem"
	"github.com/sammcj/ingest/git"
	"github.com/sammcj/ingest/template"
	"github.com/sammcj/ingest/token"
	"github.com/sammcj/ingest/utils"
)

var (
	includePriority  bool
	excludeFromTree  bool
	tokens           bool
	encoding         string
	output           string
	diff             bool
	gitDiffBranch    string
	gitLogBranch     string
	lineNumber       bool
	noCodeblock      bool
	relativePaths    bool
	noClipboard      bool
	templatePath     string
	jsonOutput       bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "code2prompt [path]",
		Short: "Generate an LLM prompt from a codebase directory",
		Long:  `code2prompt is a command-line tool to generate an LLM prompt from a codebase directory.`,
		Args:  cobra.ExactArgs(1),
		Run:   run,
	}

	rootCmd.Flags().StringSliceP("include", "i", nil, "Patterns to include")
	rootCmd.Flags().StringSliceP("exclude", "e", nil, "Patterns to exclude")
	rootCmd.Flags().BoolVar(&includePriority, "include-priority", false, "Include files in case of conflict between include and exclude patterns")
	rootCmd.Flags().BoolVar(&excludeFromTree, "exclude-from-tree", false, "Exclude files/folders from the source tree based on exclude patterns")
	rootCmd.Flags().BoolVar(&tokens, "tokens", false, "Display the token count of the generated prompt")
	rootCmd.Flags().StringVarP(&encoding, "encoding", "c", "", "Optional tokenizer to use for token count")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "Optional output file path")
	rootCmd.Flags().BoolVarP(&diff, "diff", "d", false, "Include git diff")
	rootCmd.Flags().StringVar(&gitDiffBranch, "git-diff-branch", "", "Generate git diff between two branches")
	rootCmd.Flags().StringVar(&gitLogBranch, "git-log-branch", "", "Retrieve git log between two branches")
	rootCmd.Flags().BoolVarP(&lineNumber, "line-number", "l", false, "Add line numbers to the source code")
	rootCmd.Flags().BoolVar(&noCodeblock, "no-codeblock", false, "Disable wrapping code inside markdown code blocks")
	rootCmd.Flags().BoolVar(&relativePaths, "relative-paths", false, "Use relative paths instead of absolute paths, including the parent directory")
	rootCmd.Flags().BoolVar(&noClipboard, "no-clipboard", false, "Disable copying to clipboard")
	rootCmd.Flags().StringVarP(&templatePath, "template", "t", "", "Optional Path to a custom Handlebars template")
	rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "Print output as JSON")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	path := args[0]
	includePatterns, _ := cmd.Flags().GetStringSlice("include")
	excludePatterns, _ := cmd.Flags().GetStringSlice("exclude")

	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	// Setup template
	templateContent, templateName := template.GetTemplate(templatePath)
	hbs := template.SetupHandlebars(templateContent, templateName)

	// Setup progress spinner
	spinner := utils.SetupSpinner("Traversing directory and building tree...")
	defer spinner.Finish()

	// Traverse directory
	tree, files, err := filesystem.TraverseDirectory(absPath, includePatterns, excludePatterns, includePriority, lineNumber, relativePaths, excludeFromTree, noCodeblock)
	if err != nil {
		log.Fatalf("Failed to build directory tree: %v", err)
	}

	// Handle git operations
	gitDiff := ""
	if diff {
		spinner.Describe("Generating git diff...")
		gitDiff = git.GetGitDiff(absPath)
	}

	gitDiffBranchContent := ""
	if gitDiffBranch != "" {
		spinner.Describe("Generating git diff between two branches...")
		branches := strings.Split(gitDiffBranch, ",")
		if len(branches) != 2 {
			log.Fatal("Please provide exactly two branches separated by a comma.")
		}
		gitDiffBranchContent = git.GetGitDiffBetweenBranches(absPath, branches[0], branches[1])
	}

	gitLogBranchContent := ""
	if gitLogBranch != "" {
		spinner.Describe("Generating git log between two branches...")
		branches := strings.Split(gitLogBranch, ",")
		if len(branches) != 2 {
			log.Fatal("Please provide exactly two branches separated by a comma.")
		}
		gitLogBranchContent = git.GetGitLog(absPath, branches[0], branches[1])
	}

	// Prepare data for template
	data := map[string]interface{}{
		"absolute_code_path": utils.Label(absPath),
		"source_tree":        tree,
		"files":              files,
		"git_diff":           gitDiff,
		"git_diff_branch":    gitDiffBranchContent,
		"git_log_branch":     gitLogBranchContent,
	}

	// Handle undefined variables
	template.HandleUndefinedVariables(&data, templateContent)

	// Render template
	rendered := template.RenderTemplate(hbs, data)

	// Handle output
	handleOutput(rendered, tokens, encoding, noClipboard, output, jsonOutput)
}

func handleOutput(rendered string, countTokens bool, encoding string, noClipboard bool, output string, jsonOutput bool) {
	if countTokens {
		tokenCount := token.CountTokens(rendered, encoding)
		modelInfo := token.GetModelInfo(encoding)
		utils.PrintColoredMessage("i", fmt.Sprintf("Token count: %d, Model info: %s", tokenCount, modelInfo), color.FgYellow)
	}

	if jsonOutput {
		jsonData := map[string]interface{}{
			"prompt": rendered,
			"token_count": token.CountTokens(rendered, encoding),
			"model_info": token.GetModelInfo(encoding),
		}
		jsonBytes, _ := json.MarshalIndent(jsonData, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		if !noClipboard {
			err := utils.CopyToClipboard(rendered)
			if err != nil {
				utils.PrintColoredMessage("!", fmt.Sprintf("Failed to copy to clipboard: %v", err), color.FgRed)
			} else {
				utils.PrintColoredMessage("✓", "Copied to clipboard successfully.", color.FgGreen)
			}
		}

		if output != "" {
			err := utils.WriteToFile(output, rendered)
			if err != nil {
				utils.PrintColoredMessage("!", fmt.Sprintf("Failed to write to file: %v", err), color.FgRed)
			}
		}

		if !jsonOutput && output == "" {
			fmt.Println(rendered)
		}
	}
}
