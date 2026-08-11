package token

import (
	"fmt"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/pkoukk/tiktoken-go"
	"github.com/sammcj/ingest/utils"
)

var (
	apiUsedOnce    sync.Once
	apiWarningOnce sync.Once

	encoderMu sync.Mutex
	encoders  = map[string]*tiktoken.Tiktoken{}
)

// Options controls how a token count is produced.
type Options struct {
	// Encoding is the offline vocabulary used when not counting via the API.
	Encoding string
	// Model is the model the count is intended for. It selects the Anthropic
	// API model and the offline correction factor.
	Model string
	// UseAPI counts with Anthropic's count_tokens endpoint instead of offline.
	UseAPI bool
	// NoCorrection reports the raw offline count with no correction applied.
	NoCorrection bool
}

// Correction factors applied to o200k_base counts. Anthropic has never
// published a Claude tokeniser, so offline counting approximates it.
//
// legacyCorrection was measured against Anthropic's count_tokens endpoint:
// o200k_base underestimates pre-Opus-4.7 Claude models by roughly 15%.
//
// Claude Opus 4.7 introduced a new tokeniser used by every model since,
// including Opus 5, Sonnet 5 and Fable 5. Anthropic documents it as producing
// approximately 30% more tokens for the same text, which compounds with the
// legacy factor. Both figures are approximations that vary with content; use
// --anthropic for an exact count.
const (
	legacyCorrection = 1.18
	modernCorrection = 1.53
)

// legacyTokeniserModels are the Claude models that predate the Opus 4.7
// tokeniser change. Anything else Claude-branded is assumed to be newer.
var legacyTokeniserModels = []string{
	"claude-2",
	"claude-instant",
	"claude-3",
	"claude-sonnet-4-",
	"claude-haiku-4-",
	"claude-opus-4-0",
	"claude-opus-4-1",
	"claude-opus-4-2", // claude-opus-4-20250514
	"claude-opus-4-5",
	"claude-opus-4-6",
}

// supportedEncodings maps ingest's encoding names onto bundled vocabularies.
var supportedEncodings = map[string]string{
	"o200k":                  "o200k_base",
	"o200k_base":             "o200k_base",
	"gpt-4o":                 "o200k_base",
	"gpt-4.1":                "o200k_base",
	"cl100k":                 "cl100k_base",
	"cl100k_base":            "cl100k_base",
	"llama3":                 "cl100k_base",
	"llama-3":                "cl100k_base",
	"gpt-4":                  "cl100k_base",
	"gpt-3.5-turbo":          "cl100k_base",
	"text-embedding-3-large": "cl100k_base",
	"text-embedding-3-small": "cl100k_base",
	"text-embedding-ada-002": "cl100k_base",
}

// GetTokenizer returns a cached tokeniser for the given encoding name,
// defaulting to o200k_base.
func GetTokenizer(encoding string) *tiktoken.Tiktoken {
	name, ok := supportedEncodings[strings.ToLower(encoding)]
	if !ok {
		name = "o200k_base"
	}

	encoderMu.Lock()
	defer encoderMu.Unlock()
	if tk, ok := encoders[name]; ok {
		return tk
	}

	tk, err := tiktoken.GetEncoding(name)
	if err != nil {
		fmt.Printf("Failed to get tokeniser: %v\n", err)
		return nil
	}
	encoders[name] = tk
	return tk
}

// CorrectionFactor returns the multiplier applied to an offline count to
// approximate the given model's own tokeniser. Non-Claude models are counted
// with their real vocabulary, so they need no correction.
func CorrectionFactor(model string) float64 {
	model = strings.ToLower(strings.TrimSpace(model))
	if !strings.Contains(model, "claude") {
		return 1.0
	}
	for _, legacy := range legacyTokeniserModels {
		if strings.HasPrefix(model, legacy) {
			return legacyCorrection
		}
	}
	return modernCorrection
}

// Describe summarises how a count was produced, for --json output.
func Describe(opts Options) string {
	if opts.UseAPI {
		return fmt.Sprintf("Anthropic count_tokens API (%s)", opts.model())
	}
	name, ok := supportedEncodings[strings.ToLower(opts.Encoding)]
	if !ok {
		name = "o200k_base"
	}
	if opts.NoCorrection {
		return fmt.Sprintf("%s, uncorrected", name)
	}
	return fmt.Sprintf("%s x%.2f, estimating %s", name, CorrectionFactor(opts.model()), opts.model())
}

func (o Options) model() string {
	if o.Model == "" {
		return DefaultModel
	}
	return o.Model
}

func (o Options) correct(raw int) int {
	if o.NoCorrection {
		return raw
	}
	return int(float64(raw) * CorrectionFactor(o.model()))
}

// CountTokens counts the tokens in rendered, via the Anthropic API when
// requested and falling back to the offline tokeniser on failure.
func CountTokens(rendered string, opts Options) int {
	if opts.UseAPI {
		count, err := CountTokensAPI(rendered, opts.model())
		if err != nil {
			warnAPIFallback(err)
		} else {
			noteAPIUse(opts.model())
			return count
		}
	}

	tk := GetTokenizer(opts.Encoding)
	if tk == nil {
		return 0
	}
	return opts.correct(len(tk.Encode(rendered, nil, nil)))
}

// CountTokensBatch counts tokens for multiple strings, using parallel API calls
// if the Anthropic API is enabled.
func CountTokensBatch(contents []string, opts Options) []int {
	if len(contents) == 0 {
		return []int{}
	}

	if opts.UseAPI {
		counts, err := CountTokensBatchAPI(contents, opts.model(), 4)
		if err != nil {
			warnAPIFallback(err)
		} else {
			noteAPIUse(opts.model())
			return counts
		}
	}

	results := make([]int, len(contents))
	tk := GetTokenizer(opts.Encoding)
	if tk == nil {
		return results
	}
	for i, content := range contents {
		results[i] = opts.correct(len(tk.Encode(content, nil, nil)))
	}
	return results
}

func warnAPIFallback(err error) {
	apiWarningOnce.Do(func() {
		fmt.Printf("Warning: Failed to count tokens using Anthropic API: %v\nFalling back to offline tokeniser\n", err)
	})
}

func noteAPIUse(model string) {
	apiUsedOnce.Do(func() {
		fmt.Println()
		utils.PrintColouredMessage("✓", fmt.Sprintf("Using Anthropic API (%s) for token counting", model), color.FgYellow)
	})
}
