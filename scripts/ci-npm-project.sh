#!/bin/bash
# The existing CI parent owns the current tagged client/helper and restoration.
set -euo pipefail
test "$#" = 1
test "$(id -u)" != 0
test -n "${RUNNER_TEMP:-}"
test "${RUNNER_TEMP:0:1}" = /
case "$1" in
  native)
    (
      cd internal/sandbox
      HELOX_NPM_PROJECT_SELECTION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=4m -test.run '^TestNPMPublicWholeProjectSelectionIntegration$'
    ) | tee "$RUNNER_TEMP/npm-project-native.log"
    awk '
BEGIN { split("install-initial ci-frozen ci-incompatible-node install-changed peer-metadata-only declared-peer optional-platform", names, " "); for (i in names) expected[names[i]]=1 }
/--- (SKIP|FAIL):/ { bad=1 }
/^--- PASS:/ { if ($3!="TestNPMPublicWholeProjectSelectionIntegration" || parents++) bad=1 }
/npm_project_native case=/ {
  name=$3; sub(/^case=/,"",name)
  if (!(name in expected) || seen[name]++ || $0 !~ /^[[:space:]]+npm_project_resolver_real_test[.]go:[0-9]+: npm_project_native case=/) bad=1
  if (name=="ci-incompatible-node" && $4!="denied=true") bad=1
  if (name!="ci-incompatible-node" && $4!~/^placements=[0-9]+$/) bad=1
  count++
}
END { if (bad || parents!=1 || count!=7) exit 1; for (name in expected) if (seen[name]!=1) exit 1 }
' "$RUNNER_TEMP/npm-project-native.log"
    ;;
  project)
    (
      cd internal/bootstrap
      HELOX_NPM_PROJECT_QUARANTINE_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=9m -test.run '^TestNPMPublic(WholeProjectQuarantine|ProjectWorkflow|ProjectRootCLI|ProjectCLI)Integration$'
    ) | tee "$RUNNER_TEMP/npm-project-workflow.log"
    awk '
BEGIN {
 split("TestNPMPublicWholeProjectQuarantineIntegration TestNPMPublicProjectWorkflowIntegration TestNPMPublicProjectRootCLIIntegration TestNPMPublicProjectCLIIntegration", names, " "); for (i in names) expected[names[i]]=1
 quarantine["nested"]=3; quarantine["bin"]=1; quarantine["peer"]=3
 root["postinstall"]="COMPLETED"; root["dependencies"]="INCOMPLETE"; root["postprepare"]="INCOMPLETE"; root["empty-dependencies"]="COMPLETED"
 root["conditional-network"]="COMPLETED"
}
/--- (SKIP|FAIL):/ { bad=1 }
/^--- PASS:/ { if (!($3 in expected) || parents[$3]++) bad=1; parent_count++ }
/npm_project_quarantine case=/ {
 name=$3; sub(/^case=/,"",name)
 if (!(name in quarantine) || inspected[name]++ || $4!="archives=" quarantine[name] || $6!="required_per_archive=3" || $7!="whole_source/trace/evidence/Policy=COMPLETE" || $9!="cache/atomic_temp_target/retained_approval/tamper_refusal=COMPLETE") bad=1
}
/npm_workflow intent=/ {
 workflow++; expected_intent=(workflow%2 ? "install" : "ci"); expected_count=(workflow<=2 ? 1 : 3)
 if ($3!="intent=" expected_intent || $4!="archives=" expected_count || $5!="fresh_operation=true" || $6!="atomic_temp_target=true") bad=1
}
/npm_workflow install\/ci\/edited_install\/ci=PASS/ { if ($4!="ci_mismatch=REFUSED" || $5!="old_target/user_file=PRESERVED" || terminal++) bad=1 }
/npm_root_cli case=/ {
 name=$3; sub(/^case=/,"",name)
 if (!(name in root) || roots[name]++ || $4!="local_identity=true" || $5!="required_checks=3" || $6!="lifecycle=" root[name] || $7!="installation=REFUSED" || $8!="original_target=PRESERVED") bad=1
}
/npm_root_hook_exit=37 original_cause=PRESERVED/ { exits++ }
/npm_root_conditional network=OBSERVED stored_states=2 original_target=PRESERVED/ { conditional++ }
/npm_public_cli operation=/ {
 name=$3; sub(/^operation=/,"",name)
 if ((name!="install" && name!="ci") || public[name]++ || $4!="Policy=ALLOW" || $5!="publication=COMPLETED" || $6!="root=NOT_REQUIRED") bad=1
}
END {
 if (bad || parent_count!=4 || workflow!=4 || terminal!=1 || exits!=2 || conditional!=1 || public["install"]!=1 || public["ci"]!=1) exit 1
 for (name in expected) if (parents[name]!=1) exit 1
 for (name in quarantine) if (inspected[name]!=1) exit 1
 for (name in root) if (roots[name]!=1) exit 1
}' "$RUNNER_TEMP/npm-project-workflow.log"
    ;;
  *) echo 'unsupported npm project CI profile' >&2; exit 2 ;;
esac
