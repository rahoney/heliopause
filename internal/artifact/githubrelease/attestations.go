package githubrelease

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/klauspost/compress/snappy"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	maximumAttestations        = 16
	maximumAttestationResponse = 2 << 20
	maximumAttestationBundle   = 1 << 20
	attestationTimeout         = 90 * time.Second
	attestationBlobHost        = "tmaproduction.blob.core.windows.net"
)

func (c *Client) readAttestations(ctx context.Context, resolved domain.ResolvedArtifact) ([][]byte, error) {
	if err := validRequest(ctx, c); err != nil {
		return nil, err
	}
	selector, _, err := SelectionForResolved(resolved)
	if err != nil {
		return nil, err
	}
	digest := strings.TrimPrefix(resolved.DeclaredIntegrity(), "sha256:")
	operation, cancel := context.WithTimeout(ctx, attestationTimeout)
	defer cancel()
	endpoint := *c.api
	endpoint.Path = "/repos/" + selector.Owner() + "/" + selector.Repo() + "/attestations/sha256:" + digest
	endpoint.RawQuery = url.Values{"per_page": {strconv.Itoa(maximumAttestations)}}.Encode()
	response, err := c.request(operation, endpoint.String(), "application/vnd.github+json", false)
	if err != nil {
		return nil, errors.Join(errors.New("release attestation request failed"), operation.Err())
	}
	if response.Header.Get("Link") != "" || !strings.HasPrefix(strings.ToLower(response.Header.Get("Content-Type")), "application/json") {
		_ = response.Body.Close()
		return nil, errors.New("release attestation response is unsupported or paginated")
	}
	body, err := readAttestationResponse(operation, response, maximumAttestationResponse)
	if err != nil {
		return nil, err
	}
	var result struct {
		Attestations []struct {
			RepositoryID uint64          `json:"repository_id"`
			BundleURL    string          `json:"bundle_url"`
			Bundle       json.RawMessage `json:"bundle"`
		} `json:"attestations"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if decoder.Decode(&result) != nil || ensureEOF(decoder) != nil || len(result.Attestations) == 0 || len(result.Attestations) > maximumAttestations {
		return nil, errors.New("release attestation list is invalid")
	}
	bundles := make([][]byte, 0, len(result.Attestations))
	for _, item := range result.Attestations {
		if err := operation.Err(); err != nil {
			return nil, err
		}
		if item.RepositoryID == 0 {
			return nil, errors.New("release attestation repository record is invalid")
		}
		var bundle []byte
		inline := bytes.TrimSpace(item.Bundle)
		if len(inline) > 0 && !bytes.Equal(inline, []byte("null")) {
			bundle = append([]byte(nil), inline...)
		} else {
			if len(item.BundleURL) > 4096 {
				return nil, errors.New("release attestation location is excessive")
			}
			location, err := url.Parse(item.BundleURL)
			if err != nil || !validAttestationLocation(location, item.RepositoryID) {
				return nil, errors.New("release attestation location is invalid")
			}
			blob, err := c.request(operation, location.String(), "application/octet-stream", false)
			if err != nil {
				return nil, errors.Join(errors.New("release attestation bundle request failed"), operation.Err())
			}
			encoded, err := readAttestationResponse(operation, blob, maximumAttestationBundle)
			if err != nil {
				return nil, err
			}
			size, err := snappy.DecodedLen(encoded)
			if err != nil || size <= 0 || size > maximumAttestationBundle {
				return nil, errors.New("release attestation decoded size is invalid")
			}
			bundle, err = snappy.Decode(nil, encoded)
			if err != nil || len(bundle) != size {
				return nil, errors.New("release attestation compression is invalid")
			}
		}
		trimmed := bytes.TrimSpace(bundle)
		if len(trimmed) == 0 || len(bundle) > maximumAttestationBundle || trimmed[0] != '{' || !json.Valid(bundle) {
			return nil, errors.New("release attestation bundle is invalid")
		}
		bundles = append(bundles, bundle)
	}
	if err := operation.Err(); err != nil {
		return nil, err
	}
	return bundles, nil
}

func validAttestationLocation(location *url.URL, repositoryID uint64) bool {
	if location == nil || repositoryID == 0 || location.Scheme != "https" || location.Host != attestationBlobHost || location.User != nil || location.Fragment != "" || location.Opaque != "" || location.RawPath != "" {
		return false
	}
	return path.Clean(location.Path) == location.Path && strings.HasPrefix(location.Path, "/attestations/"+strconv.FormatUint(repositoryID, 10)+"/") && strings.HasSuffix(location.Path, ".json.sn")
}

func readAttestationResponse(ctx context.Context, response *http.Response, limit int64) ([]byte, error) {
	if response == nil || response.Body == nil {
		return nil, errors.New("release attestation response is unavailable")
	}
	if response.StatusCode != http.StatusOK || response.ContentLength > limit {
		_ = response.Body.Close()
		return nil, errors.New("release attestation response is unexpected")
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, limit+1))
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil || len(body) == 0 || int64(len(body)) > limit || (response.ContentLength >= 0 && int64(len(body)) != response.ContentLength) {
		return nil, errors.Join(errors.New("release attestation response is incomplete or excessive"), ctx.Err())
	}
	return body, nil
}
