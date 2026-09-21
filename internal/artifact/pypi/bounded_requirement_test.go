package pypi

import (
	"strings"
	"testing"
)

func TestBoundedCUDARequirements(t *testing.T) {
	for _, value := range []string{
		"nvidia-cublas==13.1.1.3.*",
		"nvidia-nvjitlink>=13.0.88,<14",
		"cuda-toolkit[cublas,cudart,cufft,cufile,cupti,curand,cusolver,cusparse,nvrtc,nvtx]==12.6.3; platform_system == \"Linux\"",
		"cuda-toolkit[cublas,nvjitlink,nvtx]>=13.0.88,<14; platform_system == 'Linux'",
	} {
		if _, err := ParseBoundedRequirement(value); err != nil {
			t.Fatalf("ParseBoundedRequirement(%q): %v", value, err)
		}
	}
	requirements := []BoundedRequirement{}
	for _, value := range []string{"nvidia-nvjitlink", "nvidia-nvjitlink>=13.0.88,<14"} {
		requirement, err := ParseBoundedRequirement(value)
		if err != nil {
			t.Fatal(err)
		}
		requirements = append(requirements, requirement)
	}
	if !CandidateSatisfiesBoundedRequirements("13.1.0", requirements) || CandidateSatisfiesBoundedRequirements("14.0", requirements) {
		t.Fatal("aggregate range candidate validation is wrong")
	}
}

func TestBoundedCUDARequirementsRejectUnsupportedSyntax(t *testing.T) {
	for _, value := range []string{
		"cuda-toolkit[evil]==12.6.3; platform_system == \"Linux\"",
		"cuda-toolkit[cublas,cublas]==12.6.3; platform_system == \"Linux\"",
		"cuda-toolkit[cublas]==12.6.3; platform_system != \"Linux\"",
		"cuda-toolkit[cublas]==12.6.3; sys_platform == \"linux\"",
		"nvidia-cublas==13.*", "nvidia-cublas==13.1.1", "nvidia-cublas~=13.1", "nvidia-cublas @ https://bad.example/x", "nvidia-cublas>=13,,<14",
	} {
		if _, err := ParseBoundedRequirement(value); err == nil {
			t.Fatalf("accepted unsupported requirement %q", value)
		}
	}
}

func TestBoundedExactAndRangeCollisions(t *testing.T) {
	exact, _ := ParseBoundedRequirement("nvidia-cublas==13.1.1.3.*")
	bare, _ := ParseBoundedRequirement("nvidia-cublas")
	rangeReq, _ := ParseBoundedRequirement("nvidia-nvjitlink>=13.0.88,<14")
	if !CandidateSatisfiesBoundedRequirements("13.1.1.3", []BoundedRequirement{exact, bare}) || CandidateSatisfiesBoundedRequirements("13.1.2", []BoundedRequirement{exact, bare}) || !CandidateSatisfiesBoundedRequirements("13.0.88", []BoundedRequirement{rangeReq}) {
		t.Fatal("modern collision validation is wrong")
	}
	conflicting, _ := ParseBoundedRequirement("nvidia-cublas==12.6.4.1.*")
	if CandidateSatisfiesBoundedRequirements("13.1.1.3", []BoundedRequirement{exact, conflicting}) {
		t.Fatal("incompatible exact pins were accepted")
	}
	outside, _ := ParseBoundedRequirement("nvidia-cublas>=14,<15")
	if CandidateSatisfiesBoundedRequirements("13.1.1.3", []BoundedRequirement{exact, outside}) {
		t.Fatal("exact pin outside range was accepted")
	}
}

func TestBoundedCUDAExtrasSurviveAggregation(t *testing.T) {
	left, err := ParseBoundedRequirement(`cuda-toolkit[cublas,cudart]==12.6.3; platform_system == "Linux"`)
	if err != nil {
		t.Fatal(err)
	}
	right, err := ParseBoundedRequirement(`cuda-toolkit[nvrtc,cublas]>=12,<14; platform_system == "Linux"`)
	if err != nil {
		t.Fatal(err)
	}
	aggregated, err := AggregateBoundedRequirements("cuda-toolkit", []BoundedRequirement{left, right})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := aggregated.Request(), `cuda-toolkit[cublas,cudart,nvrtc]==12.6.3,>=12,<14; platform_system == "Linux"`; got != want {
		t.Fatalf("aggregate request = %q, want %q", got, want)
	}
	if got, want := strings.Join(aggregated.Extras(), ","), "cublas,cudart,nvrtc"; got != want {
		t.Fatalf("aggregate extras = %q, want %q", got, want)
	}
}

func TestModernPyTorchRequirementFixtures(t *testing.T) {
	profiles := map[string][]string{
		"cpu":   {"filelock", "typing-extensions>=4.10", "sympy>=1.13.3"},
		"cu126": {"nvidia-cublas-cu12==12.6.4.1.*", "nvidia-cublas-cu12", "nvidia-nvjitlink-cu12>=12.6.85,<13", "nvidia-nvjitlink-cu12"},
		"cu130": {"nvidia-cublas==13.1.1.3.*", "nvidia-cublas", "nvidia-cuda-nvrtc==13.0.88.*", "nvidia-cusparse==12.6.3.3.*", "nvidia-nvjitlink>=13.0.88,<14"},
		"cu132": {"nvidia-cublas==13.2.0.1.*", "nvidia-cublas", "nvidia-nvjitlink>=13.2.0,<14", "nvidia-nvjitlink"},
	}
	for profile, requirements := range profiles {
		for _, value := range requirements {
			if _, err := ParseBoundedRequirement(value); err != nil {
				t.Fatalf("%s fixture %q: %v", profile, value, err)
			}
		}
	}
}
