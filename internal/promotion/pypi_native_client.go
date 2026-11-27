package promotion

import (
	"errors"
	"os"
	"path/filepath"
	"sort"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// Zero retains the existing pip format for its current collaborators. These
// are concrete ownership formats, not user-supplied client/plugin settings.
type pypiNativeClient uint8

const (
	nativePipClient pypiNativeClient = iota
	nativeUVClient
)

func nativeClientName(client pypiNativeClient) string {
	switch client {
	case nativePipClient:
		return "pip"
	case nativeUVClient:
		return "uv"
	}
	return ""
}
func nativeInstallerMarker(client pypiNativeClient) string {
	switch client {
	case nativePipClient:
		return "pip\n"
	case nativeUVClient:
		return "uv"
	}
	return ""
}
func nativeLookupRuntimeIdentity(client pypiNativeClient) string {
	if client == nativeUVClient {
		return sandbox.UVResolverRuntimeIdentity()
	}
	if client != nativePipClient {
		return ""
	}
	r := sandbox.PinnedPythonRuntime()
	return "python:" + r.PythonVersion + ";pip:" + r.PipVersion + ";target:" + r.InterpreterTag + "/" + r.ABITag + "/" + r.PlatformTag + ";source:" + artifactpypi.PublicPyPIProfile().Name()
}

// UV bytecode can be absent from RECORD. This first bounded native format
// refuses cache directories and freezes their absence through publication.
// It never infers authority to remove arbitrary cache bytes from their names.
func nativeUVCacheAbsences(project *pypiNativeDistribution, wheel artifactpypi.WheelInspection) ([]string, error) {
	paths := map[string]bool{}
	for _, file := range wheel.Surface.InstalledFiles {
		// UV can compile every .py file, including a wheel resource which is
		// not a required import unit. Runtime-role selection is not a cache
		// census; freeze the cache namespace of each authentic Python source.
		if (file.Scheme != artifactpypi.SchemeSite && file.Scheme != artifactpypi.SchemeDistInfo) || filepath.Ext(file.Destination) != ".py" {
			continue
		}
		claim, ok := project.files["site/"+file.Destination]
		if !ok {
			return nil, errors.New("native UV cache role has no authentic original source")
		}
		paths[filepath.Join(filepath.Dir(claim.destination.Final), "__pycache__")] = true
	}
	result := make([]string, 0, len(paths))
	for path := range paths {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("native UV compiled or unknown Python cache is unsupported")
		}
		result = append(result, path)
	}
	sort.Strings(result)
	return result, nil
}
