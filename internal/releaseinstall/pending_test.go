package releaseinstall

import (
	"strings"
	"testing"
)

func TestSystemCopyStageNameRejectsNonDigestIdentity(t *testing.T) {
	digest := strings.Repeat("a", 64)
	if name, err := SystemCopyStageName(digest); err != nil || name != ".pending-"+digest {
		t.Fatal(name, err)
	}
	for _, value := range []string{"", digest[:63], strings.ToUpper(digest), digest + "/other", "../" + digest} {
		if _, err := SystemCopyStageName(value); err == nil {
			t.Fatal("invalid identity accepted")
		}
	}
}
