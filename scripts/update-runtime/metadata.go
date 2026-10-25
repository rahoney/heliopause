package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type document struct {
	URL  string `json:"url"`
	Body []byte `json:"body"`
}
type archive struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	URL    string `json:"url"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}
type metadata struct {
	Bun          *bunSupplyMetadata  `json:"bun_supply,omitempty"`
	PNPM         *pnpmSupplyMetadata `json:"pnpm_supply,omitempty"`
	UV           *uvSupplyMetadata   `json:"uv_supply,omitempty"`
	Image        string              `json:"image"`
	Architecture string              `json:"architecture"`
	Bundled      string              `json:"bundled,omitempty"`
	Archives     []archive           `json:"archives,omitempty"`
	Documents    []document          `json:"documents,omitempty"`
}

func imageName(component, version string) (string, string, error) {
	switch component {
	case "go":
		return "golang", version + "-bookworm", nil
	case "node":
		return "node", version + "-slim", nil
	case "python":
		return "python", version + "-slim-bookworm", nil
	case "rust":
		return "rust", version + "-bookworm", nil
	}
	return "", "", errors.New("unknown runtime component")
}
func releaseURL(component, version string) string {
	switch component {
	case "go":
		return "https://go.dev/dl/?mode=json"
	case "node":
		return "https://nodejs.org/dist/index.json"
	case "python":
		return "https://raw.githubusercontent.com/python/cpython/v" + version + "/Lib/ensurepip/__init__.py"
	}
	return ""
}
func officialHTTP() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &http.Client{Transport: transport, Timeout: 90 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 4 || req.URL.Scheme != "https" || req.URL.Host != "dl.google.com" || via[0].URL.Host != "go.dev" || !strings.HasPrefix(via[0].URL.Path, "/dl/") {
			return errors.New("unapproved official redirect")
		}
		return nil
	}}
}
func getOfficial(client *http.Client, address, token string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json, application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("official endpoint %s transport failed: %w", req.URL.Host, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("official endpoint %s status %d", req.URL.Host, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(body) > 8<<20 {
		return nil, errors.New("official metadata exceeds bound")
	}
	return body, nil
}
func resolveMetadata(component, version string) (metadata, error) {
	name, tag, err := imageName(component, version)
	if err != nil {
		return metadata{}, err
	}
	client := officialHTTP()
	tokenURL := "https://auth.docker.io/token?service=registry.docker.io&scope=" + url.QueryEscape("repository:library/"+name+":pull")
	body, err := getOfficial(client, tokenURL, "")
	if err != nil {
		return metadata{}, err
	}
	var auth struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &auth); err != nil || auth.Token == "" {
		return metadata{}, errors.New("anonymous registry token unavailable")
	}
	indexURL := "https://registry-1.docker.io/v2/library/" + name + "/manifests/" + tag
	index, err := getOfficial(client, indexURL, auth.Token)
	if err != nil {
		return metadata{}, err
	}
	digest, err := amd64Manifest(index)
	if err != nil {
		return metadata{}, err
	}
	manifestURL := "https://registry-1.docker.io/v2/library/" + name + "/manifests/" + digest
	manifest, err := getOfficial(client, manifestURL, auth.Token)
	if err != nil {
		return metadata{}, err
	}
	docs := []document{{indexURL, index}, {manifestURL, manifest}}
	if address := releaseURL(component, version); address != "" {
		body, err := getOfficial(client, address, "")
		if err != nil {
			return metadata{}, err
		}
		docs = append(docs, document{address, body})
	}
	return metadataFromDocuments(component, version, docs)
}
func amd64Manifest(body []byte) (string, error) {
	var index struct {
		Schema    int `json:"schemaVersion"`
		Manifests []struct {
			Digest   string                                     `json:"digest"`
			Platform struct{ OS, Architecture, Variant string } `json:"platform"`
		} `json:"manifests"`
	}
	if err := json.Unmarshal(body, &index); err != nil {
		return "", err
	}
	if index.Schema != 2 {
		return "", errors.New("OCI index schema mismatch")
	}
	digest := ""
	for _, m := range index.Manifests {
		if m.Platform.OS == "linux" && m.Platform.Architecture == "amd64" && m.Platform.Variant == "" {
			if digest != "" {
				return "", errors.New("ambiguous amd64 image")
			}
			digest = m.Digest
		}
	}
	if !strings.HasPrefix(digest, "sha256:") || !digestPattern.MatchString(strings.TrimPrefix(digest, "sha256:")) {
		return "", errors.New("missing/invalid linux amd64 image digest")
	}
	return digest, nil
}
func metadataFromDocuments(component, version string, docs []document) (metadata, error) {
	name, tag, err := imageName(component, version)
	if err != nil {
		return metadata{}, err
	}
	expected := 2
	if releaseURL(component, version) != "" {
		expected = 3
	}
	if len(docs) != expected {
		return metadata{}, errors.New("official metadata document inventory mismatch")
	}
	base := "https://registry-1.docker.io/v2/library/" + name + "/manifests/"
	if docs[0].URL != base+tag {
		return metadata{}, errors.New("official image tag metadata mismatch")
	}
	digest, err := amd64Manifest(docs[0].Body)
	if err != nil {
		return metadata{}, err
	}
	if docs[1].URL != base+digest || "sha256:"+hash(docs[1].Body) != digest {
		return metadata{}, errors.New("official manifest digest mismatch")
	}
	var manifest struct {
		Schema int `json:"schemaVersion"`
		Config struct {
			Digest string `json:"digest"`
		} `json:"config"`
		Layers []json.RawMessage `json:"layers"`
	}
	if err := json.Unmarshal(docs[1].Body, &manifest); err != nil || manifest.Schema != 2 || !digestPattern.MatchString(strings.TrimPrefix(manifest.Config.Digest, "sha256:")) || len(manifest.Layers) == 0 {
		return metadata{}, errors.New("invalid OCI image manifest")
	}
	m := metadata{Image: name + ":" + tag + "@" + digest, Architecture: "amd64", Documents: docs}
	if expected == 3 && docs[2].URL != releaseURL(component, version) {
		return metadata{}, errors.New("official release metadata URL mismatch")
	}
	switch component {
	case "node":
		var releases []struct{ Version, NPM string }
		if err := json.Unmarshal(docs[2].Body, &releases); err != nil {
			return m, err
		}
		for _, r := range releases {
			if r.Version == "v"+version {
				if m.Bundled != "" {
					return m, errors.New("duplicate Node release")
				}
				m.Bundled = r.NPM
			}
		}
		if !versionPattern.MatchString(m.Bundled) {
			return m, errors.New("exact Node release/bundled npm unavailable")
		}
	case "python":
		matches := regexp.MustCompile(`(?m)^_PIP_VERSION = "([0-9]+\.[0-9]+\.[0-9]+)"$`).FindAllSubmatch(docs[2].Body, -1)
		if len(matches) != 1 {
			return m, errors.New("exact CPython bundled pip unavailable")
		}
		m.Bundled = string(matches[0][1])
	case "go":
		var releases []struct {
			Version string
			Stable  bool
			Files   []struct {
				Filename, OS, Arch, SHA256, Kind string
				Size                             int64
			}
		}
		if err := json.Unmarshal(docs[2].Body, &releases); err != nil {
			return m, err
		}
		for _, r := range releases {
			if r.Version != "go"+version || !r.Stable {
				continue
			}
			for _, f := range r.Files {
				if f.Kind != "archive" || !((f.OS == "linux" && f.Arch == "amd64") || (f.OS == "darwin" && (f.Arch == "amd64" || f.Arch == "arm64"))) {
					continue
				}
				expected := "go" + version + "." + f.OS + "-" + f.Arch + ".tar.gz"
				if f.Filename != expected || f.Size <= 0 || f.Size > 512<<20 || !digestPattern.MatchString(f.SHA256) {
					return m, errors.New("official SDK archive metadata invalid")
				}
				m.Archives = append(m.Archives, archive{f.OS, f.Arch, "https://go.dev/dl/" + f.Filename, f.Size, f.SHA256})
			}
		}
		seen := map[string]bool{}
		for _, a := range m.Archives {
			key := a.OS + "/" + a.Arch
			if seen[key] {
				return m, errors.New("duplicate SDK platform")
			}
			seen[key] = true
		}
		if len(seen) != 3 {
			return m, errors.New("requested supported Go SDK release/platforms unavailable")
		}
	}
	return m, nil
}
func validateMetadata(component, version string, m metadata) error {
	expected, err := metadataFromDocuments(component, version, m.Documents)
	if err != nil {
		return err
	}
	a, _ := json.Marshal(expected)
	b, _ := json.Marshal(m)
	if string(a) != string(b) {
		return errors.New("derived metadata differs from frozen official documents")
	}
	return nil
}
func hostArchive(m metadata) (archive, error) {
	for _, a := range m.Archives {
		if a.OS == runtime.GOOS && a.Arch == runtime.GOARCH {
			return a, nil
		}
	}
	return archive{}, errors.New("target SDK archive does not support this host")
}
