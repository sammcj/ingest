# Ingest

Ingest is a tool I've written to make my life easier when preparing content for LLMs.

It parses directories of plain text files, such as source code, into a single markdown file suitable for ingestion by AI/LLMs.

![ingest](screenshot2.png)

Ingest can also pass the prompt directly to an LLM such as Ollama for processing.

![ingest with --llm](screenshot.png)

## Features

- Traverse directory structures and generate a tree view
- Include/exclude files based on glob patterns
- Parse output directly to LLMs such as Ollama or any OpenAI compatible API for processing
- Generate and include git diffs and logs
- Count approximate tokens for LLM compatibility
- Customisable output templates
- Copy output to clipboard (when available)
- Export to file or print to console
- Optional JSON output
- Estimate VRAM requirements and check model compatibility using the vramestimator package

## Installation

### go install (recommended)

Make sure you have Go installed on your system, then run:

```shell
go install github.com/sammcj/ingest@HEAD
```

### Manual install

1. Download the latest release from the [releases page](https://github.com/sammcj/ingest/releases)
2. Move the binary to a directory in your PATH, e.g. `mv ingest* /usr/local/bin/ingest`

## Usage

Basic usage:

```shell
ingest [flags] <paths>
```

ingest will default to the current working directory if no path is provided, e.g:

```shell
$ ingest

⠋ Traversing directory and building tree...  [0s]
[ℹ️] Tokens (Approximate): 15,945
[✅] Copied to clipboard successfully.
```

Generate a prompt from a directory, including only Python files:

```shell
ingest -i "**/*.py" /path/to/project
```

Generate a prompt with git diff and copy to clipboard:

```shell
ingest -d /path/to/project
```

Generate a prompt for multiple files/directories:

```shell
ingest /path/to/project /path/to/other/project
```

Generate a prompt and save to a file:

```shell
ingest -o output.md /path/to/project
```

### VRAM Estimation and Model Compatibility

Ingest includes a feature to estimate VRAM requirements and check model compatibility using the [Gollama](https://github.com/sammcj/gollama)'s vramestimator package. This helps you determine if your generated content will fit within the specified model, VRAM, and quantization constraints.

To use this feature, add the following flags to your ingest command:

```shell
ingest --vram --model <model_id> [--memory <memory_in_gb>] [--quant <quantization>] [--context <context_length>] [--kvcache <kv_cache_quant>] [--quanttype <quant_type>] [other flags] <paths>
```

Examples:

Estimate VRAM usage for a specific context:

```shell
ingest --vram --model NousResearch/Hermes-2-Theta-Llama-3-8B --quant q4_k_m --context 2048 --kvcache q4_0 .
# Estimated VRAM usage: 5.35 GB
```

Calculate maximum context for a given memory constraint:

```shell
ingest --vram --model NousResearch/Hermes-2-Theta-Llama-3-8B --quant q4_k_m --memory 6 --kvcache q8_0 .
# Maximum context for 6.00 GB of memory: 5069
```

Find the best BPW (Bits Per Weight):

```shell
ingest --vram --model NousResearch/Hermes-2-Theta-Llama-3-8B --memory 6 --quanttype gguf .
# Best BPW for 6.00 GB of memory: IQ3_S
```

The tool also works for exl2 (ExllamaV2) models:

```shell
ingest --vram --model NousResearch/Hermes-2-Theta-Llama-3-8B --quant 5.0 --context 2048 --kvcache q4_0 . # For exl2 models
ingest --vram --model NousResearch/Hermes-2-Theta-Llama-3-8B --quant 5.0 --memory 6 --kvcache q8_0 . # For exl2 models
```

When using the VRAM estimation feature along with content generation, ingest will provide information about the generated content's compatibility with the specified constraints:

```shell
ingest --vram --model NousResearch/Hermes-2-Theta-Llama-3-8B --memory 8 --quant q4_0 .
⠋ Traversing directory and building tree... [0s]
[ℹ️] 14,702 Tokens (Approximate)
Maximum context for 8.00 GB of memory: 10240
Generated content (14,702 tokens) fits within maximum context.
Top 5 largest files (by estimated token count):
1. /Users/samm/git/sammcj/ingest/main.go (4,682 tokens)
2. /Users/samm/git/sammcj/ingest/filesystem/filesystem.go (2,694 tokens)
3. /Users/samm/git/sammcj/ingest/README.md (1,895 tokens)
4. /Users/samm/git/sammcj/ingest/utils/utils.go (948 tokens)
5. /Users/samm/git/sammcj/ingest/config/config.go (884 tokens)
[✅] Copied to clipboard successfully.
```

Available flags for VRAM estimation:

- `--vram`: Enable VRAM estimation and model compatibility check
- `--model`: Specify the model ID to check against (required for estimation)
- `--memory`: Specify the available memory in GB for context calculation (optional)
- `--quant`: Specify the quantization type (e.g., q4_k_m) or bits per weight (e.g., 5.0)
- `--context`: Specify the context length for VRAM estimation (optional)
- `--kvcache`: Specify the KV cache quantization (fp16, q8_0, or q4_0)
- `--quanttype`: Specify the quantization type (gguf or exl2)

Ingest will provide appropriate output based on the combination of flags used, such as estimating VRAM usage, calculating maximum context, or finding the best BPW. If the generated content fits within the specified constraints, you'll see a success message. Otherwise, you'll receive a warning that the content may not fit.

## LLM Integration

Ingest can pass the generated prompt to LLMs that have an OpenAI compatible API such as [Ollama](https://ollama.com) for processing.

```shell
ingest --llm /path/to/project
```

By default this will use any prompt suffix from your configuration file:

```shell
./ingest utils.go --llm
⠋ Traversing directory and building tree...  [0s]
This is Go code for a file named `utils.go`. It contains various utility functions for
handling terminal output, clipboard operations, and configuration directories.
...
```

You can provide a prompt suffix to append to the generated prompt:

```shell
ingest --llm -p "explain this code" /path/to/project
```

## Configuration

Ingest uses a configuration file located at `~/.config/ingest/config.json`.

You can make Ollama processing run without prompting setting `"llm_auto_run": true` in the config file.

The config file also contains:

- `llm_model`: The model to use for processing the prompt, e.g. "llama3.1:8b-q5_k_m".
- `llm_prompt_prefix`: An optional prefix to prepend to the prompt, e.g. "This is my application."
- `llm_prompt_suffix`: An optional suffix to append to the prompt, e.g. "explain this code"

Ingest uses the following directories for user-specific configuration:

- `~/.config/ingest/patterns/exclude`: Add .glob files here to exclude additional patterns.
- `~/.config/ingest/patterns/templates`: Add custom .tmpl files here for different output formats.

These directories will be created automatically on first run, along with README files explaining their purpose.

### Flags

- `-i, --include`: Patterns to include (can be used multiple times)
- `-e, --exclude`: Patterns to exclude (can be used multiple times)
- `--include-priority`: Include files in case of conflict between include and exclude patterns
- `--exclude-from-tree`: Exclude files/folders from the source tree based on exclude patterns
- `--tokens`: Display the token count of the generated prompt
- `-c, --encoding`: Optional tokeniser to use for token count
- `-o, --output`: Optional output file path
- `--llm`: Send the generated prompt to an OpenAI compatible LLM server (such as Ollama) for processing
- `-p, --prompt`: Optional prompt suffix to append to the generated prompt
- `-d, --diff`: Include git diff
- `--git-diff-branch`: Generate git diff between two branches
- `--git-log-branch`: Retrieve git log between two branches
- `-l, --line-number`: Add line numbers to the source code
- `--no-codeblock`: Disable wrapping code inside markdown code blocks
- `--relative-paths`: Use relative paths instead of absolute paths
- `-n, --no-clipboard`: Disable copying to clipboard
- `-t, --template`: Path to a custom Handlebars template
- `--json`: Print output as JSON
- `--pattern-exclude`: Path to a specific .glob file for exclude patterns
- `--print-default-excludes`: Print the default exclude patterns
- `--print-default-template`: Print the default template
- `--report`: Print the largest parsed files
- `--verbose`: Print verbose output
- `-V, --version`: Print the version number (WIP - still trying to get this to work nicely)

### Excludes

You can get a list of the default excludes by parsing `--print-default-excludes` to ingest.
These are defined in [defaultExcludes.go](https://github.com/sammcj/ingest/blob/main/filesystem/defaultExcludes.go).

To override the default excludes, create a `default.glob` file in `~/.config/ingest/patterns/exclude` with the patterns you want to exclude.

### Templates

Templates are written in standard [go templating syntax](https://pkg.go.dev/text/template).

You can get a list of the default templates by parsing `--print-default-template` to ingest.
These are defined in [template.go](https://github.com/sammcj/ingest/blob/main/template/template.go).

To override the default templates, create a `default.tmpl` file in `~/.config/ingest/patterns/templates` with the template you want to use by default.

## Contributing

Contributions are welcome, Please feel free to submit a Pull Request.

## License

- Copyright 2024 Sam McLeod
- This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgements

- Initially inspired by [mufeedvh/code2prompt](https://github.com/mufeedvh/code2prompt)
