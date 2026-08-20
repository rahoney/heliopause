package githubrelease

import (
	"strings"
	"testing"
)

func TestParseReleaseForSelectorBindsExactAsset(t *testing.T) {
	t.Parallel()
	selector, _ := ParseSelector("owner/repo@v1.2.3#tool-linux-amd64")
	body := []byte(`{"id":41,"tag_name":"v1.2.3","draft":false,"immutable":true,"published_at":"2026-08-20T01:02:03Z","assets":[{"id":7,"name":"tool-linux-amd64","state":"uploaded","size":42,"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`)
	asset, err := ParseReleaseForSelector(selector, body)
	if err != nil || asset.ReleaseID() != 41 || asset.AssetID() != 7 || asset.Size() != 42 || !asset.Immutable() || asset.Digest().String() != strings.Repeat("a", 64) {
		t.Fatalf("ParseReleaseForSelector() = %#v, %v", asset, err)
	}
}

func TestParseReleaseForSelectorFailsClosed(t *testing.T) {
	t.Parallel()
	selector, _ := ParseSelector("owner/repo@v1.2.3#tool")
	for _, body := range []string{
		`{"id":1,"tag_name":"v1.2.4","draft":false,"published_at":"2026-08-20T01:02:03Z","assets":[]}`,
		`{"id":1,"tag_name":"v1.2.3","draft":true,"published_at":"2026-08-20T01:02:03Z","assets":[]}`,
		`{"id":1,"tag_name":"v1.2.3","draft":false,"published_at":"2026-08-20T01:02:03Z","assets":[{"id":2,"name":"tool","state":"uploaded","size":1,"digest":"sha512:x"}]}`,
		`{"id":1,"tag_name":"v1.2.3","draft":false,"published_at":"2026-08-20T01:02:03Z","assets":[{"id":2,"name":"tool","state":"starter","size":1,"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`,
		`{"id":1,"tag_name":"v1.2.3","draft":false,"published_at":"bad","assets":[]}`,
	} {
		if _, err := ParseReleaseForSelector(selector, []byte(body)); err == nil {
			t.Fatalf("ParseReleaseForSelector(%s) error = nil", body)
		}
	}
}
