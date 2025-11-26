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
	"github.com/sammcj/ingest/internal/compressor"
	"github.com/sammcj/ingest/pdf"
	"github.com/sammcj/ingest/utils"
)

type FileInfo struct {
	Path      string `json:"path"`
	Extension string `json:"extension"`
	Code      string `json:"code"`
}

// New type to track excluded files and directories
type ExcludedInfo struct {
	Directories map[string]int // Directory path -> count of excluded files
	Extensions  map[string]int // File extension -> count of excluded files
	TotalFiles  int            // Total number of excluded files
	Files       []string       // List of excluded files (if total ≤ 20)
}

type treeNode struct {
	name     string
	children []*treeNode
	isDir    bool
	excluded bool
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

// Helper functions to track exclusions
func trackExcludedFile(excluded *ExcludedInfo, path string, mu *sync.Mutex) {
	mu.Lock()
	defer mu.Unlock()

	excluded.TotalFiles++

	// Track the directory
	dir := filepath.Dir(path)
	excluded.Directories[dir]++

	// Track the extension
	ext := filepath.Ext(path)
	if ext != "" {
		excluded.Extensions[ext]++
	}

	// Only store individual files if we haven't exceeded 20
	if excluded.TotalFiles <= 20 {
		excluded.Files = append(excluded.Files, path)
	}
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

func trackExcludedDirectory(excluded *ExcludedInfo, path string, mu *sync.Mutex) {
	mu.Lock()
	defer mu.Unlock()
	excluded.Directories[path] = 0 // Initialize directory count
}

func WalkDirectory(rootPath string, includePatterns, excludePatterns []string, patternExclude string, includePriority, lineNumber, relativePaths, excludeFromTree, noCodeblock, noDefaultExcludes, followSymlinks bool, comp *compressor.GenericCompressor) (string, []FileInfo, *ExcludedInfo, error) {
	var files []FileInfo
	var mu sync.Mutex
	var wg sync.WaitGroup

	excluded := &ExcludedInfo{
		Directories: make(map[string]int),
		Extensions:  make(map[string]int),
		Files:       make([]string, 0),
	}

	// Read exclude patterns
	defaultExcludes, err := ReadExcludePatterns(patternExclude, noDefaultExcludes)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to read exclude patterns: %w", err)
	}

	// Combine user-provided exclude patterns with default excludes (if not disabled)
	allExcludePatterns := append(excludePatterns, defaultExcludes...)

	// Always exclude .git directories
	allExcludePatterns = append(allExcludePatterns, "**/.git/**")

	// Read .gitignore if it exists
	gitignore, err := readGitignore(rootPath)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to read .gitignore: %w", err)
	}

	// Check if rootPath is a file or directory
	fileInfo, err := os.Stat(rootPath)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Check if rootPath is a single PDF file
	if !fileInfo.IsDir() {
		isPDF, err := pdf.IsPDF(rootPath)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to check if file is PDF: %w", err)
		}

		if isPDF {
			// Process single PDF file directly
			content, err := pdf.ConvertPDFToMarkdown(rootPath, false)
			if err != nil {
				return "", nil, nil, fmt.Errorf("failed to convert PDF: %w", err)
			}

			return fmt.Sprintf("File: %s", rootPath), []FileInfo{{
				Path:      rootPath,
				Extension: ".md",
				Code:      content,
			}}, excluded, nil
		}
	}

	var treeString string

	if !fileInfo.IsDir() {
		// Check if the single file is a symlink
		if !followSymlinks {
			linkInfo, err := os.Lstat(rootPath)
			if err != nil {
				return "", nil, nil, fmt.Errorf("failed to get symlink info: %w", err)
			}
			if linkInfo.Mode()&os.ModeSymlink != 0 {
				utils.PrintColouredMessage("ℹ️", fmt.Sprintf("Skipping symlinked file: %s", rootPath), color.FgCyan)
				return fmt.Sprintf("File: %s (symlink, skipped)", rootPath), []FileInfo{}, excluded, nil
			}
		}

		// Handle single file
		relPath := filepath.Base(rootPath)
		if shouldIncludeFile(relPath, includePatterns, allExcludePatterns, gitignore, includePriority) {
			wg.Go(func() {
				processFile(rootPath, relPath, filepath.Dir(rootPath), lineNumber, relativePaths, noCodeblock, &mu, &files, comp)
			})
		} else {
			trackExcludedFile(excluded, rootPath, &mu)
		}
		treeString = fmt.Sprintf("File: %s", rootPath)
	} else {
		// Generate the tree representation for directory
		treeString, err = generateTreeString(rootPath, allExcludePatterns)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to generate directory tree: %w", err)
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

			// Check if the path is a symlink
			if !followSymlinks {
				linkInfo, err := os.Lstat(path)
				if err != nil {
					return err
				}
				if linkInfo.Mode()&os.ModeSymlink != 0 {
					if linkInfo.IsDir() || (info != nil && info.IsDir()) {
						utils.PrintColouredMessage("ℹ️", fmt.Sprintf("Skipping symlinked directory: %s", path), color.FgCyan)
						return filepath.SkipDir
					}
					utils.PrintColouredMessage("ℹ️", fmt.Sprintf("Skipping symlinked file: %s", path), color.FgCyan)
					return nil
				}
			}

			// Check if the current path (file or directory) should be excluded
			if shouldExcludePath(relPath, allExcludePatterns, gitignore) {
				if info.IsDir() {
					trackExcludedDirectory(excluded, path, &mu)
					return filepath.SkipDir
				}
				trackExcludedFile(excluded, path, &mu)
				return nil
			}

			if !info.IsDir() && !shouldIncludeFile(relPath, includePatterns, allExcludePatterns, gitignore, includePriority) {
				trackExcludedFile(excluded, path, &mu)
				return nil
			}

			if !info.IsDir() && shouldIncludeFile(relPath, includePatterns, allExcludePatterns, gitignore, includePriority) {
				wg.Add(1)
				go func(path, relPath string) {
					defer wg.Done()
					processFile(path, relPath, rootPath, lineNumber, relativePaths, noCodeblock, &mu, &files, comp)
				}(path, relPath)
			}

			return nil
		})
	}

	wg.Wait()

	if err != nil {
		return "", nil, excluded, err
	}

	return treeString, files, excluded, nil
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
	// First check if it's a PDF
	isPDF, err := pdf.IsPDF(filePath)
	if err != nil {
		return false, err
	}
	if isPDF {
		return false, nil // Don't treat PDFs as binary files
	}

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

	// Allow PDFs and text files
	return !strings.HasPrefix(contentType, "text/") && contentType != "application/pdf", nil
}

