package advisory

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestCapturedOSVRecordConnectsDecoderToExactVersionAssessment(t *testing.T) {
	body, err := os.ReadFile("testdata/ghsa-462w-v97r-4m45.json")
	if err != nil {
		t.Fatal(err)
	}
	var identity struct {
		ID       string
		Modified time.Time
	}
	if err := json.Unmarshal(body, &identity); err != nil {
		t.Fatal(err)
	}
	record, err := decodeRecord(body, advisoryReference{identity.ID, identity.Modified})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		version string
		want    versionImpact
	}{{"2.10", impactAffected}, {"2.10.1", impactUnaffected}} {
		got, err := assessRecord(osvPackage{"PyPI", "Jinja2", tc.version}, record)
		if err != nil || got != tc.want {
			t.Fatal("captured advisory/fixed boundary mismatch", tc.version, got, err)
		}
	}
}

func TestVersionRangeEvaluatesOpenClosedAndReintroducedIntervals(t *testing.T) {
	for _, tc := range []struct {
		name, ecosystem, version, kind string
		events                         []osvEvent
		want                           versionImpact
	}{
		{"fixed excluded", "npm", "1.2.3", "SEMVER", []osvEvent{{Introduced: "0"}, {Fixed: "1.2.3"}}, impactUnaffected},
		{"before fix", "npm", "1.2.2", "SEMVER", []osvEvent{{Introduced: "0"}, {Fixed: "1.2.3"}}, impactAffected},
		{"prerelease before fix", "npm", "1.2.3-rc.1", "SEMVER", []osvEvent{{Introduced: "0"}, {Fixed: "1.2.3"}}, impactAffected},
		{"start included", "Go", "v1.2.3", "SEMVER", []osvEvent{{Introduced: "1.2.3"}}, impactAffected},
		{"below start", "Go", "v1.2.2", "SEMVER", []osvEvent{{Introduced: "1.2.3"}}, impactUnaffected},
		{"gap safe", "crates.io", "2.0.0", "SEMVER", []osvEvent{{Introduced: "0"}, {Fixed: "1.0.0"}, {Introduced: "3.0.0"}}, impactUnaffected},
		{"reintroduced", "crates.io", "3.0.0", "SEMVER", []osvEvent{{Introduced: "0"}, {Fixed: "1.0.0"}, {Introduced: "3.0.0"}}, impactAffected},
		{"last affected included", "npm", "1.2.3", "SEMVER", []osvEvent{{Introduced: "1.0.0"}, {LastAffected: "1.2.3"}}, impactAffected},
		{"limit excluded", "npm", "1.2.3", "SEMVER", []osvEvent{{Introduced: "1.0.0"}, {Limit: "1.2.3"}}, impactUnaffected},
		{"python prerelease", "PyPI", "1.0rc1", "ECOSYSTEM", []osvEvent{{Introduced: "0"}, {Fixed: "1.0"}}, impactAffected},
		{"python post fixed", "PyPI", "1.0.post1", "ECOSYSTEM", []osvEvent{{Introduced: "0"}, {Fixed: "1.0.post1"}}, impactUnaffected},
		{"python trailing zeros", "PyPI", "1.0.0", "ECOSYSTEM", []osvEvent{{Introduced: "1"}, {LastAffected: "1.0"}}, impactAffected},
		{"unknown git ordering", "Go", "v1.2.3", "GIT", []osvEvent{{Introduced: "abc"}}, impactUnknown},
		{"python differently declared ordering", "PyPI", "1.0.0", "SEMVER", []osvEvent{{Introduced: "0"}}, impactUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := assessRange(osvPackage{tc.ecosystem, "fixture", tc.version}, osvRange{Type: tc.kind, Events: tc.events})
			if err != nil || got != tc.want {
				t.Fatalf("range impact=%d want=%d err=%v", got, tc.want, err)
			}
		})
	}
}

func TestVersionRangeRejectsMalformedIntervals(t *testing.T) {
	for _, events := range [][]osvEvent{
		nil, {{Fixed: "1.0.0"}}, {{Introduced: "1.0.0"}, {Introduced: "2.0.0"}},
		{{Introduced: "2.0.0"}, {Fixed: "1.0.0"}}, {{Introduced: "1.0.0"}, {Fixed: "1.0.0"}},
		{{Introduced: "0"}, {Fixed: "2.0.0"}, {Introduced: "1.0.0"}},
		{{Introduced: "0"}, {LastAffected: "2.0.0"}, {Introduced: "2.0.0"}},
		{{Introduced: "0"}, {Fixed: "1.0.0"}, {Introduced: "0"}},
		{{Introduced: "0"}, {Fixed: "not-semver"}},
	} {
		if got, err := assessRange(osvPackage{"npm", "fixture", "1.0.0"}, osvRange{Type: "SEMVER", Events: events}); err == nil || got != impactUnknown {
			t.Fatalf("malformed interval became clean: %#v %d %v", events, got, err)
		}
	}
}

