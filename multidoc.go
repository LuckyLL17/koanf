package koanf

import (
	"errors"
	"fmt"
	"strings"

	"github.com/knadh/koanf/maps"
)

// DocError wraps any error that occurred while parsing or merging one
// document of a multi-document input. It carries the zero-based document
// index, the document's byte range inside the Provider input and, when the
// failure can be pinned to a config key, the delimited key path.
type DocError struct {
	// Doc is the zero-based index of the failing document.
	Doc int

	// Range is the byte range of the failing document in the raw input.
	Range SourceRange

	// KeyPath is the delimited key path at which the failure occurred. It
	// is empty for failures that cannot be attributed to a single key,
	// such as syntax errors while parsing.
	KeyPath string

	// Err is the underlying error.
	Err error
}

// NewDocError wraps err with the document index and source range.
func NewDocError(doc int, r SourceRange, err error) *DocError {
	return &DocError{Doc: doc, Range: r, Err: err}
}

// Error implements the error interface.
func (e *DocError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "document %d (%s)", e.Doc, e.Range)
	if e.KeyPath != "" {
		fmt.Fprintf(&b, " key %q", e.KeyPath)
	}
	fmt.Fprintf(&b, ": %v", e.Err)
	return b.String()
}

// Unwrap returns the wrapped error so errors.Is and errors.As can reach
// the underlying parser or merge error.
func (e *DocError) Unwrap() error {
	return e.Err
}

// mergeDocuments merges several parsed documents into the config.
//
// The documents are processed in original order against a private staged
// copy of the current config: each document goes through key normalization,
// the strict-merge type check (when enabled) or the custom merge function.
// The flattened maps are only rebuilt and the live config is only swapped
// in once every document succeeded, so a parse failure, a conflict in a
// later document or a cancelled Provider input leaves no state behind.
func (ko *Koanf) mergeDocuments(docs []DocFragment, opts *options) error {
	ko.mu.Lock()

	staged := maps.Copy(ko.confMap)

	for _, doc := range docs {
		c := doc.Config
		maps.IntfaceKeysToStrings(c)

		if opts.merge != nil {
			// Release the lock while the custom merge function runs so it
			// can safely call ko.Get*() methods, as merge() does. The live
			// config is never touched, so an error here simply discards
			// the staged copy.
			ko.mu.Unlock()
			err := opts.merge(c, staged)
			ko.mu.Lock()
			if err != nil {
				ko.mu.Unlock()
				return NewDocError(doc.Doc, doc.Range, err)
			}

			continue
		}

		if ko.conf.StrictMerge {
			if err := maps.MergeStrictWithDelim(c, staged, ko.conf.Delim); err != nil {
				ko.mu.Unlock()
				return ko.newDocMergeError(doc, err)
			}

			continue
		}

		maps.Merge(c, staged)
	}

	// All documents applied cleanly. Flatten once and commit atomically.
	ko.confMap = staged
	ko.confMapFlat, ko.keyMap = maps.Flatten(staged, nil, ko.conf.Delim)
	ko.keyMap = populateKeyParts(ko.keyMap, ko.conf.Delim)

	ko.mu.Unlock()
	return nil
}

// newDocMergeError wraps a strict-merge failure with the document metadata
// and the delimited key path reported by the merge.
func (ko *Koanf) newDocMergeError(doc DocFragment, err error) error {
	de := &DocError{Doc: doc.Doc, Range: doc.Range, Err: err}

	var sme *maps.StrictMergeError
	if errors.As(err, &sme) {
		de.KeyPath = sme.Key
	}

	return de
}