func PrintDefaultExcludes() {
	excludes, err := GetDefaultExcludes()
	if err != nil {
		utils.PrintColouredMessage("!", fmt.Sprintf("Failed to get default excludes: %v", err), color.FgRed)
		os.Exit(1)
	}
	fmt.Println(strings.Join(excludes, "\n"))
}

func processFile(path, relPath string, rootPath string, lineNumber, relativePaths, noCodeblock bool, mu *sync.Mutex, files *[]FileInfo, comp *compressor.GenericCompressor) {
	// Check if it's the root path being processed (explicitly provided file)
	isExplicitFile := path == rootPath

	// Check if file is a PDF
	isPDF, err := pdf.IsPDF(path)
	if err != nil {
		utils.PrintColouredMessage("!", fmt.Sprintf("Failed to check if file is PDF %s: %v", path, err), color.FgRed)
		return
	}

	if isPDF {
		if !isExplicitFile {
			// Skip PDFs during directory traversal
			return
		}

		utils.PrintColouredMessage("ℹ️", fmt.Sprintf("Converting PDF to markdown: %s", path), color.FgBlue)
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

	// Attempt compression if compressor is provided and it's not a PDF
	if comp != nil && !isPDF {
		langID, err := compressor.IdentifyLanguage(path)
		if err == nil { // Language identified
			compressedCode, err := comp.Compress(content, langID)
			if err == nil {
				code = compressedCode
				// If compressed, we might not want to add line numbers or wrap in a generic code block
				// as the compressor might handle formatting. For now, let's assume compressed output
				// is final for this file's content.
				// We'll skip line numbering and code block wrapping for compressed content.
				goto skipFormatting
			} else {
				utils.PrintColouredMessage("⚠️", fmt.Sprintf("Compression failed for %s: %v. Using original content.", path, err), color.FgYellow)
			}
		} else {
			// Language not identified for compression, use original content
			utils.PrintColouredMessage("ℹ️", fmt.Sprintf("Language not identified for compression for %s. Using original content.", path), color.FgBlue)
		}
	}

	if lineNumber {
		code = addLineNumbers(code)
	}
	if !noCodeblock {
		code = wrapCodeBlock(code, filepath.Ext(path))
	}

skipFormatting:
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
	hasExclusions := false

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
		excluded := isExcluded(relPath, excludePatterns)
		if excluded {
			hasExclusions = true
			if info.IsDir() {
				// Add the excluded directory to the tree with an X marker
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
						newNode := &treeNode{
							name:     part,
							isDir:    true,
							excluded: true,
						}
						current.children = append(current.children, newNode)
						current = newNode
					}
					if i == len(parts)-1 {
						current.isDir = true
						current.excluded = true
					}
				}
				return filepath.SkipDir
			}
			// Add excluded files to the tree with an X marker
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
					newNode := &treeNode{
						name:     part,
						isDir:    i < len(parts)-1,
						excluded: true,
					}
					current.children = append(current.children, newNode)
					current = newNode
				}
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
	if hasExclusions {
		output.WriteString("(Files/directories marked with ❌ are excluded or not included here)\n\n")
	}
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
	if node.excluded {
		output.WriteString(" ❌")
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
			if strings.HasSuffix(path, ".pdf") {
				utils.PrintColouredMessage("ℹ️", fmt.Sprintf("PDF file detected: %s. PDF to markdown conversion is supported, but the file was excluded", path), color.FgYellow)
			}
			return true
		}
	}
	return false
}

