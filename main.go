package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
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
	Version              string // This will be set by the linker at build time
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ingest [path]",
		Short: "Generate a markdown LLM prompt from a codebase directory",
		Long:  `ingest is a command-line tool to generate an LLM prompt from a codebase directory.`,
		RunE:  run,
	}

	// if run with no arguments, assume the current directory
	if len(os.Args) == 1 {
		os.Args = append(os.Args, ".")
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
	rootCmd.Flags().BoolVar(&printDefaultExcludes, "print-default-excludes", false, "Print the default exclude patterns")
	rootCmd.Flags().BoolVar(&printDefaultTemplate, "print-default-template", false, "Print the default template")
	rootCmd.Flags().BoolP("version", "v", false, "Print the version number")
	rootCmd.Flags().BoolVar(&report, "report", false, "Report the top 5 largest files included in the output")

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

	if err := utils.EnsureConfigDirectories(); err != nil {
		return fmt.Errorf("failed to ensure config directories: %w", err)
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

	// Traverse directory with parallel processing
	tree, files, err := filesystem.WalkDirectory(absPath, includePatterns, excludePatterns, patternExclude, includePriority, lineNumber, relativePaths, excludeFromTree, noCodeblock)
	if err != nil {
		spinner.Finish()
		return fmt.Errorf("failed to build directory tree: %w", err)
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
		"absolute_code_path": absPath,
		"source_tree":        tree,
		"files":              files,
		"git_diff":           gitDiff,
		"git_diff_branch":    gitDiffBranchContent,
		"git_log_branch":     gitLogBranchContent,
	}

	// Render template
	rendered, err := template.RenderTemplate(tmpl, data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Handle output
	if err := handleOutput(rendered, tokens, encoding, noClipboard, output, jsonOutput, report, files); err != nil {
		return fmt.Errorf("failed to handle output: %w", err)
	}

	return nil
}

func reportLargestFiles(files []filesystem.FileInfo) {
	sort.Slice(files, func(i, j int) bool {
		return len(files[i].Code) > len(files[j].Code)
	})

	fmt.Println("\nTop 5 largest files:")
	for i, file := range files[:min(5, len(files))] {
		fmt.Printf("%d. %s (%d bytes)\n", i+1, file.Path, len(file.Code))
	}
}

func handleOutput(rendered string, countTokens bool, encoding string, noClipboard bool, output string, jsonOutput bool, report bool, files []filesystem.FileInfo) error {
	if countTokens {
		tokenCount := token.CountTokens(rendered, encoding)
		utils.PrintColouredMessage("i", fmt.Sprintf("%d Tokens (Approximate)", tokenCount), color.FgYellow)
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
