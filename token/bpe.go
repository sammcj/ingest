package token

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/base64"
	"fmt"
	"io"
	"path"
	"strconv"

	"github.com/pkoukk/tiktoken-go"
)

// The vocabularies are embedded gzipped so that ingest never reaches out to
// OpenAI's CDN, and never writes to a cache directory, on first run.

//go:embed assets/o200k_base.tiktoken.gz
var o200kBaseGz []byte

//go:embed assets/cl100k_base.tiktoken.gz
var cl100kBaseGz []byte

// embeddedBPE serves tiktoken's vocabularies from the binary.
type embeddedBPE struct{}

func (embeddedBPE) LoadTiktokenBpe(blobpath string) (map[string]int, error) {
	var gzipped []byte
	switch path.Base(blobpath) {
	case "o200k_base.tiktoken":
		gzipped = o200kBaseGz
	case "cl100k_base.tiktoken":
		gzipped = cl100kBaseGz
	default:
		return nil, fmt.Errorf("no vocabulary bundled for %s", path.Base(blobpath))
	}
	return parseBPE(gzipped)
}

// parseBPE reads the tiktoken rank file format: one "<base64 token> <rank>"
// pair per line.
func parseBPE(gzipped []byte) (map[string]int, error) {
	zr, err := gzip.NewReader(bytes.NewReader(gzipped))
	if err != nil {
		return nil, fmt.Errorf("failed to open embedded vocabulary: %w", err)
	}
	defer zr.Close()

	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress embedded vocabulary: %w", err)
	}

	lines := bytes.Split(raw, []byte("\n"))
	ranks := make(map[string]int, len(lines))
	for i, line := range lines {
		line = bytes.TrimRight(line, "\r")
		if len(line) == 0 {
			continue
		}
		token, rankStr, found := bytes.Cut(line, []byte(" "))
		if !found {
			return nil, fmt.Errorf("malformed vocabulary at line %d", i+1)
		}
		decoded, err := base64.StdEncoding.DecodeString(string(token))
		if err != nil {
			return nil, fmt.Errorf("malformed token at line %d: %w", i+1, err)
		}
		rank, err := strconv.Atoi(string(rankStr))
		if err != nil {
			return nil, fmt.Errorf("malformed rank at line %d: %w", i+1, err)
		}
		ranks[string(decoded)] = rank
	}
	return ranks, nil
}

func init() {
	tiktoken.SetBpeLoader(embeddedBPE{})
}
