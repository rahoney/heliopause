//go:build linux

package releaseactivation

import (
	"context"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/releaseinstall"
)

func TestBootstrapFilePreparationRejectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := BootstrapFiles(ctx); err == nil {
		t.Fatal("canceled initial preparation accepted")
	}
}

func TestBootstrapNativeRegistrationUsesOnlyExactProtectedConfig(t *testing.T) {
	release, _ := operationReleaseFixture(t, "v1.0.0")
	configs := runtimeConfigFixture(t).Configs
	record, err := nativeRegistration(release.manifest, configs)
	if err != nil || record.Version != "v1.0.0" || record.ManifestSHA256 != release.manifest.SHA256() || record.ClientUID != 1000 || record.ClientGID != 1000 {
		t.Fatal(record, err)
	}
	configs[0].Body = []byte(strings.Replace(string(configs[0].Body), `"client_uid":1000`, `"client_uid":0`, 1))
	configs[0].SHA256 = bodySHA256(configs[0].Body)
	if _, err := nativeRegistration(release.manifest, configs); err == nil {
		t.Fatal("root client selected")
	}
}

func TestBootstrapStagingExecutablesRemainProtectedConsumers(t *testing.T) {
	for _, path := range []string{bootstrapDirectory + "/haa-release-installer-linux-amd64", releaseinstall.SystemStagingRoot + "/.prepared-x/new-cli", nativeVersionPath("v1.0.0", "helox-linux-amd64"), "/usr/libexec/heliopause/helox"} {
		if !protectedConsumerPath(path) {
			t.Fatal("protected consumer omitted", path)
		}
	}
	if protectedConsumerPath(releaseinstall.SystemStagingRoot + "-foreign/helox") {
		t.Fatal("foreign prefix misclassified")
	}
}
