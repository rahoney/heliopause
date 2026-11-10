package pypi

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUVExactReleaseMetadataBindsIdentityAndFiles(t *testing.T) {
	response := map[string]any{"info": map[string]string{"name": "packaging", "version": "25.0"}, "urls": []any{map[string]any{"filename": "packaging-25.0-py3-none-any.whl", "url": "https://files.pythonhosted.org/packages/packaging-25.0-py3-none-any.whl", "size": 100, "digests": map[string]string{"sha256": sampleSHA256}, "requires_python": ">=3.8", "yanked": false, "packagetype": "bdist_wheel"}}}
	body, _ := json.Marshal(response)
	page, err := ParseUVReleaseMetadata("packaging", "25.0", body)
	if err != nil || page.Project() != "packaging" || len(page.Files()) != 1 {
		t.Fatal("exact registry metadata refused", err)
	}
	for name, body := range map[string]string{
		"missing":            `{}`,
		"different project":  strings.Replace(string(body), `"name":"packaging"`, `"name":"other"`, 1),
		"different version":  strings.Replace(string(body), `"version":"25.0"`, `"version":"24.0"`, 1),
		"foreign file":       strings.ReplaceAll(string(body), "files.pythonhosted.org", "example.org"),
		"other file version": strings.ReplaceAll(string(body), "packaging-25.0-", "packaging-24.0-"),
		"bad digest":         strings.Replace(string(body), sampleSHA256, "invalid", 1),
		"unknown type":       strings.Replace(string(body), "bdist_wheel", "unknown", 1),
		"trailing":           string(body) + "{}",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseUVReleaseMetadata("packaging", "25.0", []byte(body)); err == nil {
				t.Fatal("accepted changed registry authority")
			}
		})
	}
	if got, err := UVReleaseMetadataURL("Charset_Normalizer", "3.5.2"); err != nil || got != "https://pypi.org/pypi/charset-normalizer/3.5.2/json" {
		t.Fatal("endpoint normalization", got, err)
	}
	for _, input := range []string{"../other", "x?query=y", "https://example.org"} {
		if _, err := UVReleaseMetadataURL(input, "1.0"); err == nil {
			t.Fatal("accepted endpoint injection", input)
		}
	}
}
