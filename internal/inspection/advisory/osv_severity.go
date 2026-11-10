package advisory

import (
	"errors"
	"math"
	"strings"

	cvss20 "github.com/pandatix/go-cvss/20"
	cvss30 "github.com/pandatix/go-cvss/30"
	cvss31 "github.com/pandatix/go-cvss/31"
	cvss40 "github.com/pandatix/go-cvss/40"
)

var errSeverity = errors.New("advisory severity vector is unsupported or invalid")

// baseSeverity interprets a bounded provider fact, not a final security verdict.
// A parser fault must remain an error; it must never become a clean zero score.
func baseSeverity(value osvSeverity) (score float64, resultErr error) {
	defer func() {
		if recover() != nil {
			score, resultErr = 0, errSeverity
		}
	}()
	if value.Score == "" || len(value.Score) > 512 {
		return 0, errSeverity
	}
	for _, ch := range value.Score {
		if ch < 33 || ch > 126 {
			return 0, errSeverity
		}
	}
	switch value.Type {
	case "CVSS_V2":
		parsed, err := cvss20.ParseVector(value.Score)
		if err != nil || parsed == nil {
			return 0, errSeverity
		}
		score = parsed.BaseScore()
	case "CVSS_V3":
		switch {
		case strings.HasPrefix(value.Score, "CVSS:3.0/"):
			parsed, err := cvss30.ParseVector(value.Score)
			if err != nil || parsed == nil {
				return 0, errSeverity
			}
			score = parsed.BaseScore()
		case strings.HasPrefix(value.Score, "CVSS:3.1/"):
			parsed, err := cvss31.ParseVector(value.Score)
			if err != nil || parsed == nil {
				return 0, errSeverity
			}
			score = parsed.BaseScore()
		default:
			return 0, errSeverity
		}
	case "CVSS_V4":
		parsed, err := cvss40.ParseVector(value.Score)
		if err != nil || parsed == nil {
			return 0, errSeverity
		}
		// First validate the entire vector. Then score its mandatory base
		// metrics alone; optional threat/environment values cannot lower it.
		parts := strings.Split(value.Score, "/")
		if len(parts) > 12 {
			parsed, err = cvss40.ParseVector(strings.Join(parts[:12], "/"))
			if err != nil || parsed == nil {
				return 0, errSeverity
			}
		}
		score = parsed.Score()
	default:
		return 0, errSeverity
	}
	if math.IsNaN(score) || math.IsInf(score, 0) || score < 0 || score > 10 {
		return 0, errSeverity
	}
	return score, nil
}
