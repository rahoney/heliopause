package pypi

import (
	"errors"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// SourceProfile is the canonical source policy for one Python distribution
// index. It is intentionally a named profile; callers cannot supply URLs.
type SourceProfile struct {
	name              string
	source            domain.SourceID
	indexURL          string
	indexHost         string
	distributionHosts []string
	ownedProjects     map[string]bool
	resourcePolicy    ResourcePolicy
}

func (p SourceProfile) Name() string                   { return p.name }
func (p SourceProfile) Source() domain.SourceID        { return p.source }
func (p SourceProfile) IndexURL() string               { return p.indexURL }
func (p SourceProfile) IndexHost() string              { return p.indexHost }
func (p SourceProfile) ResourcePolicy() ResourcePolicy { return p.resourcePolicy }
func (p SourceProfile) DistributionHosts() []string {
	return append([]string(nil), p.distributionHosts...)
}

var (
	publicPyPIProfile = mustSourceProfile(SourceProfile{
		name:              "pypi",
		source:            mustSourceID("pypi"),
		indexURL:          "https://pypi.org/simple/",
		indexHost:         "pypi.org",
		distributionHosts: []string{"files.pythonhosted.org"},
		resourcePolicy:    defaultResourcePolicy(),
	})
	pyTorchProfiles = map[string]SourceProfile{}
)

func init() {
	for name, locked := range runtimeidentity.PythonSourceProfiles {
		owned := make(map[string]bool, len(locked.OwnedProjects))
		for _, project := range locked.OwnedProjects {
			owned[project] = true
		}
		profile := SourceProfile{
			name:              locked.Name,
			source:            mustSourceID(locked.SourceID),
			indexURL:          locked.IndexURL,
			indexHost:         locked.IndexHost,
			distributionHosts: append([]string(nil), locked.DistributionHosts...),
			ownedProjects:     owned,
			resourcePolicy:    defaultResourcePolicy(),
		}
		switch profile.name {
		case "pytorch:cpu":
			profile.resourcePolicy = pyTorchCPUResourcePolicy()
		case "pytorch:cu126":
			profile.resourcePolicy = pyTorchCU126ResourcePolicy()
		}
		pyTorchProfiles[strings.TrimPrefix(name, "pytorch:")] = mustSourceProfile(profile)
	}
}

func mustSourceID(value string) domain.SourceID {
	id, err := domain.NewSourceID(value)
	if err != nil {
		panic(err)
	}
	return id
}

func mustSourceProfile(profile SourceProfile) SourceProfile {
	if profile.name == "" || profile.source.String() == "" || profile.indexURL == "" || profile.indexHost == "" || len(profile.distributionHosts) == 0 || !profile.resourcePolicy.valid() {
		panic("invalid source profile")
	}
	return profile
}

// PublicPyPIProfile returns the immutable canonical PyPI profile.
func PublicPyPIProfile() SourceProfile { return publicPyPIProfile }

// PyTorchProfile resolves only a named official profile.
func PyTorchProfile(name string) (SourceProfile, bool) {
	profile, ok := pyTorchProfiles[name]
	return profile, ok
}

// AllSourceProfiles returns the immutable Python source profile set in stable
// order for endpoint validation and composition wiring.
func AllSourceProfiles() []SourceProfile {
	profiles := []SourceProfile{publicPyPIProfile}
	for _, profile := range pyTorchProfiles {
		profiles = append(profiles, profile)
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].name < profiles[j].name })
	return profiles
}

// ProfileForSource returns the canonical profile for a normalized source ID.
func ProfileForSource(source domain.SourceID) (SourceProfile, bool) {
	if source == publicPyPIProfile.source {
		return publicPyPIProfile, true
	}
	for _, profile := range pyTorchProfiles {
		if profile.source == source {
			return profile, true
		}
	}
	return SourceProfile{}, false
}

func IsPyTorchSource(source domain.SourceID) bool {
	profile, ok := ProfileForSource(source)
	return ok && strings.HasPrefix(profile.name, "pytorch:")
}

// IsPyTorchOwnedProject is deliberately an allowlist, not a name heuristic.
func IsPyTorchOwnedProject(project string) bool {
	normalized, err := NormalizeProjectName(project)
	if err != nil {
		return false
	}
	for _, profile := range pyTorchProfiles {
		if profile.ownedProjects[normalized] {
			return true
		}
	}
	return false
}

func sourceForDistributionURL(rawURL string, profile SourceProfile) (domain.SourceID, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.RawQuery != "" || parsed.Path == "" {
		return domain.SourceID{}, errors.New("distribution URL is invalid")
	}
	filename := path.Base(parsed.Path)
	if filename == "." || filename == "/" {
		return domain.SourceID{}, errors.New("distribution filename is invalid")
	}
	if strings.EqualFold(parsed.Hostname(), "files.pythonhosted.org") {
		return publicPyPIProfile.source, nil
	}
	for _, host := range profile.distributionHosts {
		if strings.EqualFold(parsed.Hostname(), host) {
			return profile.source, nil
		}
	}
	return domain.SourceID{}, errors.New("distribution URL host is not trusted")
}

func validateDistributionURLForSource(rawURL, filename string, profile SourceProfile, allowHashFragment bool) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.RawQuery != "" || parsed.Path == "" || path.Base(parsed.Path) != filename || filename == "." || filename == "/" {
		return errors.New("distribution URL is invalid")
	}
	if !allowHashFragment && parsed.Fragment != "" {
		return errors.New("distribution URL must not have a fragment")
	}
	source, err := sourceForDistributionURL(rawURL, profile)
	if err != nil {
		return err
	}
	if source == profile.source && IsPyTorchSource(profile.source) {
		base, err := url.Parse(profile.indexURL)
		if err != nil || !strings.HasPrefix(parsed.Path, base.Path) {
			return errors.New("PyTorch distribution URL is outside the selected profile")
		}
	}
	return nil
}

func sourceOwnsProject(profile SourceProfile, source domain.SourceID, project string) bool {
	if IsPyTorchSource(source) {
		normalized, err := NormalizeProjectName(project)
		return err == nil && profile.source == source && profile.ownedProjects[normalized]
	}
	return source == publicPyPIProfile.source && !IsPyTorchOwnedProject(project)
}
