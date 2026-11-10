# PNPM Node bin code vector

`pnpm-node-bin-shim.normalized` is a test-only vector, never an executable.
Its code is derived from the MIT-licensed PNPM renderer at commit
`91774e72a04d52ce61db06339696a433d383c764`, with the bin target replaced by
`{{HAA_BIN_TARGET}}` and the optional data-only NODE_PATH block excluded.

The copyright and license terms are preserved in
[PNPM NOTICE](../../../../tools/pnpm/NOTICE). The independent upstream/native
comparison and maintenance boundary are documented in
[B3 §9](../../../../docs/planning/evidence/package-client-quarantine/p2c5b3-pnpm-layout-plan-20261024.md#9-b3b2b--node-bin-shim-전체-code-검증-지도두-번째-검토).
