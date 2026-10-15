// Package huml implements a koanf.Parser that parses HUML bytes as conf maps.
package huml

import (
	"fmt"

	"github.com/huml-lang/go-huml"
	"github.com/knadh/koanf/v2"
)

// HUML implements a HUML parser.
type HUML struct{}

// Parser returns a HUML Parser.
func Parser() *HUML {
	return &HUML{}
}

// Unmarshal parses the given HUML bytes.
func (p *HUML) Unmarshal(b []byte) (map[string]any, error) {
	var result map[string]any
	err := huml.Unmarshal(b, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// SupportsMultiDoc declares that a HUML input always contains a single
// document.
func (p *HUML) SupportsMultiDoc() bool { return false }

// UnmarshalDocs is not supported by the HUML parser.
func (p *HUML) UnmarshalDocs(b []byte) ([]koanf.DocFragment, error) {
	return nil, fmt.Errorf("huml: %w", koanf.ErrMultiDocNotSupported)
}

// Marshal marshals the given config map to HUML bytes.
func (p *HUML) Marshal(o map[string]any) ([]byte, error) {
	return huml.Marshal(o)
}
