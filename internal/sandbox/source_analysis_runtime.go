package sandbox

import (
	"errors"
	"strconv"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

const sourceAnalysisProfile = ports.SourceAnalysisResourceProfile

// Only controller-owned files and fixed flags enter the analyzer command.
// Neither original source names nor artifact-provided configuration is argv.
func sourceAnalysisCommand() []string {
	core, _ := nativeRuntimeToolIdentity(nativeRuntimeSemgrep)
	rules, _ := nativeRuntimeToolIdentity(nativeRuntimeSemgrepRules)
	return []string{core.guestPath, "-json_nodots", "-use_parmap", "-tree_sitter_only", "-filter_irrelevant_rules", "-no_filter_irrelevant_patterns", "-j", "1", "-rules", rules.guestPath, "-targets", sourceAnalysisRoot + "/" + sourceAnalysisTargetsName, "-timeout", strconv.Itoa(ports.SourceAnalysisRuleTimeoutSeconds), "-max_memory", strconv.Itoa(ports.SourceAnalysisHeapLimitMiB), "-max_match_per_file", "100"}
}

func sourceAnalysisCreateArguments(supply, input *readonlyToolVolume, transaction string, lease ObservationResourceLease) ([]string, error) {
	if _, err := domain.ParseSandboxSessionID(transaction); err != nil || lease.CgroupParent == "" || lease.CPU < 0 || !supply.valid() || !supply.semgrep || supply.source != nil || !supply.sealed || !input.valid() || input.source == nil || !input.sealed || supply.volume.name == input.volume.name {
		return nil, errors.New("source analysis requires separate sealed supply/input and a resource lease")
	}
	toolMount, err := supply.readerMount()
	if err != nil {
		return nil, err
	}
	sourceMount, err := input.readerMount()
	if err != nil {
		return nil, err
	}
	args := pythonResolverCreateArguments("none", nil, []string{"--tmpfs", boundaryHelperMount}, []string{"--mount", toolMount, "--mount", sourceMount})
	for i := 1; i+1 < len(args); i++ {
		switch args[i] {
		case "--memory":
			args[i+1] = strconv.FormatInt(ports.SourceAnalysisMemoryLimit, 10)
		case "--ulimit":
			seconds := strconv.Itoa(ports.SourceAnalysisCPUSeconds)
			args[i+1] = "cpu=" + seconds + ":" + seconds
		}
	}
	bound := []string{"create", "--cpuset-cpus", strconv.Itoa(lease.CPU), "--cgroup-parent", lease.CgroupParent,
		"--label", closureVolumeLabel + "=" + transaction, "--label", closureManifestLabel + "=" + input.source.manifestID,
		"--label", "io.heliopause.observation-phase=" + string(phasePreparation)}
	return append(bound, args[1:]...), nil
}