func TestRecordAssessmentRequiresMappedSubjectAndSeparatesUnknown(t *testing.T) {
	query := osvPackage{"PyPI", "Fixture_Project", "1.0.0"}
	record := osvRecord{Affected: []osvAffected{{Package: osvPackage{Ecosystem: "PyPI", Name: "fixture-project"}, Versions: []string{"1.0"}}}}
	if got, err := assessRecord(query, record); err != nil || got != impactAffected {
		t.Fatal("normalized explicit Python version match lost", got, err)
	}
	record.Affected[0].Versions = []string{"2.0"}
	if got, err := assessRecord(query, record); err != nil || got != impactUnaffected {
		t.Fatal("explicit unmatched version lost", got, err)
	}
	record.Affected[0].Versions = nil
	if got, err := assessRecord(query, record); err != nil || got != impactUnknown {
		t.Fatal("missing version evidence became clean", got, err)
	}
	record.Affected[0].Package.Name = "other-project"
	if got, err := assessRecord(query, record); err == nil || got != impactUnknown {
		t.Fatal("wrong provider subject became clean", got, err)
	}
	stamp := time.Now()
	record.Withdrawn = &stamp
	if got, err := assessRecord(query, record); err != nil || got != impactWithdrawn {
		t.Fatal("withdrawal collapsed into vulnerability or generic safe result", got, err)
	}
	query.Version = "1.0+cpu"
	if got, err := assessRecord(query, record); err != nil || got != impactWithdrawn {
		t.Fatal("local build lost withdrawal", got, err)
	}
}

func TestPackageMatchingReusesPublicNameRules(t *testing.T) {
	for _, tc := range []struct {
		query, subject osvPackage
		want           bool
	}{
		{osvPackage{"crates.io", "serde_json", "1.0.0"}, osvPackage{Ecosystem: "crates.io", Name: "Serde-Json"}, true},
		{osvPackage{"crates.io", "serde", "1.0.0"}, osvPackage{Ecosystem: "crates.io", Name: "ſerde"}, false},
		{osvPackage{"PyPI", "fixture.project", "1.0"}, osvPackage{Ecosystem: "PyPI", Name: "Fixture_Project"}, true},
		{osvPackage{"PyPI", "fixture", "1.0"}, osvPackage{Ecosystem: "npm", Name: "fixture"}, false},
		{osvPackage{"Go", "example.org/Module", "v1.0.0"}, osvPackage{Ecosystem: "Go", Name: "example.org/module"}, false},
	} {
		if got := samePackage(tc.query, tc.subject); got != tc.want {
			t.Fatalf("subject mapping=%v want=%v query=%#v subject=%#v", got, tc.want, tc.query, tc.subject)
		}
	}
}

func TestOfficialLocalBuildDoesNotHideUpstreamEndpoint(t *testing.T) {
	query := osvPackage{"PyPI", "torch", "2.6.0+cpu"}
	for _, tc := range []struct {
		name     string
		versions []string
		events   []osvEvent
		want     versionImpact
	}{
		{"public last affected", nil, []osvEvent{{Introduced: "0"}, {LastAffected: "2.6.0"}}, impactUnknown},
		{"public enumerated only", []string{"2.6.0"}, nil, impactUnknown},
		{"exact local affected", nil, []osvEvent{{Introduced: "0"}, {Fixed: "2.7.0"}}, impactAffected},
		{"local fix", nil, []osvEvent{{Introduced: "0"}, {Fixed: "2.6.0+cpu"}}, impactUnknown},
		{"public fix", nil, []osvEvent{{Introduced: "0"}, {Fixed: "2.6.0"}}, impactUnaffected},
		{"exact local enumerated", []string{"2.6.0+cpu"}, nil, impactAffected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry := osvAffected{Package: osvPackage{Ecosystem: "PyPI", Name: "torch"}, Versions: tc.versions}
			if tc.events != nil {
				entry.Ranges = []osvRange{{Type: "ECOSYSTEM", Events: tc.events}}
			}
			record := osvRecord{ID: "GHSA-aaaa-bbbb-cccc", Affected: []osvAffected{entry}}
			if got, err := assessRecord(query, record); err != nil || got != tc.want {
				t.Fatal(got, tc.want, err)
			}
		})
	}
}
