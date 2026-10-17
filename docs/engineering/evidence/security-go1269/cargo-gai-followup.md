# Cargo resolver address-selection configuration

CI307, run38274326838 at `2eea071`, fails
`TestLinuxCargoTransitiveBuildCLIIntegration` during private dependency
selection, before the build. The first fault is
`COMMAND / STREAM_FAULT / OPEN_RESULT_CLASSIFICATION_IMAGE`.

The authenticated observer reports CONTROL, DIRECT_EXEC_ROOT, pinned kernel
CARGO, OCI root `/`, and read flags557056. The image locator4ed561d866fb8230
matches `/etc/gai.conf`; the executable locator05aa2efcccd45bfb matches the
locked Cargo executable. Both matches are unique in the preserved data-only
catalogue for the unchanged Rust1.99.0 image digest. These diagnostic hashes
interpret the error; they never authorize access.

`IsPinnedCargoResolverRead` already handles `/etc/host.conf` and
`/etc/nsswitch.conf`, but omits `/etc/gai.conf`. The latter configures glibc's
address-selection ordering ([manual](https://man7.org/linux/man-pages/man5/gai.conf.5.html)).
The previous successful network run does not prove every conditional runtime
read occurred. The exact DNS response that triggered this remote read was not
captured, so its timing is not inferred.

The observer, latch suite, actual bootstrap integration test and runtime lock
are byte-identical to both the baseline main and CI305's successful source.
This is separate from the host Go vulnerability and the APT mirror stall.
CI307 completes APT acquisition and the previously failing Go transitive
download passes82.80s. Required fails because the Linux job fails; cleanup
preserves original1 and reports CONFIRMED_STOPPED/RESTORED.

The bounded repair reuses the Cargo resolver classifier for exactly this
immutable OCI configuration read. Existing Cargo image, expected process
group/start, CONTROL/direct-root, consumed admission, active control network,
read-only flags and exact root-anchor checks remain. No ARTIFACT permission,
network endpoint, generic `/etc` allowance, runtime lock or M12 contract changes.

The same final Cargo DNS boundary fixture fails against the old classifier
(exit1) and passes with the one-line repair. Both builds use locked gVisor
7c6199801fd233d6d55309af4645d4746a077de7, its unchanged HAA patch, and verified
Bazel8.3.1. The fixed observer builds and all102 existing latch groups pass,
including wrong roles, provenance, images, starts, admission state, mounts,
profiles and writes. The before log SHA-256 is
`0a125f685874ff8df33977dd18820712dd5f059b0fbb1720a9eeae5b7e0fd977`;
the full after log is
`22f5b1f6f107eb1e26e881966a3522fe36a73be9546b26f427c18da076295cf6`.

Canonical Quick (build, default tests, vet, Staticcheck and workflow), security,
vulnerability and documentation checks pass under actual Go1.26.9. The exact
Rust image digest remains
`sha256:59037199c44290f2befcdd58dcc540164763fc296950255aaefeef096a1866b0`;
the preserved data-only catalogue SHA-256 is
`0e927e4d000bba2e3464e9b98cfb52b1e07f84da4e9dd87b01c718cfec1375e3`.
No installed local runtime was replaced. These focused regressions do not
replace actual Cargo and common-consumer Linux/Required CI. CI308 subsequently
passed all ten jobs and45 Linux consumers, including Cargo transitive37.65s and
PyTorch CPU full282.09s. PR30 then merged and main CI38328099413 also passed. The change is restricted to Cargo resolver
CONTROL reads; there is no direct CUDA-profile change requiring a new GPU E2E
campaign. Existing full latches cover the shared classifier's negative boundary.
Original logs, source snapshots, exact build inputs and receipts are preserved
under `/tmp/haa-security-go1269-validation/cargo-gai/`; CI307's full raw is
`/tmp/haa-security-go1269-validation/remote-38274326838-job114896785383.log`.
