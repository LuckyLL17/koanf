// Package json implements a koanf.Parser that parses JSON bytes as conf maps.
package json

import (
	"encoding/json"
	"fmt"

	"github.com/knadh/koanf/v2"
)

// JSON implements a JSON parser.
type JSON struct{}

// Parser returns a JSON Parser.
func Parser() *JSON {
	return &JSON{}
}

// Unmarshal parses the given JSON bytes.
func (p *JSON) Unmarshal(b []byte) (map[string]any, error) {
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SupportsMultiDoc declares that a JSON input always contains a single
// document. Multiple JSON values in one input are not supported.
func (p *JSON) SupportsMultiDoc() bool { return false }

// UnmarshalDocs is not supported by the JSON parser.
func (p *JSON) UnmarshalDocs(b []byte) ([]koanf.DocFragment, error) {
	return nil, fmt.Errorf("json: %w", koanf.ErrMultiDocNotSupported)
}

// Marshal marshals the given config map to JSON bytes.
func (p *JSON) Marshal(o map[string]any) ([]byte, error) {
	return json.Marshal(o)
}
