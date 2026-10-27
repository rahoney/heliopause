#!/bin/bash
# Hosted-CI-only acquired DATA probes. No installed runtime is created here.
set -euo pipefail
case "$#:${1-}" in
  1:inspect|1:graph|1:namespace) ;;
  *) echo 'unsupported pipx CI profile' >&2; exit 2 ;;
esac
pipx_profile=$1
test "$(id -u)" != 0
test -n "${RUNNER_TEMP:-}"
test "${RUNNER_TEMP:0:1}" = /
pipx_root=$PWD
pipx_work=$(mktemp -d "$RUNNER_TEMP/haa-pipx.XXXXXXXX")
pipx_cleanup() {
  local primary=$? cleanup_status=0
  trap - EXIT
  rm -rf -- "$pipx_work" || cleanup_status=1
  if ((cleanup_status)); then echo 'pipx CI input cleanup unproven' >&2; fi
  if ((primary == 0)); then primary=$cleanup_status; fi
  exit "$primary"
}
trap pipx_cleanup EXIT
umask 077
go run ./scripts/generate-runtime-lock.go -check
go run ./scripts/update-runtime fetch-pipx-supply --root "$pipx_root" --output "$pipx_work/supply"
go run ./scripts/update-runtime materialize-pipx --root "$pipx_root" --input "$pipx_work/supply" --output "$pipx_work/pipx-runtime-linux-amd64.zip" --notices-output "$pipx_work/pipx-NOTICE.txt"

# This checks trusted Go test dispatch, never artifact-supplied completion.
# A child PASS cannot replace its parent; empty filters/SKIP/duplicates fail.
pipx_ci_run() {
  local package=$1 expected=$2 label=$3 status=0
  shift 3
  local log="$pipx_work/$label.log"
  (
    cd "$package"
    "$@"
  ) >"$log" 2>&1 || status=$?
  cat -- "$log"
  if ((status)); then return "$status"; fi
  awk -v wanted="$expected" '
    BEGIN { n=split(wanted,names,","); for(i=1;i<=n;i++) expected[names[i]]=1 }
    /--- SKIP:/ { bad=1 }
    /^--- PASS: / { if($3 in expected) seen[$3]++ }
    END { if(bad) exit 1; for(name in expected) if(seen[name]!=1) exit 1 }
  ' "$log"
}

case "$pipx_profile" in
  inspect)
    pipx_ci_run internal/bootstrap TestPipxAcquiredRuntimeRequiredInspectionIntegration inspect env HELOX_PIPX_INSPECTION_INTEGRATION=1 HELOX_PIPX_RUNTIME_ZIP="$pipx_work/pipx-runtime-linux-amd64.zip" /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=6m -test.run '^TestPipxAcquiredRuntimeRequiredInspectionIntegration$'
    ;;
  graph)
    # An explicit historical producer SBOM is rebased only inside the existing
    # unsigned source/staging fixture. Current full producer census is separate.
    cp -- "$pipx_root/docs/planning/evidence/m13-003-release-build/v0.1.0/helox-release-sbom.cdx.json" "$pipx_work/helox-release-sbom.cdx.json"
    pipx_ci_run internal/artifact/githubrelease TestPipxActualCompleteGraphCurrentEvidencePolicyAndStaging graph env HELOX_PIPX_GRAPH_INTEGRATION=1 HELOX_PIPX_RELEASE_ACTUAL_DIR="$pipx_work" /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=8m -test.run '^TestPipxActualCompleteGraphCurrentEvidencePolicyAndStaging$'
    ;;
  namespace)
    # Auth/crypto filesystem doubles only; all root writes remain in private
    # mount+network namespaces. No service or native runtime is activated.
    pipx_ci_run internal/releaseactivation TestPipxBootstrapCopyRecoveryAndCleanupInDisposableNamespace,TestInitialNativeFilesInDisposableNamespace,TestProtectedStageInDisposableNamespace bootstrap sudo -n env HELOX_PIPX_ROOT_DATA_NAMESPACE=1 HELOX_BOOTSTRAP_NAMESPACE=1 HELOX_STAGE_NAMESPACE=1 HELOX_PIPX_RELEASE_ACTUAL_DIR="$pipx_work" unshare --mount --net --fork "$RUNNER_TEMP/helox-releaseactivation-test" -test.v -test.failfast -test.timeout=3m -test.run '^(TestPipxBootstrapCopyRecoveryAndCleanupInDisposableNamespace|TestInitialNativeFilesInDisposableNamespace|TestProtectedStageInDisposableNamespace)$'
    pipx_ci_run internal/releaseinstall TestInterruptedSystemCopyInDisposableNamespace pending sudo -n env HELOX_COPY_INTERRUPTION_NAMESPACE=1 unshare --mount --net --fork "$RUNNER_TEMP/helox-releaseinstall-test" -test.v -test.failfast -test.timeout=3m -test.run '^TestInterruptedSystemCopyInDisposableNamespace$'
    ;;
esac
