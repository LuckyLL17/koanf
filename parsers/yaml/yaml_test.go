package yaml

import (
	"errors"
	"strings"
	"testing"

	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAML_Unmarshal(t *testing.T) {

	testCases := []struct {
		name   string
		input  []byte
		keys   []string
		values []any
		isErr  bool
	}{
		{
			name:  "Empty YAML",
			input: []byte(`{}`),
		},
		{
			name: "Valid YAML",
			input: []byte(`key: val
name: test
number: 2`),
			keys:   []string{"key", "name", "number"},
			values: []any{"val", "test", 2},
		},
		{
			name: "Invalid YAML - wrong indentation",
			input: []byte(`key: val
			name: test
			number: 2`),
			isErr: true,
		},
		{
			name: "Complex YAML - All types",
			input: []byte(`---
array:
- 1
- 2
- 3
boolean: true
color: gold
'null':
number: 123
object:
  a: b
  c: d
string: Hello World`),
			keys: []string{"array", "boolean", "color", "null", "number", "object", "string"},
			values: []any{[]any{1, 2, 3},
				true,
				"gold",
				nil,
				123,
				map[string]any{"a": "b", "c": "d"},
				"Hello World"},
		},
		{
			name: "Valid YAML - With comments",
			input: []byte(`---
key: #Here is a single-line comment
- value line 5
#Here is a
#multi-line comment
- value line 13`),
			keys:   []string{"key"},
			values: []any{[]any{"value line 5", "value line 13"}},
		},
	}

	y := Parser()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := y.Unmarshal(tc.input)
			if tc.isErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				for i, k := range tc.keys {
					v := out[k]
					assert.Equal(t, tc.values[i], v)
				}
			}
		})
	}
}

func TestYAML_SupportsMultiDoc(t *testing.T) {
	assert.True(t, Parser().SupportsMultiDoc())

	// The capability must be discoverable through the core interface.
	var p any = Parser()
	_, ok := p.(koanf.MultiDocParser)
	assert.True(t, ok)
}

func TestYAML_UnmarshalDocs(t *testing.T) {
	y := Parser()

	t.Run("explicit documents keep index, order and source range", func(t *testing.T) {
		in := []byte("---\na: 1\n---\nb: 2\n---\nc: 3\n")
		docs, err := y.UnmarshalDocs(in)
		require.NoError(t, err)
		require.Len(t, docs, 3)

		m1 := strings.Index(string(in), "---")
		m2 := strings.Index(string(in)[m1+1:], "---") + m1 + 1
		m3 := strings.Index(string(in)[m2+1:], "---") + m2 + 1

		assert.Equal(t, koanf.DocFragment{
			Doc:    0,
			Range:  koanf.SourceRange{Offset: m1, Length: m2 - m1},
			Config: map[string]any{"a": 1},
		}, docs[0])
		assert.Equal(t, koanf.DocFragment{
			Doc:    1,
			Range:  koanf.SourceRange{Offset: m2, Length: m3 - m2},
			Config: map[string]any{"b": 2},
		}, docs[1])
		assert.Equal(t, 2, docs[2].Doc)
		assert.Equal(t, m3, docs[2].Range.Offset)
		assert.Equal(t, len(in)-m3, docs[2].Range.Length)
		assert.Equal(t, map[string]any{"c": 3}, docs[2].Config)

		// Every range must point back at its own marker and bytes.
		for _, d := range docs {
			r := d.Range
			assert.True(t, strings.HasPrefix(string(in[r.Offset:r.End()]), "---"))
			assert.Equal(t, r.End(), r.Offset+r.Length)
		}
	})

	t.Run("implicit document before the first marker", func(t *testing.T) {
		in := []byte("a: 1\n---\nb: 2\n")
		docs, err := y.UnmarshalDocs(in)
		require.NoError(t, err)
		require.Len(t, docs, 2)

		assert.Equal(t, 0, docs[0].Doc)
		assert.Equal(t, 0, docs[0].Range.Offset)
		assert.Equal(t, map[string]any{"a": 1}, docs[0].Config)
		assert.Equal(t, 1, docs[1].Doc)
		assert.Equal(t, map[string]any{"b": 2}, docs[1].Config)
	})

	t.Run("comments and directives before the first marker are not a document", func(t *testing.T) {
		in := []byte("# leading comment\n---\na: 1\n")
		docs, err := y.UnmarshalDocs(in)
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, strings.Index(string(in), "---"), docs[0].Range.Offset)
		assert.Equal(t, map[string]any{"a": 1}, docs[0].Config)
	})

	t.Run("empty input has no documents", func(t *testing.T) {
		docs, err := y.UnmarshalDocs(nil)
		require.NoError(t, err)
		assert.Empty(t, docs)
	})

	t.Run("explicit trailing empty document is kept with a nil config", func(t *testing.T) {
		in := []byte("---\na: 1\n---\n")
		docs, err := y.UnmarshalDocs(in)
		require.NoError(t, err)
		require.Len(t, docs, 2)
		assert.Nil(t, docs[1].Config)
	})

	t.Run("single document without markers", func(t *testing.T) {
		in := []byte("a: 1\n")
		docs, err := y.UnmarshalDocs(in)
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, 0, docs[0].Doc)
		assert.Equal(t, 0, docs[0].Range.Offset)
		assert.Equal(t, len(in), docs[0].Range.Length)
	})

	t.Run("parse failure reports the failing document and range", func(t *testing.T) {
		in := []byte("---\na: 1\n---\nb: [untermined\n")
		docs, err := y.UnmarshalDocs(in)
		require.Error(t, err)
		assert.Nil(t, docs)

		var de *koanf.DocError
		require.True(t, errors.As(err, &de))
		assert.Equal(t, 1, de.Doc)
		assert.Equal(t, strings.Index(string(in)[len("---\na: 1\n"):], "---")+len("---\na: 1\n"), de.Range.Offset)
		assert.NotEmpty(t, de.Err.Error())
	})
}

func TestYAML_Marshal(t *testing.T) {
	testCases := []struct {
		name   string
		input  map[string]any
		output []byte
		isErr  bool
	}{
		{
			name:  "Empty YAML",
			input: map[string]any{},
			output: []byte(`{}
`),
		},
		{
			name: "Valid YAML",
			input: map[string]any{
				"key":    "val",
				"name":   "test",
				"number": 2,
			},
			output: []byte(`key: val
name: test
number: 2
`),
		},
		{
			name: "Complex YAML - All types",
			input: map[string]any{
				"array":   []any{1, 2, 3, 4, 5},
				"boolean": true,
				"color":   "gold",
				"null":    nil,
				"number":  123,
				"object":  map[string]any{"a": "b", "c": "d"},
				"string":  "Hello World",
			},
			output: []byte(`array:
    - 1
    - 2
    - 3
    - 4
    - 5
boolean: true
color: gold
"null": null
number: 123
object:
    a: b
    c: d
string: Hello World
`),
		},
	}

	y := Parser()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := y.Marshal(tc.input)
			if tc.isErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.output, out)
			}
		})
	}
}
