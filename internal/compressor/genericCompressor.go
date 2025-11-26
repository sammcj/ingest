package compressor

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// GenericCompressor handles the compression of source code.
type GenericCompressor struct{}

// NewGenericCompressor creates a new GenericCompressor.
func NewGenericCompressor() *GenericCompressor {
	return &GenericCompressor{}
}

// Compress takes source code content and a language identifier,
// and returns the compressed code as a string.
func (gc *GenericCompressor) Compress(content []byte, languageIdentifier string) (string, error) {
	// Special cases for languages that need custom handling
	switch languageIdentifier {
	case "html":
		return gc.compressHTML(content), nil
	case "rust":
		return gc.compressRust(content), nil
	case "java":
		return gc.compressJava(content), nil
	case "swift":
		return gc.compressSwift(content), nil
	}

	lang, err := GetLanguage(languageIdentifier)
	if err != nil {
		return "", fmt.Errorf("could not get language for '%s': %w", languageIdentifier, err)
	}

	tree, err := ParseSource(content, lang)
	if err != nil {
		return "", fmt.Errorf("could not parse source for '%s': %w", languageIdentifier, err)
	}
	defer tree.Close()

	queryStr, err := GetQuery(languageIdentifier)
	if err != nil {
		return "", fmt.Errorf("could not get query for '%s': %w", languageIdentifier, err)
	}

	query, err := CompileQuery(queryStr, lang)
	if err != nil {
		return "", fmt.Errorf("could not compile query for '%s': %w", languageIdentifier, err)
	}
	defer query.Close()

	matches, err := ExecuteQuery(tree, query, content)
	if err != nil {
		return "", fmt.Errorf("could not execute query for '%s': %w", languageIdentifier, err)
	}

	processedChunks := gc.processCaptures(matches, query, content, languageIdentifier)

	sort.SliceStable(processedChunks, func(i, j int) bool {
		return processedChunks[i].StartByte < processedChunks[j].StartByte
	})

	var result strings.Builder
	processedNodes := make(map[uint32]struct{})

	for _, chunk := range processedChunks {
		if _, exists := processedNodes[chunk.StartByte]; !exists {
			result.WriteString(chunk.Content)
			result.WriteString("\n// -----\n")
			processedNodes[chunk.StartByte] = struct{}{}
		}
	}

	if result.Len() == 0 {
		return "// No relevant code found after compression.\n", nil
	}

	return result.String(), nil
}

// compressHTML is a specialized function to compress HTML content
// since Tree-sitter doesn't handle HTML well with the JavaScript parser
func (gc *GenericCompressor) compressHTML(content []byte) string {
	// Convert content to string for easier processing
	htmlStr := string(content)

	// Extract HTML comments
	commentRegex := regexp.MustCompile(`<!--([\s\S]*?)-->`)
	comments := commentRegex.FindAllString(htmlStr, -1)

	// Extract important tags (doctype, html, head, body, script, style)
	doctypeRegex := regexp.MustCompile(`<!DOCTYPE[^>]*>`)
	doctypes := doctypeRegex.FindAllString(htmlStr, -1)

	htmlTagRegex := regexp.MustCompile(`<html[^>]*>|</html>`)
	htmlTags := htmlTagRegex.FindAllString(htmlStr, -1)

	headTagRegex := regexp.MustCompile(`<head[^>]*>|</head>`)
	headTags := headTagRegex.FindAllString(htmlStr, -1)

	bodyTagRegex := regexp.MustCompile(`<body[^>]*>|</body>`)
	bodyTags := bodyTagRegex.FindAllString(htmlStr, -1)

	// Combine all extracted elements
	var chunks []string
	chunks = append(chunks, doctypes...)
	chunks = append(chunks, htmlTags...)
	chunks = append(chunks, headTags...)
	chunks = append(chunks, comments...)
	chunks = append(chunks, bodyTags...)

	if len(chunks) == 0 {
		return "// No relevant HTML elements found after compression.\n"
	}

	var result strings.Builder
	for _, chunk := range chunks {
		result.WriteString(strings.TrimSpace(chunk))
		result.WriteString("\n// -----\n")
	}

	return result.String()
}

