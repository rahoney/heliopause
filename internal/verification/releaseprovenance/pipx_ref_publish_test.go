package releaseprovenance

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"testing"
)

func TestPipxDependencyRefPublisherRequiresNullEnvironment(t *testing.T) {
	body, id := actualPipxDependency(t, "pluggy")
	for _, value := range []any{"", false, 0, []any{}, map[string]any{}, "release"} {
		changed := pipxProofMutation(t, body, func(_, group, _ map[string]any) { group["publisher"].(map[string]any)["environment"] = value })
		if _, err := VerifyPipxDependencyPublish(context.Background(), changed, id); err == nil {
			t.Fatal("non-null environment accepted", value)
		}
	}
	if !validPyPIPublisherEnvironment(json.RawMessage("null"), "") || validPyPIPublisherEnvironment(nil, "") || validPyPIPublisherEnvironment(json.RawMessage("null"), "deploy") {
		t.Fatal("environment shape boundary changed")
	}
	policy, _ := pipxDependencyPolicy("pytest", "9.1.1")
	if policy.ref != "refs/heads/release-9.1.1" || policy.trigger != "workflow_dispatch" || policy.environment != "deploy" {
		t.Fatal("pytest release publisher was replaced with synthetic tag")
	}
}

func TestPipxDependencyRefPublisherSignedClaims(t *testing.T) {
	body, id := actualPipxDependency(t, "pluggy")
	policy, _ := pipxDependencyPolicy(id.Project, id.Version)
	_, der, err := decodePyPIPublish(body, id.WheelFilename(), id.WheelSHA256, policy)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	if !validPyPIPublisherClaims(cert, policy) {
		t.Fatal("actual signed ref subject refused")
	}
	environmentOID := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 23}
	subjectOID := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 24}
	for _, mode := range []string{"empty-environment", "other-environment", "missing-subject", "duplicate-subject", "wrong-ref", "environment-subject", "wrong-trigger", "branch-ref"} {
		t.Run(mode, func(t *testing.T) {
			changed := *cert
			changed.Extensions = nil
			expected := policy
			for _, extension := range cert.Extensions {
				if extension.Id.Equal(subjectOID) {
					switch mode {
					case "missing-subject":
						continue
					case "duplicate-subject":
						changed.Extensions = append(changed.Extensions, extension)
					case "wrong-ref":
						extension.Value, _ = asn1.Marshal("repo:pytest-dev/pluggy:ref:refs/tags/other")
					case "environment-subject":
						extension.Value, _ = asn1.Marshal("repo:pytest-dev/pluggy:environment:release")
					}
				}
				changed.Extensions = append(changed.Extensions, extension)
			}
			if mode == "empty-environment" || mode == "other-environment" {
				value := ""
				if mode == "other-environment" {
					value = "release"
				}
				encoded, err := asn1.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				changed.Extensions = append(changed.Extensions, pkix.Extension{Id: environmentOID, Value: encoded})
			}
			if mode == "wrong-trigger" {
				expected.trigger = "pull_request"
			}
			if mode == "branch-ref" {
				expected.ref = "refs/heads/main"
			}
			if validPyPIPublisherClaims(&changed, expected) {
				t.Fatal("ref publisher claim mutation accepted")
			}
		})
	}
}
