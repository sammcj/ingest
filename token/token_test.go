package token

import (
	"strings"
	"testing"
)

// The embedded vocabularies must produce the same encodings as tiktoken itself,
// with no network access and no cache directory.
func TestEmbeddedVocabularies(t *testing.T) {
	t.Setenv("TIKTOKEN_CACHE_DIR", "/nonexistent")

	cases := []struct {
		encoding string
		text     string
		want     []int
	}{
		// Reference vectors from OpenAI's tiktoken.
		{"cl100k", "hello world", []int{15339, 1917}},
		{"cl100k", "tiktoken is great!", []int{83, 1609, 5963, 374, 2294, 0}},
		{"o200k", "hello world", []int{24912, 2375}},
	}

	for _, tc := range cases {
		tk := GetTokenizer(tc.encoding)
		if tk == nil {
			t.Fatalf("%s: no tokeniser returned", tc.encoding)
		}
		got := tk.Encode(tc.text, nil, nil)
		if len(got) != len(tc.want) {
			t.Fatalf("%s: encoded %q to %v, want %v", tc.encoding, tc.text, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("%s: encoded %q to %v, want %v", tc.encoding, tc.text, got, tc.want)
			}
		}
	}
}

// Round-tripping catches a vocabulary that parsed without error but decoded the
// base64 token bytes incorrectly.
func TestRoundTrip(t *testing.T) {
	text := "func main() { fmt.Println(\"g'day, 世界 🦘\") }"
	for _, encoding := range []string{"o200k", "cl100k"} {
		tk := GetTokenizer(encoding)
		if tk == nil {
			t.Fatalf("%s: no tokeniser returned", encoding)
		}
		if got := tk.Decode(tk.Encode(text, nil, nil)); got != text {
			t.Errorf("%s: round trip produced %q, want %q", encoding, got, text)
		}
	}
}

func TestUnknownEncodingFallsBackToO200k(t *testing.T) {
	text := "the quick brown fox"
	want := GetTokenizer("o200k").Encode(text, nil, nil)
	got := GetTokenizer("p50k").Encode(text, nil, nil)
	if len(got) != len(want) {
		t.Errorf("unknown encoding produced %d tokens, want %d from o200k", len(got), len(want))
	}
}

func TestCorrectionFactor(t *testing.T) {
	cases := []struct {
		model string
		want  float64
	}{
		{"claude-opus-5", modernCorrection},
		{"claude-fable-5", modernCorrection},
		{"claude-sonnet-5", modernCorrection},
		{"claude-opus-4-7", modernCorrection},
		{"claude-opus-4-8", modernCorrection},
		{"claude-mythos-5", modernCorrection},
		{"Claude-Opus-5", modernCorrection},
		{"claude-sonnet-4-5", legacyCorrection},
		{"claude-haiku-4-5-20251001", legacyCorrection},
		{"claude-opus-4-1", legacyCorrection},
		{"claude-opus-4-20250514", legacyCorrection},
		{"claude-opus-4-6", legacyCorrection},
		{"claude-3-5-sonnet-20241022", legacyCorrection},
		{"gpt-4o", 1.0},
		{"", 1.0}, // an empty model string is not Claude-branded
	}

	for _, tc := range cases {
		if got := CorrectionFactor(tc.model); got != tc.want {
			t.Errorf("CorrectionFactor(%q) = %v, want %v", tc.model, got, tc.want)
		}
	}
}

func TestDefaultModelUsesModernTokeniser(t *testing.T) {
	if got := CorrectionFactor(DefaultModel); got != modernCorrection {
		t.Errorf("default model %q has factor %v, want %v", DefaultModel, got, modernCorrection)
	}
}

func TestCountTokensAppliesCorrection(t *testing.T) {
	text := strings.Repeat("the quick brown fox jumps over the lazy dog. ", 20)

	raw := CountTokens(text, Options{Encoding: "o200k", NoCorrection: true})
	if raw == 0 {
		t.Fatal("raw count is zero")
	}

	opus5 := CountTokens(text, Options{Encoding: "o200k", Model: "claude-opus-5"})
	if want := int(float64(raw) * modernCorrection); opus5 != want {
		t.Errorf("claude-opus-5 count = %d, want %d", opus5, want)
	}

	sonnet45 := CountTokens(text, Options{Encoding: "o200k", Model: "claude-sonnet-4-5"})
	if want := int(float64(raw) * legacyCorrection); sonnet45 != want {
		t.Errorf("claude-sonnet-4-5 count = %d, want %d", sonnet45, want)
	}

	if opus5 <= sonnet45 {
		t.Errorf("Claude 5 count (%d) should exceed the legacy count (%d)", opus5, sonnet45)
	}
}

func TestCountTokensBatchMatchesIndividualCounts(t *testing.T) {
	contents := []string{"one", "two two", "three three three"}
	opts := Options{Encoding: "o200k", Model: DefaultModel}

	batch := CountTokensBatch(contents, opts)
	if len(batch) != len(contents) {
		t.Fatalf("batch returned %d counts, want %d", len(batch), len(contents))
	}
	for i, content := range contents {
		if want := CountTokens(content, opts); batch[i] != want {
			t.Errorf("batch[%d] = %d, want %d", i, batch[i], want)
		}
	}

	if got := CountTokensBatch(nil, opts); len(got) != 0 {
		t.Errorf("empty batch returned %v", got)
	}
}

func TestDescribe(t *testing.T) {
	cases := []struct {
		opts     Options
		contains []string
	}{
		{Options{Encoding: "o200k", Model: "claude-opus-5"}, []string{"o200k_base", "1.53", "claude-opus-5"}},
		{Options{Encoding: "o200k", NoCorrection: true}, []string{"o200k_base", "uncorrected"}},
		{Options{UseAPI: true}, []string{"count_tokens", DefaultModel}},
	}

	for _, tc := range cases {
		got := Describe(tc.opts)
		for _, want := range tc.contains {
			if !strings.Contains(got, want) {
				t.Errorf("Describe(%+v) = %q, want it to contain %q", tc.opts, got, want)
			}
		}
	}
}
