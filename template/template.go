package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"github.com/sammcj/ingest/utils"
)

func SetupTemplate(templatePath string) (*template.Template, error) {
	var templateContent string
	var err error

	if templatePath != "" {
		templateContent, err = readTemplateFile(templatePath)
	} else {
		templateContent, err = getDefaultTemplate()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("default").Parse(templateContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return tmpl, nil
}

func readTemplateFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func getDefaultTemplate() (string, error) {
	// Check for user-specific template
	home, err := homedir.Dir()
	if err == nil {
		userTemplateDir := filepath.Join(home, ".config", "ingest", "patterns", "templates")
		userDefaultTemplate := filepath.Join(userTemplateDir, "default.tmpl")
		if _, err := os.Stat(userDefaultTemplate); err == nil {
			return readTemplateFile(userDefaultTemplate)
		}
	}

	// If no user-specific template, use built-in template
	return readEmbeddedTemplate()
}

func readEmbeddedTemplate() (string, error) {
	return `
Source Trees:

{{.source_trees}}

{{if .excluded}}
Excluded Content:
{{if le .excluded.TotalFiles 20}}
Files:
{{range .excluded.Files}}
- {{.}}
{{end}}
{{else}}
Directories with excluded files:
{{range $dir, $count := .excluded.Directories}}
{{if gt $count 0}}- {{$dir}}: {{$count}} files{{end}}
{{end}}

File extensions excluded:
{{range $ext, $count := .excluded.Extensions}}
- {{$ext}}: {{$count}} files
{{end}}
{{end}}

{{end}}

{{range .files}}
{{if .Code}}
` + "`{{.Path}}:`" + `

{{.Code}}

{{end}}
{{end}}

{{range .git_data}}
{{if or .GitDiff .GitDiffBranch .GitLogBranch}}
Git Information for {{.Path}}:
{{if .GitDiff}}
Git Diff:
{{.GitDiff}}
{{end}}
{{if .GitDiffBranch}}
Git Diff Between Branches:
{{.GitDiffBranch}}
{{end}}
{{if .GitLogBranch}}
Git Log Between Branches:
{{.GitLogBranch}}
{{end}}
{{end}}
{{end}}
`, nil
}

func RenderTemplate(tmpl *template.Template, data map[string]any) (string, error) {
	var output strings.Builder
	err := tmpl.Execute(&output, data)
	if err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}
	return output.String(), nil
}

func PrintDefaultTemplate() {
	defaultTemplate, err := readEmbeddedTemplate()
	if err != nil {
		utils.PrintColouredMessage("!", fmt.Sprintf("Failed to get default template: %v", err), color.FgRed)
		os.Exit(1)
	}
	fmt.Println(defaultTemplate)
}