func (gc *GenericCompressor) processCaptures(matches []*sitter.QueryMatch, query *sitter.Query, source []byte, languageIdentifier string) []CodeChunk {
	var chunks []CodeChunk
	seenNodes := make(map[uint32]struct{}) // Tracks nodes already processed to avoid duplicates

	for _, match := range matches {
		for _, capture := range match.Captures {
			node := capture.Node
			if node == nil {
				continue
			}

			if _, ok := seenNodes[node.StartByte()]; ok {
				continue
			}

			captureName := query.CaptureNameForId(capture.Index)
			nodeContent := node.Content(source)
			chunkContent := ""

			switch {
			case strings.HasPrefix(captureName, "definition.function"),
				strings.HasPrefix(captureName, "definition.method"),
				strings.HasPrefix(captureName, "definition.class"):

				jsNode := node                  // The node captured by the query pattern.
				actualDeclarationNode := jsNode // Default for non-JS or simple cases. This is the node that has the .body child.
				var bodyNode *sitter.Node

				if languageIdentifier == "javascript" || languageIdentifier == "typescript" {
					// Check for arrow function with expression body (not statement block)
					arrowName := ""
					for _, c := range match.Captures {
						if query.CaptureNameForId(c.Index) == "arrow.name" {
							arrowName = c.Node.Content(source)
							break
						}
					}

					// Handle arrow functions with expression bodies
					if arrowName != "" {
						var arrowFunc *sitter.Node
						for _, c := range match.Captures {
							if query.CaptureNameForId(c.Index) == "arrow.function" {
								arrowFunc = c.Node
								break
							}
						}

						if arrowFunc != nil {
							// For arrow functions with expression bodies, we need to construct the signature differently
							params := arrowFunc.ChildByFieldName("parameters")
							body := arrowFunc.ChildByFieldName("body")

							if params != nil && body != nil && body.Type() != "statement_block" {
								// This is an arrow function with expression body
								signature := fmt.Sprintf("const %s = %s =>",
									strings.TrimSpace(arrowName),
									strings.TrimSpace(params.Content(source)))
								chunkContent = signature + " { ... } // Body removed"

								// Add to chunks and continue to next capture
								if chunkContent != "" {
									chunks = append(chunks, CodeChunk{
										Content:   chunkContent,
										StartByte: node.StartByte(),
										EndByte:   node.EndByte(),
									})
									seenNodes[node.StartByte()] = struct{}{}
								}
								continue
							}
						}
					}

					// Check for anonymous default exported function/class
					var anonNode *sitter.Node
					for _, c := range match.Captures {
						if query.CaptureNameForId(c.Index) == "anon.function" || query.CaptureNameForId(c.Index) == "anon.class" {
							anonNode = c.Node
							break
						}
					}

					if anonNode != nil {
						bodyNode = anonNode.ChildByFieldName("body")
						if bodyNode != nil {
							// For anonymous default exports, we need to construct the signature differently
							signature := ""
							if anonNode.Type() == "function_declaration" {
								signature = "export default function()"
							} else if anonNode.Type() == "class_declaration" {
								signature = "export default class"
							}
							chunkContent = signature + " { ... } // Body removed"

							// Add to chunks and continue to next capture
							if chunkContent != "" {
								chunks = append(chunks, CodeChunk{
									Content:   chunkContent,
									StartByte: node.StartByte(),
									EndByte:   node.EndByte(),
								})
								seenNodes[node.StartByte()] = struct{}{}
							}
							continue
						}
					}

					// Standard processing for other JavaScript constructs
					switch jsNode.Type() {
					case "export_statement":
						foundDecl := jsNode.ChildByFieldName("declaration")
						if foundDecl != nil && (isNamedDeclarationType(foundDecl) || foundDecl.Type() == "arrow_function" || foundDecl.Type() == "function_expression") {
							actualDeclarationNode = foundDecl
						} else {
							// Check for `export default function_declaration` etc. (direct named child)
							var directChildDecl *sitter.Node
							for i := 0; i < int(jsNode.NamedChildCount()); i++ {
								child := jsNode.NamedChild(i)
								// Broaden condition to find anonymous function/class declarations as well
								switch child.Type() {
								case "function_declaration", "class_declaration", "generator_function_declaration", "arrow_function", "function_expression":
									// These are types that can be default exported and might have bodies to strip.
									directChildDecl = child
								default:
									// Not a type we are looking for as a direct declaration in `export default ...`
								}
							}
							if directChildDecl != nil {
								actualDeclarationNode = directChildDecl
							} else {
								actualDeclarationNode = nil // No suitable declaration found in export statement
							}
						}
					case "lexical_declaration", "variable_declaration":
						// For `const foo = () => {}` or `var bar = function() {}`
						// jsNode is lexical_declaration/variable_declaration.
						// We need to find the specific variable_declarator with an arrow_function/function_expression.
						var targetArrowFunc *sitter.Node
						for i := 0; i < int(jsNode.NamedChildCount()); i++ {
							declarator := jsNode.NamedChild(i)
							if declarator != nil && declarator.Type() == "variable_declarator" {
								valueNode := declarator.ChildByFieldName("value")
								if valueNode != nil && (valueNode.Type() == "arrow_function" || valueNode.Type() == "function_expression") {
									// Check if this arrow_function/function_expression has a statement_block body,
									// as per the query (`body: (statement_block)`).
									bodyCheck := valueNode.ChildByFieldName("body")
									if bodyCheck != nil {
										targetArrowFunc = valueNode
										break
									}
								}
							}
						}
						if targetArrowFunc != nil {
							actualDeclarationNode = targetArrowFunc
						} else {
							actualDeclarationNode = nil // No suitable arrow function found
						}
					case "function_declaration", "generator_function_declaration", "class_declaration", "method_definition":
						// jsNode is already the one with the body.
						actualDeclarationNode = jsNode
					default:
						actualDeclarationNode = nil // Other types not meant for body stripping by this rule.
					}
				} else if languageIdentifier == "go" || languageIdentifier == "python" || languageIdentifier == "bash" || languageIdentifier == "c" {
					actualDeclarationNode = jsNode // For Go/Python/Bash/C, the captured node is the declaration itself
				} else {
					actualDeclarationNode = nil // Should not happen if language is supported
				}

				if actualDeclarationNode != nil {
					bodyNode = actualDeclarationNode.ChildByFieldName("body")
				}

				if bodyNode != nil {
					var signatureEndPos uint32
					if languageIdentifier == "python" {
						var colonNode *sitter.Node
						for i := 0; i < int(actualDeclarationNode.ChildCount()); i++ {
							childNode := actualDeclarationNode.Child(i)
							if childNode != nil && childNode.Type() == ":" {
								colonNode = childNode
								break
							}
						}
						if colonNode != nil {
							signatureEndPos = colonNode.EndByte()
						} else {
							signatureEndPos = bodyNode.StartByte() // Fallback
						}
					} else {
						// For Go and JS (functions, methods, classes, or arrow funcs with statement_block bodies)
						// Signature ends where the body node begins.
						signatureEndPos = bodyNode.StartByte()
					}

					// Ensure signaturePortion starts from the beginning of the originally captured node (`jsNode`).
					if signatureEndPos > jsNode.StartByte() && signatureEndPos <= jsNode.EndByte() {
						signaturePortion := string(source[jsNode.StartByte():signatureEndPos])
						trimmedSignature := strings.TrimSpace(signaturePortion)

						placeholder := " { ... }"
						switch languageIdentifier {
						case "python":
							placeholder = " { ... } # Body removed"
						case "javascript", "typescript", "c", "html":
							placeholder = " { ... } // Body removed"
						case "bash":
							placeholder = " { ... } # Body removed"
						}
						chunkContent = trimmedSignature + placeholder
					} else {
						// Fallback: if positions are unusual, keep the original content of the captured node.
						chunkContent = strings.TrimSpace(jsNode.Content(source))
					}
				} else {
					// No bodyNode found, or actualDeclarationNode was nil (e.g. not a strippable type, or error in logic).
					// Keep the original content of the captured node.
					chunkContent = strings.TrimSpace(jsNode.Content(source))
				}

			case strings.HasPrefix(captureName, "import"),
				strings.HasPrefix(captureName, "package"),
				strings.HasPrefix(captureName, "definition.type"),
				strings.HasPrefix(captureName, "definition.struct"),
				strings.HasPrefix(captureName, "definition.enum"),
				strings.HasPrefix(captureName, "definition.union"),
				strings.HasPrefix(captureName, "definition.typedef"),
				strings.HasPrefix(captureName, "export.other"),
				strings.HasPrefix(captureName, "command"),
				strings.HasPrefix(captureName, "rule_set"),
				strings.HasPrefix(captureName, "media"),
				strings.HasPrefix(captureName, "keyframes"),
				strings.HasPrefix(captureName, "declaration"),
				strings.HasPrefix(captureName, "comment"),
				strings.HasPrefix(captureName, "doctype"),
				strings.HasPrefix(captureName, "tag"),
				strings.HasPrefix(captureName, "script"),
				strings.HasPrefix(captureName, "style"):
				chunkContent = strings.TrimSpace(nodeContent)
			default:
				continue
			}

			if chunkContent != "" {
				chunks = append(chunks, CodeChunk{
					Content:   chunkContent,
					StartByte: node.StartByte(),
					EndByte:   node.EndByte(),
				})
				seenNodes[node.StartByte()] = struct{}{}
			}
		}
	}
	return chunks
}

