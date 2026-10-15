package koanf

import (
	"errors"
	"strconv"
)

// Provider represents a configuration provider. Providers can
// read configuration from a source (file, HTTP etc.)
type Provider interface {
	// ReadBytes returns the entire configuration as raw []bytes to be parsed.
	// with a Parser.
	ReadBytes() ([]byte, error)

	// Read returns the parsed configuration as a nested map[string]any.
	// It is important to note that the string keys should not be flat delimited
	// keys like `parent.child.key`, but nested like `{parent: {child: {key: 1}}}`.
	Read() (map[string]any, error)
}

// Parser represents a configuration format parser.
type Parser interface {
	Unmarshal([]byte) (map[string]any, error)
	Marshal(map[string]any) ([]byte, error)
}

// SourceRange identifies the byte range a single document occupies
// inside the raw bytes returned by a Provider.
type SourceRange struct {
	// Offset is the zero-based byte offset at which the document starts.
	Offset int

	// Length is the document's length in bytes.
	Length int
}

// End returns the byte offset immediately after the last document byte.
func (r SourceRange) End() int {
	return r.Offset + r.Length
}

// String renders the range as a half-open byte interval, eg. bytes 10-24.
func (r SourceRange) String() string {
	return "bytes " + strconv.Itoa(r.Offset) + "-" + strconv.Itoa(r.End())
}

// DocFragment is one configuration document produced from a single
// Provider input that contains multiple documents, such as a YAML
// stream separated by `---` markers.
type DocFragment struct {
	// Doc is the zero-based position of the document in the input, in the
	// order in which the documents appeared.
	Doc int

	// Range is the byte range of the document within the raw input.
	Range SourceRange

	// Config is the nested config map produced by parsing the document.
	// It may be nil for empty documents such as an explicit trailing
	// YAML `---` with no content.
	Config map[string]any
}

// MultiDocParser is an optional interface implemented by Parsers that can
// split one Provider input into more than one configuration document.
//
// Parsers that do not support multiple documents keep the single-document
// behaviour of Unmarshal and must still declare it explicitly by returning
// false from SupportsMultiDoc. Koanf never calls UnmarshalDocs on a parser
// that returns false.
type MultiDocParser interface {
	Parser

	// SupportsMultiDoc declares whether the parser recognises multiple
	// documents in a single input.
	SupportsMultiDoc() bool

	// UnmarshalDocs parses b into an ordered slice of DocFragments. The
	// fragments must be returned in original input order with their document
	// index and source range set.
	//
	// A failure to parse a document must be returned wrapped as *DocError
	// (use NewDocError) so the document index, source range and cause are
	// available to the caller.
	UnmarshalDocs(b []byte) ([]DocFragment, error)
}

// ErrMultiDocNotSupported is returned by UnmarshalDocs on parsers that
// declared SupportsMultiDoc() == false. Koanf never invokes UnmarshalDocs
// on such parsers itself; the sentinel exists for callers that query the
// capability directly.
var ErrMultiDocNotSupported = errors.New("multi-document input is not supported by the parser")
