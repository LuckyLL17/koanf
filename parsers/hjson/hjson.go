// Package hjson implements a koanf.Parser that parses HJSON bytes as conf maps.
// Very similar to json.
package hjson

import (
	"fmt"

	"github.com/hjson/hjson-go/v4"
	"github.com/knadh/koanf/v2"
)

// HJSON implements a HJSON parser.
type HJSON struct{}

// Parser returns a HJSON parser.
func Parser() *HJSON {
	return &HJSON{}
}

// Unmarshal parses the given HJSON bytes.
func (p *HJSON) Unmarshal(b []byte) (map[string]any, error) {
	var out map[string]any
	if err := hjson.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SupportsMultiDoc declares that an HJSON input always contains a single
// document. Like JSON, HJSON has no multi-document representation.
func (p *HJSON) SupportsMultiDoc() bool { return false }

// UnmarshalDocs is not supported by the HJSON parser.
func (p *HJSON) UnmarshalDocs(b []byte) ([]koanf.DocFragment, error) {
	return nil, fmt.Errorf("hjson: %w", koanf.ErrMultiDocNotSupported)
}

// Marshal marshals the given config map to HJSON bytes.
func (p *HJSON) Marshal(o map[string]any) ([]byte, error) {
	return hjson.Marshal(o)
}
