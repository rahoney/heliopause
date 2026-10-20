// Command release-packages builds OS-native packages from the release workflow's
// already-built native payload. It never installs or activates that payload.
package main

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goreleaser/nfpm/v2"
	_ "github.com/goreleaser/nfpm/v2/deb"
	"github.com/goreleaser/nfpm/v2/files"
	_ "github.com/goreleaser/nfpm/v2/rpm"
	"golang.org/x/mod/semver"
)

const maxInputBytes = 1 << 30

var packageFormats = []string{"deb", "rpm"}

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "release-packages:", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("release-packages", flag.ContinueOnError)
	version := flags.String("version", "", "exact release SemVer")
	epoch := flags.Int64("source-date-epoch", 0, "source commit Unix timestamp")
	assets := flags.String("assets-dir", "", "absolute native release asset directory")
	output := flags.String("output-dir", "", "absolute existing package output directory")
	lock := flags.String("runtime-lock", "scripts/runtimes.lock.json", "canonical runtime lock")
	license := flags.String("license", "LICENSE", "repository license")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("positional arguments are not supported")
	}
	return buildPackages(*version, *epoch, *assets, *output, *lock, *license)
}

func buildPackages(version string, epoch int64, assets, output, lock, license string) (resultErr error) {
	core, _, _ := strings.Cut(strings.TrimPrefix(version, "v"), "-")
	core, _, _ = strings.Cut(core, "+")
	if len(version) > 128 || !semver.IsValid(version) || len(strings.Split(core, ".")) != 3 {
		return errors.New("release version is invalid")
	}
	if epoch <= 0 || epoch > math.MaxUint32 {
		return errors.New("source timestamp is outside the package format range")
	}
	for _, directory := range []string{assets, output} {
		info, err := os.Lstat(directory)
		if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("package directories must be absolute, existing non-symlink directories")
		}
	}
	for _, format := range packageFormats {
		if _, err := os.Lstat(filepath.Join(output, "helox-linux-amd64."+format)); !errors.Is(err, os.ErrNotExist) {
			return errors.New("package output already exists or cannot be inspected")
		}
	}
	versionRoot := "/usr/lib/heliopause/releases/" + version
	mtime := time.Unix(epoch, 0).UTC()
	content := files.Contents{}
	for _, input := range []struct {
		source, destination string
		mode                os.FileMode
	}{
		{filepath.Join(assets, "helox-linux-amd64"), versionRoot + "/helox", 0o555},
		{filepath.Join(assets, "haa_gvisor_observer-linux-amd64"), versionRoot + "/haa_gvisor_observer", 0o555},
		{filepath.Join(assets, "haa-network-policy-helper-linux-amd64"), versionRoot + "/haa-network-policy-helper", 0o500},
		{filepath.Join(assets, "haa-release-installer-linux-amd64"), versionRoot + "/haa-release-installer", 0o500},
		{filepath.Join(assets, "helox-runtime-images.json"), versionRoot + "/helox-runtime-images.json", 0o444},
		{lock, versionRoot + "/runtimes.lock.json", 0o444},
		{license, "/usr/share/doc/helox/copyright", 0o444},
	} {
		if err := regularInput(input.source); err != nil {
			return err
		}
		content = append(content, packageFile(input.source, input.destination, input.mode, mtime))
	}
	stage, err := os.MkdirTemp("", "helox-package-metadata-")
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, os.RemoveAll(stage)) }()
	for _, item := range []struct{ name, destination, body string }{
		{"service", "/usr/lib/systemd/system/helox-network-policy.service", policyService(versionRoot)},
		{"readme", "/usr/share/doc/helox/README", packageReadme},
	} {
		path := filepath.Join(stage, item.name)
		if err := os.WriteFile(path, []byte(item.body), 0o600); err != nil {
			return err
		}
		content = append(content, packageFile(path, item.destination, 0o444, mtime))
	}
	content = append(content, &files.Content{Type: "symlink", Source: versionRoot + "/helox", Destination: "/usr/bin/helox", FileInfo: &files.ContentFileInfo{Owner: "root", Group: "root", MTime: mtime}})
	postinstall := filepath.Join(stage, "postinstall")
	if err := os.WriteFile(postinstall, []byte("#!/bin/sh\nset -eu\nprintf '%s\\n' 'Heliopause payload installed. Configure the protected runtime and run helox doctor before use.'\n"), 0o700); err != nil {
		return err
	}
	for _, format := range packageFormats {
		info := nfpm.WithDefaults(&nfpm.Info{
			Name: "helox", Arch: "amd64", Platform: "linux", Version: version, Release: "1", MTime: mtime,
			Maintainer: "Heliopause maintainers", Description: "Heliopause native payload; protected runtime setup and doctor are required.",
			Homepage: "https://github.com/rahoney/heliopause", License: "Apache-2.0", DisableGlobbing: true,
			Overridables: nfpm.Overridables{
				Contents: content, Scripts: nfpm.Scripts{PostInstall: postinstall},
				RPM: nfpm.RPM{BuildHost: "reproducible.helox", Compression: "gzip"},
				Deb: nfpm.Deb{Compression: "gzip"},
			},
		})
		if err := nfpm.Validate(info); err != nil {
			return err
		}
		packager, err := nfpm.Get(format)
		if err != nil {
			return err
		}
		path := filepath.Join(output, "helox-linux-amd64."+format)
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		buildErr := packager.Package(info, file)
		if err := errors.Join(buildErr, file.Close()); err != nil {
			return err
		}
	}
	return nil
}

func packageFile(source, destination string, mode os.FileMode, mtime time.Time) *files.Content {
	return &files.Content{Source: source, Destination: destination, FileInfo: &files.ContentFileInfo{Owner: "root", Group: "root", Mode: mode, MTime: mtime}}
}

func regularInput(path string) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxInputBytes {
		return errors.New("package input must be a bounded nonempty regular file")
	}
	return nil
}

func policyService(versionRoot string) string {
	return "[Unit]\nDescription=Heliopause typed network-policy helper\nAfter=docker.service\nConditionPathExists=/etc/heliopause/network-policy.json\n\n" +
		"[Service]\nType=simple\nExecStart=" + versionRoot + "/haa-network-policy-helper serve\nUser=root\nGroup=root\nUmask=0077\n" +
		"RuntimeDirectory=heliopause-network-policy\nRuntimeDirectoryMode=0755\nRuntimeDirectoryPreserve=yes\nNoNewPrivileges=yes\nRestart=no\n\n[Install]\nWantedBy=multi-user.target\n"
}

const packageReadme = `This package installs versioned native payload and a CLI symlink.
It does not activate a runtime, overwrite installed runsc/observer identities,
download images, create caller credentials, or start/enable a privileged service.

The bundled release installer is inert until explicitly invoked and does not
enroll this OS-package installation into native self-update management.
Package files remain owned by the OS package manager. Native quarantine
self-update requires a separately verified native installation; it does not
overwrite this package layout or bypass the package database.

The optional helox-network-policy.service remains disabled. Before enabling it,
configure the existing protected runtime and a root-owned, non-writable
/etc/heliopause/network-policy.json using the actual caller client_path,
client_uid and client_gid and socket_path /run/heliopause-network-policy/network-policy.sock.
No working authorization configuration is generated from guessed IDs.

Run helox doctor after controlled runtime setup. Package installation success
alone does not establish runtime readiness, verification or observation coverage.
`
