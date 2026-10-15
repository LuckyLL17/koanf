// Package yaml implements a koanf.Parser that parses YAML bytes as conf maps.
package yaml

import (
	"bytes"

	"github.com/knadh/koanf/v2"
	"go.yaml.in/yaml/v3"
)

// YAML implements a YAML parser.
type YAML struct{}

// Parser returns a YAML Parser.
func Parser() *YAML {
	return &YAML{}
}

// Unmarshal parses the given YAML bytes.
func (p *YAML) Unmarshal(b []byte) (map[string]any, error) {
	var out map[string]any
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// SupportsMultiDoc declares that YAML inputs may contain several documents
// separated by `---` document-start markers.
func (p *YAML) SupportsMultiDoc() bool { return true }

// UnmarshalDocs parses the given YAML bytes into an ordered slice of
// documents. Each fragment carries its zero-based document index and the
// byte range it occupies in b. A syntax error in one document is returned
// as a *koanf.DocError so the failing document and its range are known.
func (p *YAML) UnmarshalDocs(b []byte) ([]koanf.DocFragment, error) {
	starts := yamlDocStarts(b)

	docs := make([]koanf.DocFragment, 0, len(starts))
	for i, start := range starts {
		end := len(b)
		if i+1 < len(starts) {
			end = starts[i+1]
		}
		r := koanf.SourceRange{Offset: start, Length: end - start}

		var out map[string]any
		if err := yaml.Unmarshal(b[start:end], &out); err != nil {
			return nil, koanf.NewDocError(i, r, err)
		}

		docs = append(docs, koanf.DocFragment{Doc: i, Range: r, Config: out})
	}

	return docs, nil
}

// yamlDocStarts returns the byte offsets at which YAML documents start.
//
// An explicit document starts at a column-0 line beginning with `---`
// (optionally followed by whitespace and a comment). Content before the
// first marker is the implicit first document. Whitespace, comments and
// directives before the first marker do not start a document. With no
// marker at all, non-empty input is a single implicit document and an
// empty input is a stream with no documents.
func yamlDocStarts(b []byte) []int {
	var (
		markers       []int
		contentBefore bool
		hasContent    bool
	)

	for off := 0; len(b) > 0; {
		nl := bytes.IndexByte(b, '\n')
		var line []byte
		var adv int
		if nl < 0 {
			line, adv = b, len(b)
		} else {
			line, adv = b[:nl], nl+1
		}

		switch {
		case isYAMLDocMarker(line):
			markers = append(markers, off)
		case !isYAMLLeadIn(line):
			hasContent = true
			// Real content appearing before the first explicit marker
			// means the input starts with an implicit document.
			if len(markers) == 0 {
				contentBefore = true
			}
		}

		b = b[adv:]
		off += adv
	}

	if len(markers) == 0 {
		// A marker-less input without any content is an empty stream
		// rather than an implicit document.
		if !hasContent {
			return nil
		}
		return []int{0}
	}

	starts := make([]int, 0, len(markers)+1)
	if contentBefore {
		starts = append(starts, 0)
	}
	return append(starts, markers...)
}

// isYAMLDocMarker reports whether line is a YAML document-start marker
// (`---`) at column zero, optionally followed only by whitespace/a comment.
func isYAMLDocMarker(line []byte) bool {
	line = bytes.TrimRight(line, "\r")
	if !bytes.HasPrefix(line, []byte("---")) {
		return false
	}
	rest := bytes.TrimSpace(line[3:])
	return len(rest) == 0 || rest[0] == '#'
}

// isYAMLLeadIn reports whether a line before the first marker is allowed
// outside a document: a blank line, a comment or a `%` directive.
func isYAMLLeadIn(line []byte) bool {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return true
	}
	return line[0] == '#' || line[0] == '%'
}

// Marshal marshals the given config map to YAML bytes.
func (p *YAML) Marshal(o map[string]any) ([]byte, error) {
	return yaml.Marshal(o)
}