// compressSwift is a specialized function to compress Swift content
func (gc *GenericCompressor) compressSwift(content []byte) string {
	// Convert content to string for easier processing
	swiftStr := string(content)

	// Extract Swift comments
	singleLineCommentRegex := regexp.MustCompile(`(?m)//.*$`)
	singleLineComments := singleLineCommentRegex.FindAllString(swiftStr, -1)

	multiLineCommentRegex := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	multiLineComments := multiLineCommentRegex.FindAllString(swiftStr, -1)

	// Extract imports
	importRegex := regexp.MustCompile(`import\s+\w+`)
	imports := importRegex.FindAllString(swiftStr, -1)

	// Extract struct definitions
	structRegex := regexp.MustCompile(`(?:public\s+|private\s+|fileprivate\s+|internal\s+)?struct\s+\w+(?::\s*[^{]+)?\s*\{`)
	structs := structRegex.FindAllString(swiftStr, -1)

	// Extract class definitions
	classRegex := regexp.MustCompile(`(?:public\s+|private\s+|fileprivate\s+|internal\s+)?(?:final\s+)?class\s+\w+(?::\s*[^{]+)?\s*\{`)
	classes := classRegex.FindAllString(swiftStr, -1)

	// Extract protocol definitions
	protocolRegex := regexp.MustCompile(`(?:public\s+|private\s+|fileprivate\s+|internal\s+)?protocol\s+\w+(?::\s*[^{]+)?\s*\{`)
	protocols := protocolRegex.FindAllString(swiftStr, -1)

	// Extract enum definitions
	enumRegex := regexp.MustCompile(`(?:public\s+|private\s+|fileprivate\s+|internal\s+)?enum\s+\w+(?::\s*[^{]+)?\s*\{`)
	enums := enumRegex.FindAllString(swiftStr, -1)

	// Extract function signatures
	funcRegex := regexp.MustCompile(`(?:public\s+|private\s+|fileprivate\s+|internal\s+)?(?:static\s+|class\s+)?func\s+\w+\s*\([^)]*\)(?:\s*->\s*[^{]+)?\s*\{`)
	funcs := funcRegex.FindAllString(swiftStr, -1)

	// Extract extension blocks
	extensionRegex := regexp.MustCompile(`extension\s+\w+(?::\s*[^{]+)?\s*\{`)
	extensions := extensionRegex.FindAllString(swiftStr, -1)

	// Combine all extracted elements
	var chunks []string
	chunks = append(chunks, singleLineComments...)
	chunks = append(chunks, multiLineComments...)
	chunks = append(chunks, imports...)
	chunks = append(chunks, structs...)
	chunks = append(chunks, classes...)
	chunks = append(chunks, protocols...)
	chunks = append(chunks, enums...)
	chunks = append(chunks, funcs...)
	chunks = append(chunks, extensions...)

	if len(chunks) == 0 {
		return "// No relevant Swift elements found after compression.\n"
	}

	var result strings.Builder
	for _, chunk := range chunks {
		result.WriteString(strings.TrimSpace(chunk))
		result.WriteString("\n// -----\n")
	}

	return result.String()
}

