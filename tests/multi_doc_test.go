package koanf_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/require"
)

// failingProvider aborts while reading the raw bytes, simulating an input
// that is cancelled or errors out before parsing starts.
type failingProvider struct{}

func (failingProvider) ReadBytes() ([]byte, error) { return nil, fmt.Errorf("read cancelled") }
func (failingProvider) Read() (map[string]any, error) {
	return nil, fmt.Errorf("read cancelled")
}

// docOffsets returns the byte offsets of every `---` marker in the input.
func docOffsets(t *testing.T, in string, n int) []int {
	t.Helper()
	out := make([]int, 0, n)
	rest := in
	base := 0
	for range n {
		idx := strings.Index(rest, "---")
		require.NotEqual(t, -1, idx, "missing document marker")
		out = append(out, base+idx)
		base += idx + len("---")
		rest = in[base:]
	}
	return out
}

func TestMultiDocLoadMergesInOrder(t *testing.T) {
	in := "---\na: 1\nonly1: x\n---\na: 2\nonly2: y\n---\na: 3\nnested:\n  key: z\n"
	k := koanf.New(".")
	require.NoError(t, k.Load(rawbytes.Provider([]byte(in)), yaml.Parser()))

	// Later documents win for a shared key, documents stay in original order.
	require.Equal(t, 3, k.Get("a"))
	require.Equal(t, "x", k.Get("only1"))
	require.Equal(t, "y", k.Get("only2"))
	require.Equal(t, "z", k.Get("nested.key"))
	require.ElementsMatch(t, []string{"a", "only1", "only2", "nested.key"}, k.Keys())
}

func TestMultiDocLoadDefaultMergeAllowsTypeChanges(t *testing.T) {
	// Without StrictMerge, later documents behave like later Load() calls.
	in := "---\na: 1\n---\na: string\n"
	k := koanf.New(".")
	require.NoError(t, k.Load(rawbytes.Provider([]byte(in)), yaml.Parser()))
	require.Equal(t, "string", k.Get("a"))
}

func TestMultiDocStrictConflictRollback(t *testing.T) {
	in := "---\nparent:\n  value: fromdoc1\n  fresh: yes\n---\nparent:\n  value: 123\nshouldNotExist: true\n"
	starts := docOffsets(t, in, 2)

	k := koanf.NewWithConf(koanf.Conf{Delim: ".", StrictMerge: true})
	require.NoError(t, k.Load(confmap.Provider(map[string]any{
		"parent": map[string]any{"value": "baseline", "kept": true},
	}, "."), nil))

	snapshot := k.Raw()
	err := k.Load(rawbytes.Provider([]byte(in)), yaml.Parser())

	var de *koanf.DocError
	require.ErrorAs(t, err, &de)
	require.Equal(t, 1, de.Doc, "conflict must be attributed to the second document")
	require.Equal(t, koanf.SourceRange{Offset: starts[1], Length: len(in) - starts[1]}, de.Range)
	require.Equal(t, "parent.value", de.KeyPath, "key path must use the configured delimiter")
	require.Contains(t, err.Error(), "document 1")
	require.Contains(t, err.Error(), "bytes")
	require.Contains(t, err.Error(), "parent.value")

	var sme *maps.StrictMergeError
	require.ErrorAs(t, err, &sme)
	require.Equal(t, "parent.value", sme.Key)

	// Nothing from either document may remain after the failure.
	require.Equal(t, snapshot, k.Raw())
	require.False(t, k.Exists("parent.fresh"))
	require.False(t, k.Exists("shouldNotExist"))
	require.Equal(t, "baseline", k.Get("parent.value"))
}

func TestMultiDocStrictConflictWithCustomDelim(t *testing.T) {
	in := "---\nparent:\n  value: string\n---\nparent:\n  value: 123\n"
	k := koanf.NewWithConf(koanf.Conf{Delim: "/", StrictMerge: true})

	err := k.Load(rawbytes.Provider([]byte(in)), yaml.Parser())
	var de *koanf.DocError
	require.ErrorAs(t, err, &de)
	require.Equal(t, "parent/value", de.KeyPath)
}

