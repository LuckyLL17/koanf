package nestedtext_test

import (
	"errors"
	"testing"

	"github.com/knadh/koanf/parsers/nestedtext"
	"github.com/knadh/koanf/v2"
)

func TestParserMultiDocUnsupported(t *testing.T) {
	p := nestedtext.Parser()
	if p.SupportsMultiDoc() {
		t.Fatal("NestedText parser must declare multi-document support as false")
	}
	if _, err := p.UnmarshalDocs([]byte("key: value")); !errors.Is(err, koanf.ErrMultiDocNotSupported) {
		t.Fatalf("expected ErrMultiDocNotSupported, got %v", err)
	}
}

func TestParser(t *testing.T) {
	// test input in NestedText format
	ntsource := `ports:
  - 6483
  - 8020
  - 9332
timeout: 20
`
	// Test decoder
	nt := nestedtext.Parser()
	c, err := nt.Unmarshal([]byte(ntsource))
	if err != nil {
		t.Fatal("Unmarshal of NestedText input failed")
	}
	timeout := c["timeout"]
	if timeout != "20" {
		t.Logf("config-tree: %#v", c)
		t.Errorf("expected timeout-parameter to be 20, is %q", timeout)
	}

	// test encoder
	out, err := nt.Marshal(c)
	if err != nil {
		t.Fatal("Marshal of config to NestedText failed")
	}
	if string(out) != ntsource {
		t.Logf("config-text: %q", string(out))
		t.Errorf("expected output of Marshal(…) to equal input to Unmarshal(…); didn't")
	}
}
