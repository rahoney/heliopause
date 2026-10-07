package terraformprovider

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

var protocolVersion = regexp.MustCompile(`^[1-9][0-9]{0,2}\.[0-9]{1,3}$`)

func validProtocols(values []string) bool {
	if len(values) == 0 || len(values) > 8 {
		return false
	}
	seen := map[string]bool{}
	for _, value := range values {
		major := strings.SplitN(value, ".", 2)[0]
		if !protocolVersion.MatchString(value) || seen[major] {
			return false
		}
		seen[major] = true
	}
	return true
}

// ValidateDiscovery deliberately supports the public origin's providers.v1
// service only. Remote services, mirrors and alternate registries are not
// selected from untrusted metadata or ambient Terraform configuration.
func ValidateDiscovery(body []byte) error {
	var services map[string]string
	if err := decodeRegistryJSON(body, &services, 64<<10); err != nil {
		return err
	}
	if services["providers.v1"] != "/v1/providers/" && services["providers.v1"] != registryEndpoint+"/v1/providers/" {
		return errors.New("terraform public providers.v1 discovery is unsupported")
	}
	return nil
}

func decodeRegistryJSON(body []byte, target any, limit int) error {
	if len(body) == 0 || len(body) > limit || !utf8.Valid(body) {
		return errors.New("terraform registry JSON exceeds bounds")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := registryJSONValue(decoder, 0); err != nil {
		return errors.New("terraform registry JSON is invalid or ambiguous")
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("terraform registry JSON contains trailing data")
	}
	if err := json.Unmarshal(body, target); err != nil {
		return errors.New("terraform registry JSON has invalid schema")
	}
	return nil
}

func registryJSONValue(decoder *json.Decoder, depth int) error {
	if depth > 32 {
		return errors.New("registry JSON depth exceeds bound")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, aggregate := token.(json.Delim)
	if !aggregate {
		return nil
	}
	if delimiter != '{' && delimiter != '[' {
		return errors.New("invalid JSON delimiter")
	}
	keys := map[string]bool{}
	for decoder.More() {
		if delimiter == '{' {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			folded := strings.ToLower(name)
			if !ok || keys[folded] || name != folded {
				return errors.New("ambiguous JSON key")
			}
			keys[folded] = true
		}
		if err := registryJSONValue(decoder, depth+1); err != nil {
			return err
		}
	}
	closing, err := decoder.Token()
	if err != nil || delimiter == '{' && closing != json.Delim('}') || delimiter == '[' && closing != json.Delim(']') {
		return errors.New("invalid closing JSON delimiter")
	}
	return nil
}
