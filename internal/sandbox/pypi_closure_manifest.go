package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// closureFile is one exact final path inside /haa-site. The source wheel
// remains digest authenticated; a generated installation file must be added
// separately and cannot borrow a wheel RECORD identity.
type closureFile struct {
	path   string
	size   int64
	sha256 string
	scheme artifactpypi.InstallationScheme
	owners []string
}

type closureManifest struct {
	files            map[string]closureFile
	generatedScripts map[string]string
	inspections      map[string]artifactpypi.WheelInspection
	order            []string
	artifactCount    int
	identity         string
}

func validClosureDestination(destination string) bool {
	if destination == "" || path.IsAbs(destination) || path.Clean(destination) != destination ||
		strings.ContainsAny(destination, "\\\x00\n\r") {
		return false
	}
	for _, component := range strings.Split(destination, "/") {
		if component == "" || component == "." || component == ".." {
			return false
		}
	}
	return true
}

func (i *PythonArtifactIntroducer) buildClosureManifest(ctx context.Context, closure []domain.AcquiredArtifact, policy artifactpypi.ResourcePolicy) (closureManifest, error) {
	if i == nil || ctx == nil || ctx.Err() != nil || len(closure) == 0 || len(closure) > policy.MaxGraphArtifacts() {
		return closureManifest{}, errors.New("python closure manifest request is invalid")
	}
	runtime := PinnedPythonRuntime()
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	manifest := closureManifest{files: make(map[string]closureFile), generatedScripts: make(map[string]string), inspections: make(map[string]artifactpypi.WheelInspection), artifactCount: len(closure)}
	seenArtifacts := make(map[string]struct{}, len(closure))
	var fileCount, expanded int64
	for _, artifact := range closure {
		if !pythonWheelVariant(artifact.Identity().Variant()) || artifact.Digest().String() == "" {
			return closureManifest{}, errors.New("python closure includes unsupported artifact")
		}
		filename, err := i.validatedWheelFilename(artifact)
		if err != nil {
			return closureManifest{}, err
		}
		source, err := i.artifactPath(artifact.ContentHandle(), artifact.Identity().Variant())
		if err != nil {
			return closureManifest{}, err
		}
		file, err := os.Open(source)
		if err != nil {
			return closureManifest{}, errors.New("open authenticated Python wheel")
		}
		info, statErr := file.Stat()
		if statErr != nil || !info.Mode().IsRegular() || uint64(info.Size()) != artifact.SizeBytes() {
			_ = file.Close()
			return closureManifest{}, errors.New("authenticated Python wheel changed")
		}
		inspection, inspectErr := artifactpypi.InspectWheelForSource(file, info.Size(), filename,
			artifact.Digest().String(), target, policy.WheelLimits(), artifact.Identity().Source())
		closeErr := file.Close()
		if inspectErr != nil || closeErr != nil || inspection.ObservedSHA256 != artifact.Digest().String() {
			return closureManifest{}, errors.New("python closure wheel authentication failed")
		}
		owner := artifact.Identity().Source().String() + ":" + inspection.Project + ":" + inspection.Version + ":" + inspection.ObservedSHA256
		if _, duplicate := seenArtifacts[owner]; duplicate {
			return closureManifest{}, errors.New("duplicate Python closure wheel")
		}
		seenArtifacts[owner] = struct{}{}
		manifest.inspections[owner] = inspection
		for _, ep := range inspection.Surface.EntryPointDetails {
			if ep.Group != "console_scripts" && ep.Group != "gui_scripts" {
				continue
			}
			if ep.Name == "" || path.Base(ep.Name) != ep.Name || strings.ContainsAny(ep.Name, "\\/\x00\r\n") || ep.Module == "" {
				return closureManifest{}, errors.New("python closure entry point wrapper name is invalid")
			}
			name := "bin/" + ep.Name
			if !validClosureDestination(name) {
				return closureManifest{}, errors.New("python closure entry point wrapper destination is invalid")
			}
			if _, exists := manifest.files[name]; exists {
				return closureManifest{}, errors.New("python closure entry point wrapper collides with RECORD")
			}
			if _, exists := manifest.generatedScripts[name]; exists {
				return closureManifest{}, errors.New("python closure entry point wrapper collides")
			}
			manifest.generatedScripts[name] = ep.Module
			if int64(len(manifest.generatedScripts)) > policy.MaxGraphFiles() {
				return closureManifest{}, errors.New("python closure has too many generated wrappers")
			}
		}
		for _, installed := range inspection.Surface.InstalledFiles {
			if !validClosureDestination(installed.Destination) || installed.Size < 0 ||
				installed.Size > policy.WheelLimits().MaxUncompressed ||
				(installed.SHA256 != "" && len(installed.SHA256) != 64) {
				return closureManifest{}, errors.New("python closure destination is invalid")
			}
			if fileCount >= policy.MaxGraphFiles() || expanded > policy.MaxGraphUncompressed()-installed.Size {
				return closureManifest{}, errors.New("python closure manifest exceeds policy")
			}
			fileCount++
			expanded += installed.Size
			if _, generated := manifest.generatedScripts[installed.Destination]; generated {
				return closureManifest{}, errors.New("python closure generated wrapper collides with RECORD")
			}
			prior, exists := manifest.files[installed.Destination]
			if exists {
				for _, previousOwner := range prior.owners {
					if previousOwner == owner {
						return closureManifest{}, fmt.Errorf("python wheel repeats final destination: %s", installed.Destination)
					}
				}
				// Shared ownership is restricted to identical site bytes. This is
				// the same final destination, regardless of archive scheme.
				if prior.scheme != artifactpypi.SchemeSite || installed.Scheme != artifactpypi.SchemeSite ||
					prior.sha256 == "" || prior.sha256 != installed.SHA256 || prior.size != installed.Size {
					return closureManifest{}, fmt.Errorf("python closure destination collides: %s", installed.Destination)
				}
				prior.owners = append(prior.owners, owner)
				manifest.files[installed.Destination] = prior
				continue
			}
			manifest.files[installed.Destination] = closureFile{
				path: installed.Destination, size: installed.Size, sha256: installed.SHA256,
				scheme: installed.Scheme, owners: []string{owner},
			}
		}
	}
	manifest.order = make([]string, 0, len(manifest.files))
	for name := range manifest.files {
		manifest.order = append(manifest.order, name)
	}
	sort.Strings(manifest.order)
	digest := sha256.New()
	for _, name := range manifest.order {
		entry := manifest.files[name]
		sort.Strings(entry.owners)
		manifest.files[name] = entry
		_, _ = fmt.Fprintf(digest, "%s\x00%d\x00%s\x00%s\x00%s\n", name, entry.size, entry.sha256, entry.scheme, strings.Join(entry.owners, ","))
	}
	generatedNames := make([]string, 0, len(manifest.generatedScripts))
	for name := range manifest.generatedScripts {
		generatedNames = append(generatedNames, name)
	}
	sort.Strings(generatedNames)
	for _, name := range generatedNames {
		_, _ = fmt.Fprintf(digest, "wrapper\x00%s\x00%s\n", name, manifest.generatedScripts[name])
	}
	manifest.identity = hex.EncodeToString(digest.Sum(nil))
	return manifest, nil
}
