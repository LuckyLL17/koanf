// Package kdl implements a koanf.Parser that parses KDL bytes as conf maps.
package kdl

import (
	"fmt"

	"github.com/knadh/koanf/v2"
	kdl "github.com/sblinch/kdl-go"
)

// KDL implements a KDL parser.
type KDL struct{}

// Parser returns a KDL Parser.
func Parser() *KDL {
	return &KDL{}
}

// Unmarshal parses the given KDL bytes.
func (p *KDL) Unmarshal(b []byte) (map[string]any, error) {
	var o map[string]any
	err := kdl.Unmarshal(b, &o)
	return o, err
}

// SupportsMultiDoc declares that a KDL input always contains a single
// document. KDL has no multi-document representation.
func (p *KDL) SupportsMultiDoc() bool { return false }

// UnmarshalDocs is not supported by the KDL parser.
func (p *KDL) UnmarshalDocs(b []byte) ([]koanf.DocFragment, error) {
	return nil, fmt.Errorf("kdl: %w", koanf.ErrMultiDocNotSupported)
}

// Marshal marshals the given config map to KDL bytes.
func (p *KDL) Marshal(o map[string]any) ([]byte, error) {
	return kdl.Marshal(o)
}
