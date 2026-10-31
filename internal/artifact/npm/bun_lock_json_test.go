package npm

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestBunLockJSONPreservesActualNativeSamples(t *testing.T) {
	for name, packages := range map[string]int{"single": 1, "transitive": 2, "peer": 4} {
		t.Run(name, func(t *testing.T) {
			body, err := os.ReadFile("testdata/bun-lock/" + name + ".bun.lock")
			if err != nil {
				t.Fatal(err)
			}
			original := bytes.Clone(body)
			normalized, err := normalizeBunLockJSON(body)
			if err != nil || !bytes.Equal(body, original) || len(normalized) != len(body) {
				t.Fatal("actual native JSONC failed or input mutated", err)
			}
			var wire struct {
				Version  int                        `json:"lockfileVersion"`
				Config   int                        `json:"configVersion"`
				Packages map[string]json.RawMessage `json:"packages"`
			}
			if json.Unmarshal(normalized, &wire) != nil || wire.Version != 2 || wire.Config != 1 || len(wire.Packages) != packages {
				t.Fatal("native lock semantic content changed")
			}
			for i, b := range body {
				if (b == '\n' || b == '\r') && normalized[i] != b {
					t.Fatal("source positions changed")
				}
			}
		})
	}
}

func TestBunLockJSONCommentsCommasAndEscapedStrings(t *testing.T) {
	body := []byte("// first\r\n{\n" + `"url":"https://example.test/*literal*/[,]",` + "/* block\ncomment */" + `"escaped":"\"//literal\\", "unicode":"\uD83D\uDE00", "values":[true,false,null,1,"x",{},[],],` + "\n}\n// last")
	original := bytes.Clone(body)
	normalized, err := normalizeBunLockJSON(body)
	if err != nil || !bytes.Equal(body, original) || len(body) != len(normalized) {
		t.Fatal("supported lexical subset failed", err)
	}
	var value map[string]any
	if json.Unmarshal(normalized, &value) != nil || value["url"] != "https://example.test/*literal*/[,]" || value["escaped"] != "\"//literal\\" || value["unicode"] != "😀" || len(value["values"].([]any)) != 7 {
		t.Fatal("string or value data was rewritten")
	}
	normalized[0] = 'x'
	if !bytes.Equal(body, original) {
		t.Fatal("normalized output aliases caller data")
	}
}

func TestBunLockJSONRejectsAmbiguityAndGrammarRepair(t *testing.T) {
	cases := map[string][]byte{
		"empty":             {},
		"invalid-utf8":      {'{', '"', 0xff, '"', ':', '1', '}'},
		"duplicate":         []byte(`{"a":1,"a":2}`),
		"escaped-duplicate": []byte(`{"a":1,"\u0061":2}`),
		"object-comma":      []byte(`{,}`),
		"array-comma":       []byte(`[,]`),
		"missing-value":     []byte(`{"a":,}`),
		"double-comma":      []byte(`[1,,]`),
		"wrong-close":       []byte(`{"a":[1,}`),
		"trailing-comma":    []byte(`{},`),
		"trailing-value":    []byte(`{} {}`),
		"unclosed-comment":  []byte(`{} /* unfinished`),
		"nested-comment":    []byte(`/* outer /* inner */ trailing */ {}`),
		"bad-escape":        []byte(`{"a":"\q"}`),
		"short-unicode":     []byte(`{"a":"\u123"}`),
		"bad-hex":           []byte(`{"a":"\u12xz"}`),
		"unpaired-high":     []byte(`{"a":"\ud800"}`),
		"unpaired-low":      []byte(`{"a":"\udfff"}`),
		"high-followed-bmp": []byte(`{"a":"\ud800\u0041"}`),
		"unclosed-string":   []byte(`{"a":"// no close}`),
		"literal-control":   []byte("{\"a\":\"line\nfeed\"}"),
		"token-bound":       []byte("[" + strings.Repeat("0,", 200000) + "0]"),
		"depth-bound":       []byte(strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34)),
		"byte-bound":        bytes.Repeat([]byte{' '}, ProjectControlLimit+1),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			original := bytes.Clone(body)
			value, err := normalizeBunLockJSON(body)
			if err == nil || value != nil || !bytes.Equal(body, original) {
				t.Fatal("invalid input was repaired, exposed partial output or mutated")
			}
		})
	}
}
