# Tree-sitter Based Code Compressor - Development Plan

## Overview

This document outlines the development plan for a Tree-sitter based code compressor. The goal is to extract key structural information from source code, such as imports, package definitions, function/method/class signatures, and comments, while omitting detailed implementation bodies.

## Phases

### Phase 1: Basic Language Identification & Parsing

- [x] Identify language from file extension.
- [x] Parse source code into an AST using Tree-sitter.
- [x] Implement basic Tree-sitter query for Go (imports, functions, methods, types).

**Status:** Completed.

### Phase 2: Code Chunk Extraction & Compression Logic

- [x] Define `CodeChunk` struct (Note: `OriginalLine` field is a placeholder, actual line mapping not yet implemented - see Phase 6).
- [x] Implement `GenericCompressor` with `Compress` method.
- [x] Execute queries and process captures from matches.
- [x] Implement logic to strip function/method bodies and retain signatures.
  - [x] Go
  - [x] Python
  - [x] JavaScript (for named functions, classes, methods, including exported and default exported named variants)
- [x] Handle different types of captures (imports, packages, type definitions, comments).
- [x] Sort and combine extracted code chunks.

**Status:** Largely completed. Core compression logic is functional for Go, Python, and key JavaScript constructs. All `internal/compressor` tests are passing.

### Phase 3: Language-Specific Queries & Refinements

- **Go:**
  - [x] Query for package, imports, type definitions, function declarations, method declarations, comments.
- **Python:**
  - [x] Query for imports, function definitions, class definitions, comments.
- **JavaScript:**
  - [x] Query for imports, comments, method definitions.
  - [x] Query and processing logic for:
    - [x] Exported named functions/classes/generators (e.g., `export function foo() {}`).
    - [x] Default exported named functions/classes/generators (e.g., `export default function foo() {}`).
    - [x] Standalone named functions/classes/generators.
  - [ ] **Outstanding/Needs Refinement for JavaScript:**
    - [ ] **Arrow Functions:** Implement body stripping for arrow functions assigned to variables (e.g., `const myArrow = () => { /* body */ };`). The test file `example.js` includes `const myArrowFunc = (a, b) => { /* Arrow function body */ ... }` which is currently not processed for body stripping.
    - [ ] **Anonymous Default Exports:** Clarify and potentially implement body stripping for anonymous default exported functions and classes (e.g., `export default function() { /* body */ }`). Currently, these are captured by `@export.other` and kept whole. The test file `example.js` includes `export default function() { console.log("Anon default func"); }` and `export default class { constructor() { this.x = 1;} }`, implying these might need stripping.
    - [ ] Review other common JS constructs (e.g., object methods not part of class syntax, IIFEs) if they need specific handling.
- **Other Languages:**
  - [ ] Plan for and implement queries for other languages as needed (e.g., TypeScript, Java, C++).

**Status:** Go and Python are well-covered by current tests. JavaScript has significantly improved and handles many common cases, with tests passing for implemented features. Specific outstanding items for JavaScript are noted above.

### Phase 4: Testing & Edge Cases

- [x] Write initial unit tests for Go, Python, JavaScript compressor logic. (Current tests for `internal/compressor` are passing).
- [ ] Expand test coverage with more complex real-world examples and edge cases for all supported languages, particularly for JavaScript features listed as outstanding in Phase 3.
- [ ] Test interactions between different JavaScript export/import syntaxes.

**Status:** Basic unit tests are in place and passing. Further testing is required, especially for JavaScript refinements.

### Phase 5: Integration & CLI

- [ ] Integrate the `GenericCompressor` into `main.go`.
- [ ] Add CLI flags for language selection, input file/directory, output options.
- [ ] Handle file system traversal for multiple files.

**Status:** Not started.

### Phase 6: Advanced Features (Future Considerations)

- [ ] **Line Number Mapping:** Fully implement `OriginalLine` in `CodeChunk` to map compressed chunks back to their original line numbers.
- [ ] Contextual Compression: Explore options for keeping more context if needed (e.g., call sites of a function, specific variable assignments).
- [ ] Configuration: Allow users to customize what to extract/omit via configuration files or advanced CLI options.
- [ ] Performance Optimization: Profile and optimize parsing and query execution for large codebases.

**Status:** Not started.

Languages supported by `smacker/go-tree-sitter`:

```
bash
c
cpp
csharp
css
cue
dockerfile
elixir
elm
golang
groovy
hcl
html
java
javascript
kotlin
lua
markdown
ocaml
php
protobuf
python
ruby
rust
scala
sql
svelte
swift
toml
typescript
markdown
yaml
```

The most important to support initially (if we even have to do anything to enable them?) are:

- Go
- Python
- JavaScript
- TypeScript
- bash
- rust
- swift
- toml
- yaml
- css
- c
- html
- sql