func TestMultiDocParseFailureRollback(t *testing.T) {
	in := "---\nfromfirst: yes\n---\nbad: [unterminated\nfromsecond: no\n"
	starts := docOffsets(t, in, 2)

	k := koanf.New(".")
	err := k.Load(rawbytes.Provider([]byte(in)), yaml.Parser())

	var de *koanf.DocError
	require.ErrorAs(t, err, &de)
	require.Equal(t, 1, de.Doc)
	require.Equal(t, starts[1], de.Range.Offset)
	require.Empty(t, de.KeyPath, "syntax errors have no key path")
	require.Error(t, de.Err)

	// The first document must not leave any state behind.
	require.Empty(t, k.Keys())
	require.False(t, k.Exists("fromfirst"))
}

func TestMultiDocCustomMergeCalledPerDocument(t *testing.T) {
	in := "---\na: 1\n---\nb: 2\n---\nc: 3\n"

	var seen []string
	merge := func(src, dest map[string]any) error {
		for key := range src {
			seen = append(seen, key)
		}
		maps.Merge(src, dest)
		return nil
	}

	k := koanf.New(".")
	require.NoError(t, k.Load(rawbytes.Provider([]byte(in)), yaml.Parser(), koanf.WithMergeFunc(merge)))
	require.Equal(t, []string{"a", "b", "c"}, seen)
	require.Equal(t, 1, k.Get("a"))
	require.Equal(t, 2, k.Get("b"))
	require.Equal(t, 3, k.Get("c"))
}

func TestMultiDocCustomMergeErrorRollback(t *testing.T) {
	in := "---\na: 1\n---\nb: 2\n"
	starts := docOffsets(t, in, 2)

	boom := errors.New("merge refused")
	k := koanf.New(".")
	require.NoError(t, k.Load(confmap.Provider(map[string]any{"base": true}, "."), nil))
	before := k.Raw()

	err := k.Load(rawbytes.Provider([]byte(in)), yaml.Parser(), koanf.WithMergeFunc(
		func(src, dest map[string]any) error {
			if _, ok := src["b"]; ok {
				return boom
			}
			maps.Merge(src, dest)
			return nil
		}))

	var de *koanf.DocError
	require.ErrorAs(t, err, &de)
	require.Equal(t, 1, de.Doc)
	require.Equal(t, starts[1], de.Range.Offset)
	require.ErrorIs(t, err, boom)
	require.Equal(t, before, k.Raw())
	require.False(t, k.Exists("a"))
}

func TestMultiDocEmptyStream(t *testing.T) {
	k := koanf.New(".")
	require.NoError(t, k.Load(rawbytes.Provider(nil), yaml.Parser()))
	require.Empty(t, k.Keys())
}

func TestMultiDocProviderReadFailure(t *testing.T) {
	k := koanf.New(".")
	require.NoError(t, k.Load(confmap.Provider(map[string]any{"base": true}, "."), nil))
	before := k.Raw()

	err := k.Load(failingProvider{}, yaml.Parser())
	require.ErrorContains(t, err, "read cancelled")
	require.Equal(t, before, k.Raw())
}

func TestNonMultiParserKeepsSingleDocumentBehavior(t *testing.T) {
	p := json.Parser()
	require.False(t, p.SupportsMultiDoc())

	// A concatenated multi-object input is invalid JSON and must surface as
	// the plain single-document parser error, not as a *DocError.
	k := koanf.New(".")
	err := k.Load(rawbytes.Provider([]byte(`{"a":1}{"b":2}`)), p)
	require.Error(t, err)

	var de *koanf.DocError
	require.False(t, errors.As(err, &de), "single-document parsers must keep their original error shape")
	require.False(t, k.Exists("a"))
}
