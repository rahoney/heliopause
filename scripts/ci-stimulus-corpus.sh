#!/bin/bash
# The existing parent owns the tagged bootstrap client and helper lifecycle.
set -euo pipefail
test "$#" = 0
test "$(id -u)" != 0
test -n "${RUNNER_TEMP:-}"
test "${RUNNER_TEMP:0:1}" = /
(
  cd internal/bootstrap
  HELOX_STIMULUS_CORPUS_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=6m -test.run '^TestLinuxStimulus(Node|Python)(Policy|Entry)CorpusIntegration$'
) | tee "$RUNNER_TEMP/stimulus-corpus.log"
awk '
BEGIN {
 split("TestLinuxStimulusNodePolicyCorpusIntegration TestLinuxStimulusNodeEntryCorpusIntegration TestLinuxStimulusPythonPolicyCorpusIntegration TestLinuxStimulusPythonEntryCorpusIntegration", names, " ")
 for (i in names) parents[names[i]]=0
 split("normal normal-config-read conditional-chain delayed-chain detached-child", cases, " ")
 for (i in cases) { expected["stimulus_node/" cases[i]]=0; expected["stimulus_python/" cases[i]]=0 }
 expected["stimulus_node_entry/normal"]=0; expected["stimulus_node_entry/activate"]=0
 expected["stimulus_python_entry/normal"]=0; expected["stimulus_python_entry/activate"]=0
}
/no tests to run|--- (SKIP|FAIL):/ { bad=1 }
/^--- PASS:/ { if (!($3 in parents) || parents[$3]++) bad=1 }
/(stimulus_node|stimulus_python)(_entry)? case=/ {
 name=$3; sub(/^case=/,"",name); key=$2 "/" name
 if (!(key in expected) || expected[key]++) bad=1
 if ($2 ~ /_entry$/) {
   complete=(name=="normal" ? "true" : "false")
   if ($4!="lifecycle_policy=ALLOW" || $5!="runtime_complete=" complete || $6!="source_unchanged=true" || $7!="host_install=NOT_PERFORMED") bad=1
 } else {
   normal=(name=="normal" || name=="normal-config-read"); allow=(normal && !(name=="normal-config-read" && $2=="stimulus_node"))
   network=$6; sub(/^network=/,"",network)
   if ($4!~/^current_checks=[1-9][0-9]*$/ || $5!="stored_states=2" || network!~/^[0-9]+$/ || (normal ? network!=0 : network==0) || $7!="policy=" (allow ? "ALLOW" : "MANUAL_REVIEW") || $8!="approval=" (allow ? "true" : "false")) bad=1
 }
}
END { if (bad) exit 1; for (name in parents) if (parents[name]!=1) exit 1; for (name in expected) if (expected[name]!=1) exit 1 }
' "$RUNNER_TEMP/stimulus-corpus.log"
