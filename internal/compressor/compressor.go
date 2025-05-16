package compressor

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/css"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
)

// LanguageMap maps language identifiers to their Tree-sitter Language.
var LanguageMap = map[string]*sitter.Language{
	"go":         golang.GetLanguage(),
	"python":     python.GetLanguage(),
	"javascript": javascript.GetLanguage(),
	"bash":       bash.GetLanguage(),
	"c":          c.GetLanguage(),
	"css":        css.GetLanguage(),
	"html":       javascript.GetLanguage(), // Use JavaScript parser for HTML
	"rust":       javascript.GetLanguage(), // Use JavaScript parser for Rust
	"java":       javascript.GetLanguage(), // Use JavaScript parser for Java
	"swift":      javascript.GetLanguage(), // Use JavaScript parser for Swift
}

// IdentifyLanguage identifies the programming language of a file based on its extension.
func IdentifyLanguage(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go", nil
	case ".py":
		return "python", nil
	case ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx":
		return "javascript", nil
	case ".sh", ".bash":
		return "bash", nil
	case ".c", ".h":
		return "c", nil
	case ".css":
		return "css", nil
	case ".html", ".htm":
		return "html", nil
	case ".rs":
		return "rust", nil
	case ".java":
		return "java", nil
	case ".swift":
		return "swift", nil
	// Add more extensions and languages here
	default:
		return "", fmt.Errorf("unsupported file extension: %s", ext)
	}
}

// GetLanguage returns the Tree-sitter language for a given language identifier.
func GetLanguage(langIdentifier string) (*sitter.Language, error) {
	lang, ok := LanguageMap[strings.ToLower(langIdentifier)]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", langIdentifier)
	}
	return lang, nil
}

// ParseSource parses the source code content using the appropriate Tree-sitter language.
func ParseSource(content []byte, lang *sitter.Language) (*sitter.Tree, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(lang)
	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
	}
	return tree, nil
}

// QueryMap holds Tree-sitter queries for different languages.
var QueryMap = map[string]string{
	"go": `
        (package_clause) @package
        (import_declaration) @import
        (type_declaration) @definition.type
        (function_declaration name: (identifier) @definition.function) @definition.function.full
        (method_declaration name: (field_identifier) @definition.method) @definition.method.full
        (comment) @comment
        `,
	"python": `
        (import_statement) @import
        (import_from_statement) @import
        (function_definition name: (identifier) @definition.function) @definition.function.full
        (class_definition name: (identifier) @definition.class) @definition.class.full
        (comment) @comment
        `,
	"bash": `
        (function_definition name: (word) @definition.function) @definition.function.full
        (command name: (command_name) @command_name) @command
        (comment) @comment
        `,
	"c": `
        (preproc_include) @import
        (function_definition declarator: (function_declarator declarator: (identifier) @definition.function)) @definition.function.full
        (struct_specifier name: (type_identifier) @definition.struct) @definition.struct.full
        (enum_specifier name: (type_identifier) @definition.enum) @definition.enum.full
        (union_specifier name: (type_identifier) @definition.union) @definition.union.full
        (type_definition) @definition.typedef
        (comment) @comment
        `,
	"css": `
        (import_statement) @import
        (rule_set) @rule_set
        (media_statement) @media
        (keyframes_statement) @keyframes
        (declaration) @declaration
        (comment) @comment
        `,
	"html": `
        ; HTML elements in JavaScript parser
        ((comment) @comment)
        ((regex) @regex)
        ((string) @string)
        ((template_string) @template_string)
        ((identifier) @identifier)
        ((property_identifier) @property)
        `,
	"rust": `
        ; Rust elements using JavaScript parser
        ((comment) @comment)
        ((string) @string)
        ((regex) @regex)
        ((template_string) @template_string)
        ((identifier) @identifier)
        ((property_identifier) @property)
        `,
	"java": `
        ; Java elements using JavaScript parser
        ((comment) @comment)
        ((string) @string)
        ((regex) @regex)
        ((template_string) @template_string)
        ((identifier) @identifier)
        ((property_identifier) @property)
        `,
	"swift": `
        ; Swift elements using JavaScript parser
        ((comment) @comment)
        ((string) @string)
        ((regex) @regex)
        ((template_string) @template_string)
        ((identifier) @identifier)
        ((property_identifier) @property)
        `,
	"javascript": `
        (import_statement) @import
        (comment) @comment
        (method_definition name: (_) @_.name) @definition.method.full

        ; === Definitions whose bodies should be stripped ===

        ; Case 1a: Exported NAMED function/class/generator (e.g., export function foo() {})
        ; Captures the export_statement node. 'declaration' is the field name for the exported item.
        (export_statement
          declaration: (function_declaration name: (identifier))
        ) @definition.function.full
        (export_statement
          declaration: (generator_function_declaration name: (identifier))
        ) @definition.function.full
        (export_statement
          declaration: (class_declaration name: (identifier))
        ) @definition.class.full

        ; Case 1b: Export DEFAULT NAMED function/class/generator (e.g., export default function foo() {})
        ; Captures the export_statement node. The declaration is a direct child after the 'default' keyword.
        (export_statement
          "default" ; Matches the anonymous 'default' keyword child node by its string content
          (function_declaration name: (identifier))
        ) @definition.function.full
        (export_statement
          "default"
          (generator_function_declaration name: (identifier))
        ) @definition.function.full
        (export_statement
          "default"
          (class_declaration name: (identifier))
        ) @definition.class.full

        ; Case 2: Standalone (non-exported by above rules) NAMED function/class/generator
        ; Captures the function_declaration or class_declaration node itself.
        (function_declaration name: (identifier)) @definition.function.full
        (generator_function_declaration name: (identifier)) @definition.function.full
        (class_declaration name: (identifier)) @definition.class.full

        ; Case 3: Arrow functions assigned to variables (const/let/var)
        ; The whole lexical_declaration or variable_declaration is captured if it contains an arrow function with a block body.
        (lexical_declaration
          (variable_declarator
            value: (arrow_function body: (statement_block))
          )
        ) @definition.function.full
        (variable_declaration
          (variable_declarator
            value: (arrow_function body: (statement_block))
          )
        ) @definition.function.full

        ; Case 4: Arrow functions with expression bodies assigned to variables
        ; These are arrow functions that don't have a statement_block body (e.g., const myArrow = () => expression)
        (lexical_declaration
          (variable_declarator
            name: (identifier) @arrow.name
            value: (arrow_function) @arrow.function
          )
        ) @definition.function.full
        (variable_declaration
          (variable_declarator
            name: (identifier) @arrow.name
            value: (arrow_function) @arrow.function
          )
        ) @definition.function.full

        ; Case 5: Anonymous default exported functions and classes
        ; These are functions and classes without names that are exported as default
        (export_statement
          "default"
          (function_declaration) @anon.function
        ) @definition.function.full
        (export_statement
          "default"
          (class_declaration) @anon.class
        ) @definition.class.full

        ; === Other export forms to be kept whole ===
        ; Capture name is @export.other

        ; export const x = 1;, export let y = 2; (declaration is lexical_declaration)
        (export_statement declaration: (lexical_declaration)) @export.other
        ; export var z = 3; (declaration is variable_declaration)
        (export_statement declaration: (variable_declaration)) @export.other

        ; export { foo, bar };, export { foo as bar } from 'module';, export * from 'module';
        (export_statement (export_clause)) @export.other

        ; export default foo; (where foo is an identifier/expression - 'value' is the field name for these)
        (export_statement value: (identifier)) @export.other
        (export_statement value: (string)) @export.other
        (export_statement value: (number)) @export.other
        (export_statement value: (object)) @export.other
        (export_statement value: (array)) @export.other
        (export_statement value: (arrow_function)) @export.other
        `,
}