// compressJava is a specialized function to compress Java content
func (gc *GenericCompressor) compressJava(content []byte) string {
	// Convert content to string for easier processing
	javaStr := string(content)

	// Extract Java comments
	singleLineCommentRegex := regexp.MustCompile(`(?m)//.*$`)
	singleLineComments := singleLineCommentRegex.FindAllString(javaStr, -1)

	multiLineCommentRegex := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	multiLineComments := multiLineCommentRegex.FindAllString(javaStr, -1)

	// Extract package declaration
	packageRegex := regexp.MustCompile(`package\s+[^;]+;`)
	packages := packageRegex.FindAllString(javaStr, -1)

	// Extract imports
	importRegex := regexp.MustCompile(`import\s+[^;]+;`)
	imports := importRegex.FindAllString(javaStr, -1)

	// Extract class definitions
	classRegex := regexp.MustCompile(`(?:public\s+|private\s+|protected\s+)?(?:abstract\s+|final\s+)?class\s+\w+(?:\s+extends\s+\w+)?(?:\s+implements\s+[^{]+)?\s*\{`)
	classes := classRegex.FindAllString(javaStr, -1)

	// Extract interface definitions
	interfaceRegex := regexp.MustCompile(`(?:public\s+|private\s+|protected\s+)?interface\s+\w+(?:\s+extends\s+[^{]+)?\s*\{`)
	interfaces := interfaceRegex.FindAllString(javaStr, -1)

	// Extract enum definitions
	enumRegex := regexp.MustCompile(`(?:public\s+|private\s+|protected\s+)?enum\s+\w+\s*\{`)
	enums := enumRegex.FindAllString(javaStr, -1)

	// Extract method signatures
	methodRegex := regexp.MustCompile(`(?:public\s+|private\s+|protected\s+)?(?:static\s+|final\s+|abstract\s+)*(?:<[^>]+>\s+)?(?:\w+(?:\[\])?\s+)?\w+\s*\([^)]*\)(?:\s+throws\s+[^{]+)?\s*\{`)
	methods := methodRegex.FindAllString(javaStr, -1)

	// Combine all extracted elements
	var chunks []string
	chunks = append(chunks, singleLineComments...)
	chunks = append(chunks, multiLineComments...)
	chunks = append(chunks, packages...)
	chunks = append(chunks, imports...)
	chunks = append(chunks, classes...)
	chunks = append(chunks, interfaces...)
	chunks = append(chunks, enums...)
	chunks = append(chunks, methods...)

	if len(chunks) == 0 {
		return "// No relevant Java elements found after compression.\n"
	}

	var result strings.Builder
	for _, chunk := range chunks {
		result.WriteString(strings.TrimSpace(chunk))
		result.WriteString("\n// -----\n")
	}

	return result.String()
}