func ProcessSingleFile(path string, lineNumber, relativePaths, noCodeblock, followSymlinks bool, comp *compressor.GenericCompressor) (FileInfo, error) {
	// Check if the file is a symlink
	if !followSymlinks {
		linkInfo, err := os.Lstat(path)
		if err != nil {
			return FileInfo{}, fmt.Errorf("failed to get symlink info: %w", err)
		}
		if linkInfo.Mode()&os.ModeSymlink != 0 {
			utils.PrintColouredMessage("ℹ️", fmt.Sprintf("Skipping symlinked file: %s", path), color.FgCyan)
			return FileInfo{}, fmt.Errorf("file is a symlink and --follow-symlinks is not set")
		}
	}

	// Check if it's a PDF first
	isPDF, err := pdf.IsPDF(path)
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to check if file is PDF: %w", err)
	}

	if isPDF {
		content, err := pdf.ConvertPDFToMarkdown(path, false)
		if err != nil {
			return FileInfo{}, fmt.Errorf("failed to convert PDF: %w", err)
		}

		return FileInfo{
			Path:      path,
			Extension: ".md",
			Code:      content,
		}, nil
	}

	// Handle non-PDF files
	content, err := os.ReadFile(path)
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to read file: %w", err)
	}

	code := string(content)

	// Attempt compression if compressor is provided and it's not a PDF
	if comp != nil && !isPDF {
		langID, err := compressor.IdentifyLanguage(path)
		if err == nil { // Language identified
			compressedCode, err := comp.Compress(content, langID)
			if err == nil {
				code = compressedCode
				// Skip standard formatting for compressed content
				goto skipSingleFileFormatting
			} else {
				utils.PrintColouredMessage("⚠️", fmt.Sprintf("Compression failed for %s: %v. Using original content.", path, err), color.FgYellow)
			}
		} else {
			utils.PrintColouredMessage("ℹ️", fmt.Sprintf("Language not identified for compression for %s. Using original content.", path), color.FgBlue)
		}
	}

	if lineNumber {
		code = addLineNumbers(code)
	}
	if !noCodeblock {
		code = wrapCodeBlock(code, filepath.Ext(path))
	}

skipSingleFileFormatting:
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
