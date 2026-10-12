# Final CI APT mirror follow-up

CI38263676518 (CI306), Linux job114860325477, was still running after80 minutes.
The user supplied its live tail: repeated `Ign:` entries for
`http://azure.archive.ubuntu.com/ubuntu` and successful `Hit:`/`Get:` entries
for `https://archive.ubuntu.com/ubuntu`. These are package-index acquisition
before Docker/runtime setup, not Go integration output. The partial tail has
no timestamps or terminal error; it does not establish the mirror's underlying
DNS/TCP/server failure or the exact total time spent in APT.

The baseline and patched workflow use the same APT setup commands. In source
qualification CI38258121064, Azure indexes downloaded normally and all ten
jobs passed. That success does not qualify a new workflow change.

Upstream reports the same mirror retry pattern in
[runner-images issue14594](https://github.com/actions/runner-images/issues/14594).
Its [APT configuration](https://github.com/actions/runner-images/blob/main/images/ubuntu/scripts/build/configure-apt.sh)
also bounds retries and HTTP/HTTPS timeouts. This supports a CI acquisition
repair; it is not evidence about the earlier SumDB error.

The repair changes only Linux CI package acquisition:

- Use the official Ubuntu HTTPS archive in the existing hosted-runner mirror
  list, preserving Ubuntu source suites, components, keyring and signature checks.
- Bound HTTP/HTTPS inactivity to15 seconds, retry once, and bound update/install
  commands to10/15 minutes with a30-second forced-termination grace period.
- Fail on any index-update error and on install failure. No warning-only
  continuation, signature bypass, package-version relaxation or test skip.
- Keep Docker/containerd package pins, runtime lock, toolchain identity and
  all integration commands unchanged.

Local canonical workflow, docs and security checks pass under Go1.26.9.
The extracted complete lifecycle shell also passes `bash -n`. A separate
Ubuntu24.04 APT state under `/tmp` used the same command-line acquisition
options and the existing Ubuntu archive keyring. Its unavailable loopback
mirror failed with exit100 in1.06s; the official mirror authenticated all four
Noble suites and acquired main/amd64 indexes with exit0 in5.61s. This did not
install packages or modify host APT configuration. It is a narrow index and
signature check, not the hosted runner's complete package installation.

Original local logs and command/hash receipts are in
`/tmp/haa-security-go1269-validation/apt-followup-host/`. An earlier sandbox
attempt could not create sockets/resolve the host and is preserved separately
in `apt-followup/`; it does not reproduce the remote mirror stall.

The completed CI306 raw log confirms88m32.37s without further APT output before
concurrency cancellation; no integration test had started. Its exact transport
or internal APT cause remains unknown. CI307 at2eea071 completes package-index
acquisition (12.4MB/3s), package download (126MB/4s) and runtime installation.
The APT repair therefore passes that actual hosted acquisition. Linux/Required
subsequently fail on a distinct Cargo configuration read, tracked in the
[Cargo follow-up](./cargo-gai-followup.md). Prior host security and SumDB
qualification remain historical evidence for their exact source.
