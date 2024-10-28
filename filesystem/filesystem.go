package filesystem

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/sammcj/ingest/pdf"
	"github.com/sammcj/ingest/utils"
)

type FileInfo struct {
	Path      string `json:"path"`
	Extension string `json:"extension"`
	Code      string `json:"code"`
}

type treeNode struct {
	name     string
	children []*treeNode
	isDir    bool
}

func ReadExcludePatterns(patternExclude string, noDefaultExcludes bool) ([]string, error) {
	var patterns []string

	// If a specific pattern exclude file is provided, use it
	if patternExclude != "" {
		return readGlobFile(patternExclude)
	}

	if !noDefaultExcludes {
		// Get the default excludes
		defaultPatterns, err := GetDefaultExcludes()
		if err != nil {
			return nil, fmt.Errorf("failed to read default exclude patterns: %w", err)
		}
		patterns = defaultPatterns
	}

	// Check for user-specific patterns
	home, err := homedir.Dir()
	if err == nil {
		userPatternsDir := filepath.Join(home, ".config", "ingest", "patterns", "exclude")
		userDefaultGlob := filepath.Join(userPatternsDir, "default.glob")

		// If user has a default.glob, it overrides the default patterns
		if _, err := os.Stat(userDefaultGlob); err == nil {
			return readGlobFile(userDefaultGlob)
		}

		// Read other user-defined patterns
		userPatterns, _ := readGlobFilesFromDir(userPatternsDir)

		// Combine user patterns with default patterns (if not disabled)
		patterns = append(patterns, userPatterns...)
	}

	return patterns, nil
}

func readGlobFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return patterns, nil
}

func readGlobFilesFromDir(dir string) ([]string, error) {
	var patterns []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".glob") {
			filePatterns, err := readGlobFile(path)
			if err != nil {
				return err
			}
			patterns = append(patterns, filePatterns...)
		}
		return nil
	})
	return patterns, err
}

func WalkDirectory(rootPath string, includePatterns, excludePatterns []string, patternExclude string, includePriority, lineNumber, relativePaths, excludeFromTree, noCodeblock, noDefaultExcludes bool) (string, []FileInfo, error) {
	var files []FileInfo
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Read exclude patterns
	defaultExcludes, err := ReadExcludePatterns(patternExclude, noDefaultExcludes)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read exclude patterns: %w", err)
	}

	// Combine user-provided exclude patterns with default excludes (if not disabled)
	allExcludePatterns := append(excludePatterns, defaultExcludes...)

	// Always exclude .git directories
	allExcludePatterns = append(allExcludePatterns, "**/.git/**")

	// Read .gitignore if it exists
	gitignore, err := readGitignore(rootPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read .gitignore: %w", err)
	}

	// Check if rootPath is a file or directory
	fileInfo, err := os.Stat(rootPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get file info: %w", err)
	}

	var treeString string

	if !fileInfo.IsDir() {
		// Handle single file
		relPath := filepath.Base(rootPath)
		if shouldIncludeFile(relPath, includePatterns, allExcludePatterns, gitignore, includePriority) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				processFile(rootPath, relPath, filepath.Dir(rootPath), lineNumber, relativePaths, noCodeblock, &mu, &files)
			}()
		}
		treeString = fmt.Sprintf("File: %s", rootPath)
	} else {
		// Generate the tree representation for directory
		treeString, err = generateTreeString(rootPath, allExcludePatterns)
		if err != nil {
			return "", nil, fmt.Errorf("failed to generate directory tree: %w", err)
		}

		// Process files in directory
		err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(rootPath, path)
			if err != nil {
				return err
			}

			// Check if the current path (file or directory) should be excluded
			if shouldExcludePath(relPath, allExcludePatterns, gitignore) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if !info.IsDir() && shouldIncludeFile(relPath, includePatterns, allExcludePatterns, gitignore, includePriority) {
				wg.Add(1)
				go func(path, relPath string, info os.FileInfo) {
					defer wg.Done()
					processFile(path, relPath, rootPath, lineNumber, relativePaths, noCodeblock, &mu, &files)
				}(path, relPath, info)
			}

			return nil
		})
	}

	wg.Wait()

	if err != nil {
		return "", nil, err
	}

	return treeString, files, nil
}

// New helper function to check if a path should be excluded
func shouldExcludePath(path string, excludePatterns []string, gitignore *ignore.GitIgnore) bool {
	for _, pattern := range excludePatterns {
		if match, _ := doublestar.Match(pattern, path); match {
			return true
		}
	}
	return gitignore != nil && gitignore.MatchesPath(path)
}

func shouldIncludeFile(path string, includePatterns, excludePatterns []string, gitignore *ignore.GitIgnore, includePriority bool) bool {
	// Check if the file is explicitly included
	included := len(includePatterns) == 0 || matchesAny(path, includePatterns)

	// Check if the file is explicitly excluded
	excluded := isExcluded(path, excludePatterns) || (gitignore != nil && gitignore.MatchesPath(path))

	if included && excluded {
		return includePriority
	}
	return included && !excluded
}

func matchesAny(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if match, _ := doublestar.Match(pattern, path); match {
			return true
		}
	}
	return false
}

func readGitignore(rootPath string) (*ignore.GitIgnore, error) {
	gitignorePath := filepath.Join(rootPath, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		return nil, nil
	}

	return ignore.CompileIgnoreFile(gitignorePath)
}

