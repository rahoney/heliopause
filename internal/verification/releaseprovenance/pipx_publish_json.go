package releaseprovenance

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

var errPipxJSON = errors.New("pipx publish JSON is invalid or exceeds bounds")

// This decoder is deliberately scoped to the supported PEP 740 publish subset.
// It runs before maps/protobuf decoding, including on the decoded DSSE payload.
func boundPipxJSON(body []byte, limit int) error {
	if len(body) == 0 || len(body) > limit || !utf8.Valid(body) {
		return errPipxJSON
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.UseNumber()
	remaining := 8192
	var walk func(int) error
	walk = func(depth int) error {
		remaining--
		if depth > 12 || remaining < 0 {
			return errPipxJSON
		}
		token, err := d.Token()
		if err != nil {
			return errPipxJSON
		}
		if delimiter, ok := token.(json.Delim); ok {
			switch delimiter {
			case '{':
				seen := make(map[string]bool)
				for d.More() {
					keyToken, err := d.Token()
					key, ok := keyToken.(string)
					remaining--
					if err != nil || !ok || len(key) > 128 || seen[key] || remaining < 0 {
						return errPipxJSON
					}
					seen[key] = true
					if err := walk(depth + 1); err != nil {
						return err
					}
				}
				end, err := d.Token()
				if err != nil || end != json.Delim('}') {
					return errPipxJSON
				}
			case '[':
				for d.More() {
					if err := walk(depth + 1); err != nil {
						return err
					}
				}
				end, err := d.Token()
				if err != nil || end != json.Delim(']') {
					return errPipxJSON
				}
			default:
				return errPipxJSON
			}
		}
		return nil
	}
	if err := walk(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errPipxJSON
	}
	return nil
}

// All listed fields are required, exact-case and exclusive. The caller has
// already bounded the full document and rejected duplicate decoded keys.
func pipxObject(body []byte, keys ...string) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if json.Unmarshal(body, &object) != nil || object == nil || len(object) != len(keys) {
		return nil, errPipxJSON
	}
	for _, key := range keys {
		if _, found := object[key]; !found {
			return nil, errPipxJSON
		}
	}
	return object, nil
}

func pipxOne(body []byte) (json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(body))
	start, err := d.Token()
	if err != nil || start != json.Delim('[') || !d.More() {
		return nil, errPipxJSON
	}
	var value json.RawMessage
	if d.Decode(&value) != nil || d.More() {
		return nil, errPipxJSON
	}
	end, err := d.Token()
	if err != nil || end != json.Delim(']') {
		return nil, errPipxJSON
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, errPipxJSON
	}
	return value, nil
}

func pipxString(body []byte) string {
	var value string
	if json.Unmarshal(body, &value) != nil {
		return ""
	}
	return value
}

func pipxBase64(body []byte, limit int) ([]byte, error) {
	value := pipxString(body)
	if value == "" || len(value) > base64.StdEncoding.EncodedLen(limit) {
		return nil, errPipxJSON
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(value)
	if err != nil || len(decoded) == 0 || len(decoded) > limit || base64.StdEncoding.EncodeToString(decoded) != value {
		return nil, errPipxJSON
	}
	return decoded, nil
}

func decodePipxPublish(body []byte, expected PipxPublishIdentity) ([]byte, []byte, error) {
	if err := boundPipxJSON(body, maxBundle); err != nil {
		return nil, nil, err
	}
	outer, err := pipxObject(body, "version", "attestation_bundles")
	if err != nil || string(outer["version"]) != "1" {
		return nil, nil, errPipxJSON
	}
	groupBody, err := pipxOne(outer["attestation_bundles"])
	if err != nil {
		return nil, nil, err
	}
	group, err := pipxObject(groupBody, "publisher", "attestations")
	if err != nil {
		return nil, nil, err
	}
	publisher, err := pipxObject(group["publisher"], "kind", "repository", "workflow", "environment")
	if err != nil || pipxString(publisher["kind"]) != "GitHub" || pipxString(publisher["repository"]) != "pypa/pipx" || pipxString(publisher["workflow"]) != "release.yml" || pipxString(publisher["environment"]) != "release" {
		return nil, nil, errPipxJSON
	}
	attestationBody, err := pipxOne(group["attestations"])
	if err != nil {
		return nil, nil, err
	}
	attestation, err := pipxObject(attestationBody, "version", "verification_material", "envelope")
	if err != nil || string(attestation["version"]) != "1" {
		return nil, nil, errPipxJSON
	}
	material, err := pipxObject(attestation["verification_material"], "certificate", "transparency_entries")
	if err != nil {
		return nil, nil, err
	}
	cert, err := pipxBase64(material["certificate"], 16<<10)
	if err != nil {
		return nil, nil, err
	}
	entry, err := pipxOne(material["transparency_entries"])
	if err != nil {
		return nil, nil, err
	}
	envelope, err := pipxObject(attestation["envelope"], "statement", "signature")
	if err != nil {
		return nil, nil, err
	}
	statement, err := pipxBase64(envelope["statement"], 64<<10)
	if err != nil || !validPipxStatement(statement, expected) {
		return nil, nil, errPipxJSON
	}
	if _, err := pipxBase64(envelope["signature"], 256); err != nil {
		return nil, nil, err
	}
	// RawMessages preserve the exact certificate, statement, signature and log
	// entry values. No upstream payload is reserialized or resigned.
	converted, err := json.Marshal(map[string]any{
		"mediaType": "application/vnd.dev.sigstore.bundle.v0.3+json",
		"verificationMaterial": map[string]any{
			"certificate": map[string]any{"rawBytes": material["certificate"]},
			"tlogEntries": []json.RawMessage{entry},
		},
		"dsseEnvelope": map[string]any{
			"payloadType": "application/vnd.in-toto+json", "payload": envelope["statement"],
			"signatures": []any{map[string]any{"sig": envelope["signature"]}},
		},
	})
	return converted, cert, err
}

func validPipxStatement(body []byte, expected PipxPublishIdentity) bool {
	if boundPipxJSON(body, 64<<10) != nil {
		return false
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(body, &raw) != nil || raw == nil {
		return false
	}
	if predicate, found := raw["predicate"]; found {
		trimmed := bytes.TrimSpace(predicate)
		if !bytes.Equal(trimmed, []byte("null")) {
			if _, err := pipxObject(predicate); err != nil {
				return false
			}
		}
		delete(raw, "predicate")
	}
	if len(raw) != 3 || pipxString(raw["_type"]) != "https://in-toto.io/Statement/v1" || pipxString(raw["predicateType"]) != pipxPublishPredicate {
		return false
	}
	subjectBody, err := pipxOne(raw["subject"])
	if err != nil {
		return false
	}
	subject, err := pipxObject(subjectBody, "name", "digest")
	if err != nil || pipxString(subject["name"]) != expected.WheelFilename() {
		return false
	}
	digest, err := pipxObject(subject["digest"], "sha256")
	return err == nil && pipxString(digest["sha256"]) == expected.WheelSHA256
}
