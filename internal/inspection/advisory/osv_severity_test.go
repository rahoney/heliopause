package advisory

import (
	"math"
	"strings"
	"testing"
)

func TestCVSSBaseSeverityUsesVersionedVectorWithoutHumanLabels(t *testing.T) {
	for _, tc := range []struct {
		kind, vector string
		want         float64
	}{
		{"CVSS_V2", "AV:N/AC:L/Au:N/C:C/I:C/A:C", 10},
		{"CVSS_V2", "AV:N/AC:L/Au:N/C:N/I:N/A:N", 0},
		{"CVSS_V3", "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", 9.8},
		{"CVSS_V3", "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", 9.8},
		{"CVSS_V3", "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:N", 0},
		{"CVSS_V4", "CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:H/VI:H/VA:H/SC:H/SI:H/SA:H", 10},
		{"CVSS_V4", "CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:N/VI:N/VA:N/SC:N/SI:N/SA:N", 0},
		{"CVSS_V4", "CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:H/VI:H/VA:H/SC:N/SI:N/SA:N", 9.3},
		{"CVSS_V4", "CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:N/VI:N/VA:N/SC:H/SI:H/SA:H", 7.9},
		{"CVSS_V3", "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H/E:U/RL:O/RC:U", 9.8},
		{"CVSS_V4", "CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:H/VI:H/VA:H/SC:H/SI:H/SA:H/E:U", 10},
		{"CVSS_V4", "CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:H/VI:H/VA:H/SC:H/SI:H/SA:H/MVC:N/MVI:N/MVA:N/MSC:N/MSI:N/MSA:N", 10},
	} {
		t.Run(tc.vector, func(t *testing.T) {
			score, err := baseSeverity(osvSeverity{tc.kind, tc.vector})
			if err != nil || math.Abs(score-tc.want) > 0.00001 {
				t.Fatalf("base score=%v want=%v err=%v", score, tc.want, err)
			}
		})
	}
}

func TestCVSSMalformedUnsupportedAndDuplicateMetricsNeverBecomeCleanZero(t *testing.T) {
	for _, value := range []osvSeverity{
		{"unknown", "HIGH"}, {"CVSS_V3", "HIGH"}, {"CVSS_V3", ""},
		{"CVSS_V3", "CVSS:3.1/AV:N/AC:L"},
		{"CVSS_V3", "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H/AV:N"},
		{"CVSS_V3", "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H\n"},
		{"CVSS_V3", "CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:N/VI:N/VA:N/SC:N/SI:N/SA:N"},
		{"CVSS_V4", "CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:H/VI:H/VA:H/SC:H/SI:H/SA:H/MVC:INVALID"},
		{"CVSS_V4", "CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:H/VI:H/VA:H/SC:H/SI:H/SA:H/E:U/E:A"},
		{"CVSS_V2", strings.Repeat("/", 512)}, {"CVSS_V2", strings.Repeat("A", 513)},
	} {
		if _, err := baseSeverity(value); err == nil {
			t.Fatal("invalid/unsupported vector became a score", value)
		}
	}
}

func FuzzBaseSeverity(f *testing.F) {
	f.Add(byte(0), "AV:N/AC:L/Au:N/C:C/I:C/A:C")
	f.Add(byte(1), "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H")
	f.Add(byte(2), "CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:H/VI:H/VA:H/SC:H/SI:H/SA:H")
	f.Add(byte(3), "HIGH")
	f.Fuzz(func(t *testing.T, kind byte, vector string) {
		if len(vector) > 513 {
			return
		}
		types := [4]string{"CVSS_V2", "CVSS_V3", "CVSS_V4", "UNKNOWN"}
		score, err := baseSeverity(osvSeverity{types[kind%4], vector})
		if err == nil && (math.IsNaN(score) || math.IsInf(score, 0) || score < 0 || score > 10) {
			t.Fatal("invalid successful score", score)
		}
	})
}