func addLineNumbers(code string) string {
	lines := strings.Split(code, "\n")
	for i := range lines {
		lines[i] = fmt.Sprintf("%4d | %s", i+1, lines[i])
	}
	return strings.Join(lines, "\n")
}

func wrapCodeBlock(code, extension string) string {
	if extension == "" {
		return fmt.Sprintf("```\n%s\n```", code)
	}
	return fmt.Sprintf("```%s\n%s\n```", extension[1:], code)
}

func isBinaryFile(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Read the first 512 bytes of the file
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false, err
	}

	// Use http.DetectContentType to determine the content type
	contentType := http.DetectContentType(buffer[:n])

	// Check if the content type starts with "text/"
	return !strings.HasPrefix(contentType, "text/"), nil
}

func PrintDefaultExcludes() {
	excludes, err := GetDefaultExcludes()
	if err != nil {
		utils.PrintColouredMessage("!", fmt.Sprintf("Failed to get default excludes: %v", err), color.FgRed)
		os.Exit(1)
	}
	fmt.Println(strings.Join(excludes, "\n"))
}

func processFile(path, relPath string, rootPath string, lineNumber, relativePaths, noCodeblock bool, mu *sync.Mutex, files *[]FileInfo) {
	// Check if file is a PDF
	isPDF, err := pdf.IsPDF(path)
	if err != nil {
		utils.PrintColouredMessage("!", fmt.Sprintf("Failed to check if file is PDF %s: %v", path, err), color.FgRed)
		return
	}

	if isPDF {
		content, err := pdf.ConvertPDFToMarkdown(path, false)
		if err != nil {
			utils.PrintColouredMessage("!", fmt.Sprintf("Failed to convert PDF %s: %v", path, err), color.FgRed)
			return
		}

		filePath := path
		if relativePaths {
			filePath = filepath.Join(filepath.Base(rootPath), relPath)
		}

		mu.Lock()
		*files = append(*files, FileInfo{
			Path:      filePath,
			Extension: ".md",
			Code:      content,
		})
		mu.Unlock()
		return
	}

	// Check if the file is binary
	isBinary, err := isBinaryFile(path)
	if err != nil {
		utils.PrintColouredMessage("!", fmt.Sprintf("Failed to check if file is binary %s: %v", path, err), color.FgRed)
		return
	}

	if isBinary {
		return // Skip binary files
	}

	content, err := os.ReadFile(path)
	if err != nil {
		utils.PrintColouredMessage("!", fmt.Sprintf("Failed to read file %s: %v", path, err), color.FgRed)
		return
	}

	code := string(content)
	if lineNumber {
		code = addLineNumbers(code)
	}
	if !noCodeblock {
		code = wrapCodeBlock(code, filepath.Ext(path))
	}

	filePath := path
	if relativePaths {
		filePath = filepath.Join(filepath.Base(rootPath), relPath)
	}

	mu.Lock()
	*files = append(*files, FileInfo{
		Path:      filePath,
		Extension: filepath.Ext(path),
		Code:      code,
	})
	mu.Unlock()
}

func generateTreeString(rootPath string, excludePatterns []string) (string, error) {
	root := &treeNode{name: filepath.Base(rootPath), isDir: true}
	err := filepath.Walk(rootPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}

		// Skip the root directory
		if relPath == "." {
			return nil
		}

		// Check if the path should be excluded
		if isExcluded(relPath, excludePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		parts := strings.Split(relPath, string(os.PathSeparator))
		current := root
		for i, part := range parts {
			found := false
			for _, child := range current.children {
				if child.name == part {
					current = child
					found = true
					break
				}
			}
			if !found {
				newNode := &treeNode{name: part, isDir: info.IsDir()}
				current.children = append(current.children, newNode)
				current = newNode
			}
			if i == len(parts)-1 && !info.IsDir() {
				current.isDir = false
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	var output strings.Builder
	output.WriteString(root.name + "/\n")
	for i, child := range root.children {
		printTree(child, "", i == len(root.children)-1, &output)
	}

	return strings.TrimSuffix(output.String(), "\n"), nil
}

func printTree(node *treeNode, prefix string, isLast bool, output *strings.Builder) {
	output.WriteString(prefix)
	if isLast {
		output.WriteString("└── ")
		prefix += "    "
	} else {
		output.WriteString("├── ")
		prefix += "│   "
	}
	output.WriteString(node.name)
	if node.isDir {
		output.WriteString("/")
	}
	output.WriteString("\n")

	sort.Slice(node.children, func(i, j int) bool {
		if node.children[i].isDir != node.children[j].isDir {
			return node.children[i].isDir
		}
		return node.children[i].name < node.children[j].name
	})

	for i, child := range node.children {
		printTree(child, prefix, i == len(node.children)-1, output)
	}
}
func isExcluded(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if match, _ := doublestar.Match(pattern, path); match {
			return true
		}
	}
	return false
}

func ProcessSingleFile(path string, lineNumber, relativePaths, noCodeblock bool) (FileInfo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	code := string(content)
	if lineNumber {
		code = addLineNumbers(code)
	}
	if !noCodeblock {
		code = wrapCodeBlock(code, filepath.Ext(path))
	}

	filePath := path
	if relativePaths {
		filePath = filepath.Base(path)
	}

	return FileInfo{
		Path:      filePath,
		Extension: filepath.Ext(path),
		Code:      code,
	}, nil
}
