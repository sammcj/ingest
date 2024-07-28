package filesystem

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type FileInfo struct {
	Path      string `json:"path"`
	Extension string `json:"extension"`
	Code      string `json:"code"`
}

func TraverseDirectory(rootPath string, includePatterns, excludePatterns []string, includePriority, lineNumber, relativePaths, excludeFromTree, noCodeblock bool) (string, []FileInfo, error) {
	var files []FileInfo
	tree := ""

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}

		if shouldIncludeFile(relPath, includePatterns, excludePatterns, includePriority) {
			if info.IsDir() {
				if !excludeFromTree {
					tree += strings.Repeat("  ", strings.Count(relPath, string(os.PathSeparator))) + filepath.Base(path) + "\n"
				}
			} else {
				content, err := ioutil.ReadFile(path)
				if err != nil {
					return err
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

				files = append(files, FileInfo{
					Path:      filePath,
					Extension: filepath.Ext(path),
					Code:      code,
				})

				if !excludeFromTree {
					tree += strings.Repeat("  ", strings.Count(relPath, string(os.PathSeparator))) + filepath.Base(path) + "\n"
				}
			}
		}

		return nil
	})

	return tree, files, err
}

func shouldIncludeFile(path string, includePatterns, excludePatterns []string, includePriority bool) bool {
	included := len(includePatterns) == 0 || matchesAny(path, includePatterns)
	excluded := matchesAny(path, excludePatterns)

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

func addLineNumbers(code string) string {
	lines := strings.Split(code, "\n")
	for i := range lines {
		lines[i] = fmt.Sprintf("%4d | %s", i+1, lines[i])
	}
	return strings.Join(lines, "\n")
}

func wrapCodeBlock(code, extension string) string {
	return "```" + extension[1:] + "\n" + code + "\n```"
}
