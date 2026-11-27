package releaseprovenance

import (
	"bytes"
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func actualPipxPublish(t *testing.T) ([]byte, PipxPublishIdentity) {
	t.Helper()
	read := func(name string) []byte {
		body, err := os.ReadFile(filepath.Join("testdata", "pipx-1.18.4", name))
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	var identity PipxPublishIdentity
	if err := json.Unmarshal(read("identity.json"), &identity); err != nil {
		t.Fatal(err)
	}
	return read("provenance.json"), identity
}

func pipxProofMutation(t *testing.T, proof []byte, fn func(map[string]any, map[string]any, map[string]any)) []byte {
	t.Helper()
	var outer map[string]any
	if err := json.Unmarshal(proof, &outer); err != nil {
		t.Fatal(err)
	}
	group := outer["attestation_bundles"].([]any)[0].(map[string]any)
	attestation := group["attestations"].([]any)[0].(map[string]any)
	fn(outer, group, attestation)
	body, err := json.Marshal(outer)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestPipxOfficialPublishProvenance(t *testing.T) {
	proof, identity := actualPipxPublish(t)
	receipt, err := VerifyPipxPublish(context.Background(), proof, identity)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Identity() != identity || receipt.ProvenanceSHA256() != uvDigest(proof) || receipt.Invocation() != pipxRepository+"/actions/runs/39104119239/attempts/1" {
		t.Fatalf("unexpected publish receipt: %+v", receipt)
	}
	converted, _, err := decodePipxPublish(proof, identity)
	if err != nil {
		t.Fatal(err)
	}
	_, haa, subjects := actualSignedRelease(t)
	if _, err := Verify(context.Background(), converted, haa, subjects); err == nil {
		t.Fatal("pipx proof gained HAA authority")
	}
	uv, uvIdentity := actualUVImage(t)
	if _, err := VerifyUVImage(context.Background(), converted, uv[1], uv[2], uv[3], uvIdentity); err == nil {
		t.Fatal("pipx proof gained UV authority")
	}
	_, pnpmIdentity := actualPNPMRelease(t)
	if _, err := VerifyPNPMRelease(context.Background(), converted, pnpmIdentity); err == nil {
		t.Fatal("pipx proof gained PNPM authority")
	}
}

func TestPipxPublishRejectsWrongIdentity(t *testing.T) {
	proof, identity := actualPipxPublish(t)
	for _, field := range []string{"version", "source", "run", "digest", "prerelease", "build", "prefix", "path", "zero-run", "upper-digest"} {
		t.Run(field, func(t *testing.T) {
			value := identity
			switch field {
			case "version":
				value.Version = "1.18.5"
			case "source":
				value.SourceCommit = strings.Repeat("a", 40)
			case "run":
				value.WorkflowRun = "39104119240"
			case "digest":
				value.WheelSHA256 = strings.Repeat("b", 64)
			case "prerelease":
				value.Version += "-rc1"
			case "build":
				value.Version += "+build"
			case "prefix":
				value.Version = "v" + value.Version
			case "path":
				value.Version = "../../escape"
			case "zero-run":
				value.WorkflowRun = "0"
			case "upper-digest":
				value.WheelSHA256 = strings.ToUpper(value.WheelSHA256)
			}
			receipt, err := VerifyPipxPublish(context.Background(), proof, value)
			if err == nil || receipt != (PipxPublishReceipt{}) {
				t.Fatal("wrong identity produced receipt")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if r, err := VerifyPipxPublish(ctx, proof, identity); !errors.Is(err, context.Canceled) || r != (PipxPublishReceipt{}) {
		t.Fatal("cancellation lost")
	}
	var absentContext context.Context
	if r, err := VerifyPipxPublish(absentContext, proof, identity); err == nil || r != (PipxPublishReceipt{}) {
		t.Fatal("nil context accepted")
	}
	for _, body := range func() [][]byte {
		haa, _, _ := actualSignedRelease(t)
		uv, _ := actualUVImage(t)
		pnpm, _ := actualPNPMRelease(t)
		return [][]byte{haa, uv[0], pnpm}
	}() {
		if r, err := VerifyPipxPublish(context.Background(), body, identity); err == nil || r != (PipxPublishReceipt{}) {
			t.Fatal("different upstream gained pipx authority")
		}
	}
}

func TestPipxPublishRejectsMalformedAndTamperedProof(t *testing.T) {
	proof, identity := actualPipxPublish(t)
	mutations := map[string]func(map[string]any, map[string]any, map[string]any){
		"outer-version":         func(o, g, a map[string]any) { o["version"] = 2 },
		"outer-unknown":         func(o, g, a map[string]any) { o["extra"] = true },
		"no-bundles":            func(o, g, a map[string]any) { o["attestation_bundles"] = []any{} },
		"many-bundles":          func(o, g, a map[string]any) { o["attestation_bundles"] = []any{g, g} },
		"no-attestations":       func(o, g, a map[string]any) { g["attestations"] = nil },
		"many-attestations":     func(o, g, a map[string]any) { g["attestations"] = []any{a, a} },
		"publisher-repo":        func(o, g, a map[string]any) { g["publisher"].(map[string]any)["repository"] = "evil/pipx" },
		"publisher-environment": func(o, g, a map[string]any) { g["publisher"].(map[string]any)["environment"] = "staging" },
		"publisher-case": func(o, g, a map[string]any) {
			p := g["publisher"].(map[string]any)
			p["Kind"] = p["kind"]
			delete(p, "kind")
		},
		"publisher-workflow":  func(o, g, a map[string]any) { g["publisher"].(map[string]any)["workflow"] = "other.yml" },
		"attestation-version": func(o, g, a map[string]any) { a["version"] = "1" },
		"no-log": func(o, g, a map[string]any) {
			a["verification_material"].(map[string]any)["transparency_entries"] = []any{}
		},
		"many-logs": func(o, g, a map[string]any) {
			m := a["verification_material"].(map[string]any)
			l := m["transparency_entries"].([]any)[0]
			m["transparency_entries"] = []any{l, l}
		},
		"changed-log": func(o, g, a map[string]any) {
			l := a["verification_material"].(map[string]any)["transparency_entries"].([]any)[0].(map[string]any)
			l["logIndex"] = "0"
		},
		"log-unknown": func(o, g, a map[string]any) {
			l := a["verification_material"].(map[string]any)["transparency_entries"].([]any)[0].(map[string]any)
			l["unknown"] = true
		},
		"bad-certificate": func(o, g, a map[string]any) { a["verification_material"].(map[string]any)["certificate"] = "e30=" },
		"certificate-bound": func(o, g, a map[string]any) {
			a["verification_material"].(map[string]any)["certificate"] = base64.StdEncoding.EncodeToString(make([]byte, (16<<10)+1))
		},
		"base64-newline": func(o, g, a map[string]any) {
			m := a["verification_material"].(map[string]any)
			m["certificate"] = m["certificate"].(string) + "\n"
		},
		"bad-signature": func(o, g, a map[string]any) { a["envelope"].(map[string]any)["signature"] = "AAAA" },
		"signature-bound": func(o, g, a map[string]any) {
			a["envelope"].(map[string]any)["signature"] = base64.StdEncoding.EncodeToString(make([]byte, 257))
		},
		"statement-bound": func(o, g, a map[string]any) {
			a["envelope"].(map[string]any)["statement"] = base64.StdEncoding.EncodeToString(make([]byte, (64<<10)+1))
		},
		"changed-signed-statement": func(o, g, a map[string]any) {
			e := a["envelope"].(map[string]any)
			b, err := base64.StdEncoding.DecodeString(e["statement"].(string))
			if err != nil {
				t.Fatal(err)
			}
			e["statement"] = base64.StdEncoding.EncodeToString(append(b, ' '))
		},
		"duplicate-signed-key": func(o, g, a map[string]any) {
			e := a["envelope"].(map[string]any)
			b, err := base64.StdEncoding.DecodeString(e["statement"].(string))
			if err != nil {
				t.Fatal(err)
			}
			b = bytes.Replace(b, []byte(`"_type":`), []byte(`"_type":"bad","_type":`), 1)
			e["statement"] = base64.StdEncoding.EncodeToString(b)
		},
	}
	cases := map[string][]byte{
		"empty": nil, "null": []byte("null"), "oversize": make([]byte, maxBundle+1),
		"trailing":          append(append([]byte{}, proof...), []byte("{}")...),
		"duplicate":         bytes.Replace(proof, []byte(`"version":`), []byte(`"version":1,"version":`), 1),
		"escaped-duplicate": bytes.Replace(proof, []byte(`"version":`), []byte(`"\u0076ersion":1,"version":`), 1),
		"case":              bytes.Replace(proof, []byte(`"version":`), []byte(`"Version":`), 1),
		"invalid-utf8":      append([]byte{'"', 0xff}, '"'),
	}
	for name, mutate := range mutations {
		cases[name] = pipxProofMutation(t, proof, mutate)
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if r, err := VerifyPipxPublish(context.Background(), body, identity); err == nil || r != (PipxPublishReceipt{}) {
				t.Fatal("unverified proof produced receipt")
			}
		})
	}
}

func TestPipxPublishJSONBoundsAndIdentity(t *testing.T) {
	_, identity := actualPipxPublish(t)
	body, err := json.Marshal(identity)
	if err != nil {
		t.Fatal(err)
	}
	for name, candidate := range map[string][]byte{
		"null": []byte("null"), "unknown": bytes.Replace(body, []byte("{"), []byte(`{"unknown":true,`), 1),
		"duplicate": bytes.Replace(body, []byte(`"version":`), []byte(`"version":"1.0.0","version":`), 1),
		"case":      bytes.Replace(body, []byte(`"version":`), []byte(`"Version":`), 1),
		"trailing":  append(append([]byte{}, body...), []byte("{}")...), "oversize": make([]byte, 4097),
	} {
		t.Run(name, func(t *testing.T) {
			value := identity
			err := json.Unmarshal(candidate, &value)
			if err == nil {
				t.Fatal("bad identity accepted")
			}
		})
	}
	if (PipxPublishIdentity{}).WheelFilename() != "" {
		t.Fatal("invalid filename derived")
	}
	for _, candidate := range []string{strings.Repeat("[", 14) + "0" + strings.Repeat("]", 14), "[" + strings.Repeat("0,", 8192) + "0]", `{"` + strings.Repeat("k", 129) + `":0}`, `{"a":1,"\u0061":2}`} {
		if boundPipxJSON([]byte(candidate), maxBundle) == nil {
			t.Fatal("JSON bound or duplicate bypass")
		}
	}
	for _, candidate := range []string{`"AB=="`, `"AA==\n"`, `null`, `""`, `"_w=="`} {
		if _, err := pipxBase64([]byte(candidate), 256); err == nil {
			t.Fatal("noncanonical base64 accepted")
		}
	}
}

func TestPipxPublishStatementSubset(t *testing.T) {
	_, identity := actualPipxPublish(t)
	statement := `{"_type":"https://in-toto.io/Statement/v1","subject":[{"name":"` + identity.WheelFilename() + `","digest":{"sha256":"` + identity.WheelSHA256 + `"}}],"predicateType":"` + pipxPublishPredicate + `"}`
	for _, predicate := range []string{"", `,"predicate":null`, `,"predicate":{}`} {
		if !validPipxStatement([]byte(strings.TrimSuffix(statement, "}")+predicate+"}"), identity) {
			t.Fatal("supported empty predicate refused")
		}
	}
	for _, pair := range [][2]string{{`"_type"`, `"_Type"`}, {`"subject"`, `"Subject"`}, {`"sha256"`, `"sha512"`}, {identity.WheelFilename(), "other.whl"}, {identity.WheelSHA256, strings.Repeat("a", 64)}, {pipxPublishPredicate, "https://slsa.dev/provenance/v1"}} {
		if validPipxStatement([]byte(strings.Replace(statement, pair[0], pair[1], 1)), identity) {
			t.Fatal("wrong signed subject/schema accepted")
		}
	}
	for _, predicate := range []string{`false`, `[]`, `{"unexpected":true}`, `""`} {
		if validPipxStatement([]byte(strings.TrimSuffix(statement, "}")+`,"predicate":`+predicate+"}"), identity) {
			t.Fatal("nonempty predicate accepted")
		}
	}
}

func TestPipxPublisherSignedClaims(t *testing.T) {
	proof, identity := actualPipxPublish(t)
	_, der, err := decodePipxPublish(proof, identity)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	if !validPipxPublisherClaims(cert) || validPipxPublisherClaims(nil) {
		t.Fatal("official signed claim guard differs")
	}
	for _, last := range []int{23, 24} {
		oid := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, last}
		for _, mode := range []string{"missing", "duplicate", "wrong-value", "printable-string", "trailing", "malformed", "compound"} {
			t.Run(oid.String()+"/"+mode, func(t *testing.T) {
				value := *cert
				value.Extensions = nil
				for _, extension := range cert.Extensions {
					if extension.Id.Equal(oid) {
						switch mode {
						case "missing":
							continue
						case "duplicate":
							value.Extensions = append(value.Extensions, extension)
						case "wrong-value":
							extension.Value = []byte{12, 1, 'x'}
						case "printable-string":
							extension.Value = append([]byte{}, extension.Value...)
							extension.Value[0] = 19
						case "trailing":
							extension.Value = append(append([]byte{}, extension.Value...), 0)
						case "malformed":
							extension.Value = []byte{12, 127}
						case "compound":
							extension.Value = append([]byte{}, extension.Value...)
							extension.Value[0] = 44
						}
					}
					value.Extensions = append(value.Extensions, pkix.Extension{Id: extension.Id, Value: extension.Value, Critical: extension.Critical})
				}
				if validPipxPublisherClaims(&value) {
					t.Fatal("unmatched signed claim accepted")
				}
			})
		}
	}
}
