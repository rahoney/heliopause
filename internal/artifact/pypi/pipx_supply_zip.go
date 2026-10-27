package pypi

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"sort"
	"time"
)

const pipxSupplyZIPLimit = 8 << 20

// ZIP constructs a bounded candidate from the original authenticated members.
// It neither grants a runtime role nor rewrites RECORD for an installed layout.
func (c PipxSupplyClosure) ZIP(inputs []UVWheelInput) ([]byte, error) {
	if !c.Valid() || len(inputs) != len(c.wheels) {
		return nil, errors.New("pipx ZIP requires a complete typed supply closure")
	}
	byURL := make(map[string]UVWheelInput, len(inputs))
	for _, input := range inputs {
		if input.Reader == nil || input.Size <= 0 || input.Size > pipxSupplyZIPLimit {
			return nil, errors.New("pipx ZIP input size or reader is invalid")
		}
		if _, exists := byURL[input.URL]; exists {
			return nil, errors.New("pipx ZIP input is duplicated")
		}
		byURL[input.URL] = input
	}
	wanted := make(map[string]map[string]PipxSupplyFile)
	var total int64
	allNames := map[string]bool{}
	for _, file := range c.files {
		if file.Size < 0 || file.Size > pipxSupplyZIPLimit-total || allNames[file.Path] || (file.Scheme != SchemeData && file.Path != file.Destination) {
			return nil, errors.New("pipx ZIP manifest exceeds bounds, collides or requires relocation")
		}
		total += file.Size
		allNames[file.Path] = true
		if wanted[file.Project] == nil {
			wanted[file.Project] = make(map[string]PipxSupplyFile)
		}
		wanted[file.Project][file.Path] = file
	}
	members := make(map[string][]byte, len(c.files))
	for _, wheel := range c.wheels {
		input, ok := byURL[wheel.URL]
		if !ok || input.Size != wheel.Size {
			return nil, errors.New("pipx ZIP wheel input differs from closure")
		}
		body, err := io.ReadAll(io.NewSectionReader(input.Reader, 0, input.Size))
		if err != nil || int64(len(body)) != input.Size || pipxSupplyDigest(body) != wheel.SHA256 {
			return nil, errors.New("pipx ZIP wheel bytes changed")
		}
		archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for _, member := range archive.File {
			if member.FileInfo().IsDir() {
				continue
			} // Original directory entries carry no bytes.
			fact, ok := wanted[wheel.Project][member.Name]
			if !ok || seen[member.Name] || member.UncompressedSize64 != uint64(fact.Size) {
				return nil, errors.New("pipx ZIP member set differs from closure")
			}
			seen[member.Name] = true
			reader, err := member.Open()
			if err != nil {
				return nil, err
			}
			data, readErr := io.ReadAll(io.LimitReader(reader, fact.Size+1))
			closeErr := reader.Close()
			if readErr != nil || closeErr != nil || int64(len(data)) != fact.Size || pipxSupplyDigest(data) != fact.SHA256 {
				return nil, errors.New("pipx ZIP member bytes changed")
			}
			members[member.Name] = data
		}
		if len(seen) != len(wanted[wheel.Project]) {
			return nil, errors.New("pipx ZIP member is missing")
		}
	}
	if len(members) != len(c.files) {
		return nil, errors.New("pipx ZIP closure is incomplete")
	}
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	sort.Strings(names)
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Store}
		header.SetMode(0444)
		header.Modified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
		member, err := writer.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err := member.Write(members[name]); err != nil {
			return nil, err
		}
		if output.Len() > pipxSupplyZIPLimit {
			return nil, errors.New("pipx ZIP exceeds candidate bound")
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	if output.Len() > pipxSupplyZIPLimit {
		return nil, errors.New("pipx ZIP exceeds candidate bound")
	}
	return output.Bytes(), nil
}

func pipxSupplyDigest(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
