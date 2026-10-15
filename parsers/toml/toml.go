// Package toml implements a koanf.Parser that parses TOML bytes as conf maps.
package toml

import (
	"fmt"

	"github.com/knadh/koanf/v2"
	"github.com/pelletier/go-toml/v2"
)

// TOML implements a TOML parser.
type TOML struct{}

// Parser returns a TOML Parser.
func Parser() *TOML {
	return &TOML{}
}

// Unmarshal parses the given TOML bytes.
func (p *TOML) Unmarshal(b []byte) (map[string]any, error) {
	var outMap map[string]any

	if err := toml.Unmarshal(b, &outMap); err != nil {
		return nil, err
	}

	if len(outMap) == 0 {
		return nil, nil
	}

	return outMap, nil
}

// SupportsMultiDoc declares that a TOML input always contains a single
// document. TOML has no multi-document representation.
func (p *TOML) SupportsMultiDoc() bool { return false }

// UnmarshalDocs is not supported by the TOML parser.
func (p *TOML) UnmarshalDocs(b []byte) ([]koanf.DocFragment, error) {
	return nil, fmt.Errorf("toml: %w", koanf.ErrMultiDocNotSupported)
}

// Marshal marshals the given config map to TOML bytes.
func (p *TOML) Marshal(o map[string]any) ([]byte, error) {
	if len(o) == 0 {
		return nil, nil
	}

	out, err := toml.Marshal(&o)
	if err != nil {
		return nil, err
	}

	return out, nil
}
