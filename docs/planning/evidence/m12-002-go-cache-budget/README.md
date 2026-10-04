# M12-002 Go verified cache file budget

The user approved 20,000 aggregate cache files after the independently authenticated
gRPC 1.76.0 graph reached 41 modules and exceeded the former 10,000-file cap.
[The prior source correction](../m12-002-go-source-correction/README.md) owns the
original failure and complete preflight measurement.

[Receipt](./result.json) binds the new source, binaries and actual CLI results.
The production change is one constant: 20,000 files. The existing derived bound
becomes 40,000 filesystem entries including directories. Aggregate bytes remain
512 MiB, receipts 4 MiB, and each archive remains limited to 10,000 entries.

The same final aggregate fixture with only the old cache source overlaid
[failed](./before.log); the new source [passes](./after.log) at 20,000 regular
files. An extra file prevents reuse. A complete 20,002-file graph fails before
publication and leaves no partial cache. Existing authenticity, Evidence,
full inventory, path, symlink and approval checks remain in force.

The maintained actual CLI tests passed for dependency-free Download, pflag
Get/retained Download and gRPC Get/retained Download. The gRPC gate explicitly
requires all 41 modules and remains a positive integration test. It does not
replace the still-missing observed offline build and final M12-002 qualification.
No new CPU/CUDA full or native macOS PASS is claimed.

Canonical security and docs passed in the shared workspace. Canonical quick
passed on an isolated Linux filesystem checkout with all 563 tracked candidate
and new evidence files matched by SHA-256. The original OneDrive invocation
timed out in the default test command and remains a failed check. A bounded
focused diagnostic there printed test PASS and package ok but the Go command
did not exit; the count=1 focused run exited normally. These observations do
not establish a product defect or a precise filesystem timing mechanism. No
quality budget, skip, production input or test assertion was weakened.