// GetQuery retrieves a Tree-sitter query for a given language identifier.
func GetQuery(languageIdentifier string) (string, error) {
	query, ok := QueryMap[strings.ToLower(languageIdentifier)]
	if !ok {
		return "", fmt.Errorf("no query found for language: %s", languageIdentifier)
	}
	return query, nil
}

// CompileQuery compiles a query string for a given language.
func CompileQuery(queryStr string, lang *sitter.Language) (*sitter.Query, error) {
	query, err := sitter.NewQuery([]byte(queryStr), lang)
	if err != nil {
		return nil, fmt.Errorf("failed to compile query: %w", err)
	}
	return query, nil
}

// ExecuteQuery executes a compiled query against a syntax tree.
func ExecuteQuery(tree *sitter.Tree, query *sitter.Query, source []byte) ([]*sitter.QueryMatch, error) {
	qc := sitter.NewQueryCursor()
	qc.Exec(query, tree.RootNode())
	var matches []*sitter.QueryMatch
	for {
		match, ok := qc.NextMatch()
		if !ok {
			break
		}
		isDuplicate := false
		for _, existingMatch := range matches {
			if existingMatch.ID == match.ID && existingMatch.PatternIndex == match.PatternIndex {
				if len(existingMatch.Captures) == len(match.Captures) {
					allCapturesSame := true
					for i := range match.Captures {
						if existingMatch.Captures[i].Index != match.Captures[i].Index ||
							existingMatch.Captures[i].Node.StartByte() != match.Captures[i].Node.StartByte() ||
							existingMatch.Captures[i].Node.EndByte() != match.Captures[i].Node.EndByte() {
							allCapturesSame = false
							break
						}
					}
					if allCapturesSame {
						isDuplicate = true
						break
					}
				}
			}
		}
		if !isDuplicate {
			matches = append(matches, match)
		}
	}
	return matches, nil
}

// CodeChunk represents a captured piece of code.
type CodeChunk struct {
	Content      string
	StartByte    uint32
	EndByte      uint32
	OriginalLine int
}

func isNamedDeclarationType(n *sitter.Node) bool {
	if n == nil {
		return false
	}
	switch n.Type() {
	case "function_declaration", "generator_function_declaration", "class_declaration":
		nameNode := n.ChildByFieldName("name")
		return nameNode != nil && nameNode.Type() == "identifier"
	default:
		return false
	}
}

func LogCaptures(matches []*sitter.QueryMatch, query *sitter.Query, source []byte) {
	for _, match := range matches {
		for _, capture := range match.Captures {
			captureName := query.CaptureNameForId(capture.Index)
			nodeContent := capture.Node.Content(source)
			fmt.Printf("Capture: @%s, Node Type: %s, Content: %s\n",
				captureName, capture.Node.Type(), strings.TrimSpace(nodeContent))
		}
	}
}
