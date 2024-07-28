package template

import (
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/aymerick/raymond"
)

func SetupHandlebars(templateContent, templateName string) *raymond.Template {
	tpl, err := raymond.Parse(templateContent)
	if err != nil {
		fmt.Printf("Failed to parse template: %v\n", err)
		return nil
	}
	return tpl
}

func RenderTemplate(tpl *raymond.Template, data map[string]interface{}) string {
	result, err := tpl.Exec(data)
	if err != nil {
		fmt.Printf("Failed to render template: %v\n", err)
		return ""
	}
	return result
}

func ExtractUndefinedVariables(templateContent string) []string {
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	matches := re.FindAllStringSubmatch(templateContent, -1)

	var undefinedVars []string
	for _, match := range matches {
		if len(match) > 1 {
			undefinedVars = append(undefinedVars, match[1])
		}
	}

	return undefinedVars
}

func HandleUndefinedVariables(data *map[string]interface{}, templateContent string) {
	undefinedVars := ExtractUndefinedVariables(templateContent)
	for _, v := range undefinedVars {
		if _, exists := (*data)[v]; !exists {
			fmt.Printf("Enter value for '%s': ", v)
			var value string
			fmt.Scanln(&value)
			(*data)[v] = value
		}
	}
}

func GetTemplate(templatePath string) (string, string) {
	if templatePath != "" {
		content, err := ioutil.ReadFile(templatePath)
		if err != nil {
			fmt.Printf("Failed to read custom template file: %v\n", err)
			return "", ""
		}
		return string(content), "custom"
	}
	return defaultTemplate, "default"
}

const defaultTemplate = `
Project Path: {{ absolute_code_path }}

Source Tree:

` + "```" + `
{{ source_tree }}
` + "```" + `

{{#each files}}
{{#if code}}
` + "`{{path}}:`" + `

{{code}}

{{/if}}
{{/each}}
`