// compressRust is a specialized function to compress Rust content
func (gc *GenericCompressor) compressRust(content []byte) string {
	// Convert content to string for easier processing
	rustStr := string(content)

	// Extract Rust comments
	singleLineCommentRegex := regexp.MustCompile(`(?m)//.*$`)
	singleLineComments := singleLineCommentRegex.FindAllString(rustStr, -1)

	multiLineCommentRegex := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	multiLineComments := multiLineCommentRegex.FindAllString(rustStr, -1)

	// Extract imports
	importRegex := regexp.MustCompile(`use\s+[^;]+;`)
	imports := importRegex.FindAllString(rustStr, -1)

	// Extract struct definitions
	structRegex := regexp.MustCompile(`(?:pub\s+)?struct\s+\w+\s*\{[^}]*\}`)
	structs := structRegex.FindAllString(rustStr, -1)

	// Extract trait definitions
	traitRegex := regexp.MustCompile(`(?:pub\s+)?trait\s+\w+\s*\{[^}]*\}`)
	traits := traitRegex.FindAllString(rustStr, -1)

	// Extract enum definitions
	enumRegex := regexp.MustCompile(`(?:pub\s+)?enum\s+\w+\s*\{[^}]*\}`)
	enums := enumRegex.FindAllString(rustStr, -1)

	// Extract function signatures
	funcRegex := regexp.MustCompile(`(?:pub\s+)?fn\s+\w+\s*\([^)]*\)(?:\s*->\s*[^{]+)?\s*\{`)
	funcs := funcRegex.FindAllString(rustStr, -1)

	// Extract impl blocks
	implRegex := regexp.MustCompile(`impl(?:\s+\w+)?\s+for\s+\w+\s*\{`)
	impls := implRegex.FindAllString(rustStr, -1)

	// Combine all extracted elements
	var chunks []string
	chunks = append(chunks, singleLineComments...)
	chunks = append(chunks, multiLineComments...)
	chunks = append(chunks, imports...)
	chunks = append(chunks, structs...)
	chunks = append(chunks, traits...)
	chunks = append(chunks, enums...)
	chunks = append(chunks, funcs...)
	chunks = append(chunks, impls...)

	if len(chunks) == 0 {
		return "// No relevant Rust elements found after compression.\n"
	}

	var result strings.Builder
	for _, chunk := range chunks {
		result.WriteString(strings.TrimSpace(chunk))
		result.WriteString("\n// -----\n")
	}

	return result.String()
}
