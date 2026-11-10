package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/rahoney/heliopause/internal/hosttool"
)

//go:embed semgrep_build_worker.sh
var semgrepBuildWorker []byte

const semgrepSDKPrepare = `#!/bin/sh
set -eu
apk add --cache=no --network=no --repositories-file=/dev/null /input/apks/*.apk
addgroup -g 1000 builder
adduser -D -u 1000 -G builder -h /home/builder builder
mkdir -p /work
chown builder:builder /work
apk query --installed --format json '*' > /builder-package-metadata.json
opam --version > /builder-opam-version.txt
`

type semgrepBuildSDK struct {
	BaseImage    string `json:"base_image"`
	Architecture string `json:"architecture"`
	OpamVersion  string `json:"opam_version"`
	Note         string `json:"note"`
	Packages     []struct {
		Name   string `json:"name"`
		Bytes  int64  `json:"bytes"`
		SHA256 string `json:"sha256"`
	} `json:"packages"`
}

// No ReaderFrom is exposed: both process streams must cross this limit.
type semgrepBuildOutput struct {
	output    io.Writer
	remaining int64
}

func (w *semgrepBuildOutput) Write(body []byte) (int, error) {
	if int64(len(body)) > w.remaining {
		return 0, errors.New("semgrep build output exceeds its stream bound")
	}
	n, err := w.output.Write(body)
	w.remaining -= int64(n)
	return n, err
}

func readSemgrepBuildSDK(root *os.Root, selected semgrepSourcePackageInput) (semgrepBuildSDK, error) {
	var sdk semgrepBuildSDK
	for _, member := range selected.Files {
		if member.Path != "build/sdk-inputs.json" {
			continue
		}
		if member.Size <= 0 || member.Size > 128<<10 {
			return sdk, errors.New("semgrep SDK descriptor exceeds bound")
		}
		var data bytes.Buffer
		if err := copySemgrepSourceMember(root, &data, member); err != nil {
			return sdk, err
		}
		if _, err := jsonSpans(data.Bytes()); err != nil {
			return sdk, err
		}
		decoder := json.NewDecoder(bytes.NewReader(data.Bytes()))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&sdk) != nil || decoder.Decode(new(any)) != io.EOF {
			return sdk, errors.New("semgrep SDK descriptor is malformed")
		}
	}
	const basePrefix = "docker.io/library/alpine@sha256:"
	if !strings.HasPrefix(sdk.BaseImage, basePrefix) || !digestPattern.MatchString(strings.TrimPrefix(sdk.BaseImage, basePrefix)) || sdk.Architecture != "amd64" || !versionPattern.MatchString(sdk.OpamVersion) || len(sdk.Packages) == 0 || len(sdk.Packages) > 220 {
		return sdk, errors.New("semgrep SDK base or package identities are unavailable")
	}
	seen := map[string]bool{}
	var total int64
	for _, file := range sdk.Packages {
		if len(file.Name) < 5 || len(file.Name) > 200 || !strings.HasSuffix(file.Name, ".apk") || seen[file.Name] || file.Bytes <= 0 || file.Bytes > 128<<20 || !digestPattern.MatchString(file.SHA256) {
			return sdk, errors.New("semgrep SDK package identity is invalid")
		}
		for i, c := range file.Name {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || i > 0 && strings.ContainsRune("+_.-", c)) {
				return sdk, errors.New("semgrep SDK package name is unsafe")
			}
		}
		seen[file.Name] = true
		total += file.Bytes
		if total > 512<<20 {
			return sdk, errors.New("semgrep SDK package set exceeds bound")
		}
	}
	return sdk, nil
}

// rebuildSemgrep is an explicit offline developer build. It reuses the strict
// Host executor; a successful source build never grants an inspection role.
func rebuildSemgrep(args []string, out io.Writer) (result error) {
	flags := flag.NewFlagSet("update-runtime rebuild-semgrep", flag.ContinueOnError)
	flags.SetOutput(out)
	input := flags.String("input", "", "absolute frozen source collection")
	apks := flags.String("sdk-apks", "", "absolute frozen SDK APK directory")
	state := flags.String("state", "", "absolute new external source-build state directory")
	output := flags.String("output", "", "absolute new external semgrep-core-linux-amd64")
	expected := flags.String("expected-sha256", "", "optional fixed reference native digest; mismatch fails publication")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || filepath.Base(*output) != "semgrep-core-linux-amd64" || *expected != "" && !digestPattern.MatchString(*expected) {
		return errors.New("semgrep source build requires Linux amd64 and explicit valid build paths/reference")
	}
	for _, location := range []string{*input, *apks, *state, *output} {
		if !filepath.IsAbs(location) || filepath.Clean(location) != location || strings.ContainsAny(location, ",\r\n") {
			return errors.New("semgrep source build paths must be absolute and canonical")
		}
	}
	for _, directory := range []string{*input, *apks, filepath.Dir(*state), filepath.Dir(*output)} {
		canonical, err := filepath.EvalSymlinks(directory)
		if err != nil || canonical != directory {
			return errors.New("semgrep source build directory is unavailable or a symlink")
		}
	}
	for _, source := range []string{*input, *apks} {
		for _, target := range []string{*state, *output} {
			relative, err := filepath.Rel(source, target)
			if err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return errors.New("semgrep source build writes must be outside frozen inputs")
			}
		}
	}
	if _, err := os.Lstat(*output); !errors.Is(err, os.ErrNotExist) {
		return errors.New("semgrep native output exists or cannot be checked")
	}
	body, err := readFrozenSupplyFile(*input, "source-inputs.json", 2<<20)
	if err != nil {
		return err
	}
	selected, _, err := parseSemgrepPackageManifest(body, false)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(*input)
	if err != nil {
		return err
	}
	defer root.Close()
	sdk, err := readSemgrepBuildSDK(root, selected)
	if err != nil {
		return err
	}
	ctx, cancelSignal := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelSignal()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Hour)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		return err
	}
	if err := os.Mkdir(*state, 0700); err != nil {
		return errors.Join(errors.New("semgrep source-build state must be new"), executor.Close())
	}
	stateInfo, err := os.Lstat(*state)
	if err != nil || !nativeOutputDirectoryOwned(stateInfo) {
		return errors.Join(errors.New("semgrep source-build state is not protected"), executor.Close())
	}
	var containers []struct{ id, image string }
	nonce := rand.Text()
	const label = "io.heliopause.semgrep-source-build"
	record := map[string]any{"schema": "helox.semgrep-source-build/v1", "source_commit": selected.SourceCommit, "source_manifest_sha256": hash(body), "sdk_base": sdk.BaseImage, "worker_sha256": hash(semgrepBuildWorker), "sdk_prepare_sha256": hash([]byte(semgrepSDKPrepare)), "runtime_qualified": false, "complete": false}
	run := func(commandCtx context.Context, arguments ...string) ([]byte, error) {
		var data, diagnostic bytes.Buffer
		err := executor.RunOutputStreams(commandCtx, &semgrepBuildOutput{&data, 2 << 20}, &semgrepBuildOutput{&diagnostic, 2 << 20}, "docker", arguments...)
		if err != nil {
			return data.Bytes(), fmt.Errorf("semgrep source Docker %s: %w: %s", arguments[0], err, diagnostic.String())
		}
		return data.Bytes(), nil
	}
	// Every use and cleanup is tied to the ID, immutable image and invocation label.
	inspect := func(commandCtx context.Context, id, image string) (bool, int, error) {
		data, err := run(commandCtx, "inspect", "--", id)
		if err != nil {
			return false, 0, err
		}
		var info []struct {
			ID, Image string
			State     struct {
				Running  bool
				ExitCode int
				Status   string
			}
			Config struct{ Labels map[string]string }
		}
		if json.Unmarshal(data, &info) != nil || len(info) != 1 || info[0].ID != id || info[0].Image != image || info[0].Config.Labels[label] != nonce {
			return false, 0, errors.New("semgrep source owned container identity changed")
		}
		if info[0].State.Status != "created" && info[0].State.Status != "running" && info[0].State.Status != "exited" {
			return false, 0, errors.New("semgrep source container state is unknown")
		}
		return info[0].State.Running, info[0].State.ExitCode, nil
	}
	var sdkImage string
	defer func() {
		cleanupCtx, stopCleanup := context.WithTimeout(context.Background(), 2*time.Minute)
		defer stopCleanup()
		var cleanupErrors []string
		for i := len(containers) - 1; i >= 0; i-- {
			owned := containers[i]
			running, _, err := inspect(cleanupCtx, owned.id, owned.image)
			if err == nil && running {
				_, err = run(cleanupCtx, "stop", "--time", "10", "--", owned.id)
				if err == nil {
					_, err = run(cleanupCtx, "wait", "--", owned.id)
				}
				if err == nil {
					running, _, err = inspect(cleanupCtx, owned.id, owned.image)
					if running {
						err = errors.New("semgrep source container did not stop")
					}
				}
			}
			if err == nil {
				_, err = run(cleanupCtx, "rm", "--", owned.id)
			}
			if err != nil {
				cleanupErrors = append(cleanupErrors, err.Error())
			}
		}
		if sdkImage != "" && len(cleanupErrors) == 0 {
			data, err := run(cleanupCtx, "image", "inspect", "--", sdkImage)
			var images []struct {
				ID     string
				Config struct{ Labels map[string]string }
			}
			if err == nil && (json.Unmarshal(data, &images) != nil || len(images) != 1 || images[0].ID != sdkImage || images[0].Config.Labels[label] != nonce) {
				err = errors.New("semgrep source SDK image identity changed")
			}
			if err == nil {
				_, err = run(cleanupCtx, "image", "rm", "--", sdkImage)
			}
			if err != nil {
				cleanupErrors = append(cleanupErrors, err.Error())
			}
		}
		// Keep the source offer and bounded diagnostics; discard successful transient copies.
		if result == nil && len(cleanupErrors) == 0 {
			current, err := os.Lstat(*state)
			if err == nil && !os.SameFile(stateInfo, current) {
				err = errors.New("semgrep source state identity changed")
			}
			if err == nil {
				for _, pkg := range sdk.Packages {
					if err = os.Remove(filepath.Join(*state, "sdk-input/apks", pkg.Name)); err != nil {
						break
					}
				}
			}
			if err == nil {
				for _, name := range []string{"sdk-input/apks", "sdk-input/prepare.sh", "sdk-input", "native.candidate"} {
					if err = os.Remove(filepath.Join(*state, name)); err != nil {
						break
					}
				}
			}
			if err != nil {
				cleanupErrors = append(cleanupErrors, err.Error())
			}
		}
		if err := executor.Close(); err != nil {
			cleanupErrors = append(cleanupErrors, err.Error())
		}
		if len(cleanupErrors) > 0 {
			result = errors.Join(result, errors.New(strings.Join(cleanupErrors, "; ")))
		}
		record["cleanup_errors"] = cleanupErrors
		if result != nil {
			record["first_error"] = result.Error()
		} else {
			record["complete"] = true
		}
		data, err := json.MarshalIndent(record, "", "  ")
		if err == nil {
			err = os.WriteFile(filepath.Join(*state, "result.json"), append(data, '\n'), 0600)
		}
		result = errors.Join(result, err)
		if result == nil {
			result = json.NewEncoder(out).Encode(record)
		}
	}()

	for _, directory := range []string{"source", "sdk-input", "sdk-input/apks"} {
		if err := os.Mkdir(filepath.Join(*state, directory), 0700); err != nil {
			return err
		}
	}
	if err := os.Chmod(filepath.Join(*state, "source"), 0755); err != nil {
		return err
	}
	var sourceReceipt bytes.Buffer
	if err := packageSemgrepSource([]string{"--input", *input, "--output", filepath.Join(*state, "source/semgrep-SOURCES.tar")}, &sourceReceipt); err != nil {
		return err
	}
	var captured struct {
		ManifestSHA256 string `json:"manifest_sha256"`
	}
	if json.Unmarshal(sourceReceipt.Bytes(), &captured) != nil || captured.ManifestSHA256 != hash(body) {
		return errors.New("semgrep source manifest changed during capture")
	}
	for _, required := range []struct{ name, digest string }{{"build/rebuild-worker.sh", hash(semgrepBuildWorker)}, {"build/sdk-prepare.sh", hash([]byte(semgrepSDKPrepare))}} {
		found := false
		for _, member := range selected.Files {
			if member.Path == required.name && member.SHA256 == required.digest {
				found = true
			}
		}
		if !found {
			return errors.New("semgrep source offer omits the exact canonical build recipe")
		}
	}
	apkRoot, err := os.OpenRoot(*apks)
	if err != nil {
		return err
	}
	defer apkRoot.Close()
	for _, input := range sdk.Packages {
		file, err := os.OpenFile(filepath.Join(*state, "sdk-input/apks", input.Name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0400)
		if err != nil {
			return err
		}
		err = copySemgrepSourceMember(apkRoot, file, semgrepSourcePackageFile{input.Name, input.SHA256, input.Bytes, 0444})
		if closeErr := file.Close(); err != nil || closeErr != nil {
			return errors.Join(err, closeErr)
		}
	}
	if err := os.WriteFile(filepath.Join(*state, "sdk-input/prepare.sh"), []byte(semgrepSDKPrepare), 0444); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*state, "worker.sh"), semgrepBuildWorker, 0444); err != nil {
		return err
	}
	base, err := run(ctx, "image", "inspect", "--format", "{{.Id}}", sdk.BaseImage)
	if err != nil {
		return fmt.Errorf("verify preacquired exact SDK base: %w", err)
	}
	baseID := strings.TrimSpace(string(base))
	if !strings.HasPrefix(baseID, "sha256:") || !digestPattern.MatchString(strings.TrimPrefix(baseID, "sha256:")) {
		return errors.New("semgrep SDK base image identity is malformed")
	}
	sdkContainer, err := run(ctx, "create", "--pull=never", "--runtime=runc", "--network=none", "--label", label+"="+nonce, "--pids-limit=128", "--memory=1g", "--memory-swap=1g", "--cpus=2", "--log-driver=none", "--mount", "type=bind,source="+filepath.Join(*state, "sdk-input")+",target=/input,readonly", "--user=0:0", "--entrypoint=/bin/sh", sdk.BaseImage, "/input/prepare.sh")
	if err != nil {
		return err
	}
	sdkID := strings.TrimSpace(string(sdkContainer))
	if !digestPattern.MatchString(sdkID) {
		return errors.New("semgrep SDK container identity is malformed")
	}
	containers = append(containers, struct{ id, image string }{sdkID, baseID})
	record["sdk_container_id"] = sdkID
	if running, _, err := inspect(ctx, sdkID, baseID); err != nil || running {
		return errors.Join(errors.New("semgrep SDK is not in its owned initial state"), err)
	}
	prepareLog, err := os.OpenFile(filepath.Join(*state, "sdk.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	err = executor.RunOutputStreams(ctx, &semgrepBuildOutput{prepareLog, 8 << 20}, &semgrepBuildOutput{prepareLog, 8 << 20}, "docker", "start", "--attach", "--", sdkID)
	if closeErr := prepareLog.Close(); err != nil || closeErr != nil {
		return errors.Join(err, closeErr, ctx.Err())
	}
	if running, code, err := inspect(ctx, sdkID, baseID); err != nil || running || code != 0 {
		return errors.Join(errors.New("semgrep SDK preparation did not complete successfully"), err)
	}
	image, err := run(ctx, "commit", "--change", "USER 1000:1000", "--change", "WORKDIR /work", sdkID)
	if err != nil {
		return err
	}
	sdkImage = strings.TrimSpace(string(image))
	if !strings.HasPrefix(sdkImage, "sha256:") || !digestPattern.MatchString(strings.TrimPrefix(sdkImage, "sha256:")) {
		sdkImage = ""
		return errors.New("semgrep committed SDK identity is malformed")
	}
	record["sdk_image_id"] = sdkImage
	worker, err := run(ctx, "create", "--pull=never", "--runtime=runc", "--network=none", "--read-only", "--user=1000:1000", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--label", label+"="+nonce, "--pids-limit=512", "--memory=16g", "--memory-swap=16g", "--cpus=2", "--ulimit", "fsize=1073741824:1073741824", "--log-driver=none", "--env", "HOME=/home/builder", "--env", "LANG=C.UTF-8", "--env", fmt.Sprintf("SOURCE_DATE_EPOCH=%d", selected.SourceDateEpoch), "--tmpfs", "/work:rw,exec,nosuid,nodev,size=12884901888,uid=1000,gid=1000,mode=0700", "--tmpfs", "/tmp:rw,nosuid,nodev,size=536870912,uid=1000,gid=1000,mode=0700", "--mount", "type=bind,source="+filepath.Join(*state, "source")+",target=/source,readonly", "--mount", "type=bind,source="+filepath.Join(*state, "worker.sh")+",target=/worker.sh,readonly", "--entrypoint=/bin/bash", sdkImage, "/worker.sh")
	if err != nil {
		return err
	}
	workerID := strings.TrimSpace(string(worker))
	if !digestPattern.MatchString(workerID) {
		return errors.New("semgrep source worker identity is malformed")
	}
	containers = append(containers, struct{ id, image string }{workerID, sdkImage})
	record["worker_container_id"] = workerID
	if running, _, err := inspect(ctx, workerID, sdkImage); err != nil || running {
		return errors.Join(errors.New("semgrep worker is not in its owned initial state"), err)
	}
	native, err := os.OpenFile(filepath.Join(*state, "native.candidate"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer native.Close()
	log, err := os.OpenFile(filepath.Join(*state, "build.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	digest := sha256.New()
	err = executor.RunOutputStreams(ctx, &semgrepBuildOutput{io.MultiWriter(native, digest), 256 << 20}, &semgrepBuildOutput{log, 16 << 20}, "docker", "start", "--attach", "--", workerID)
	if closeErr := log.Close(); err != nil || closeErr != nil {
		return errors.Join(err, closeErr, ctx.Err())
	}
	if running, code, err := inspect(ctx, workerID, sdkImage); err != nil || running || code != 0 {
		return errors.Join(errors.New("semgrep source compilation did not complete successfully"), err)
	}
	if err := native.Sync(); err != nil {
		return err
	}
	info, err := native.Stat()
	if err != nil || info.Size() <= 0 || info.Size() > 256<<20 {
		return errors.New("semgrep source build produced no bounded native output")
	}
	checksum := hex.EncodeToString(digest.Sum(nil))
	record["native_sha256"], record["native_bytes"] = checksum, info.Size()
	if *expected != "" && checksum != *expected {
		return errors.New("semgrep clean source build differs from the explicit reference; candidate retained")
	}
	if err := validateSemgrepNative(native); err != nil {
		return err
	}
	if _, err := native.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := publishNativeAssetWith(*output, info.Size(), 0555, func(output io.Writer) error {
		copied := sha256.New()
		count, err := io.Copy(output, io.TeeReader(native, copied))
		if err != nil || count != info.Size() || hex.EncodeToString(copied.Sum(nil)) != checksum {
			return errors.Join(errors.New("semgrep source candidate changed before publication"), err)
		}
		return nil
	}); err != nil {
		return err
	}
	record["published"] = true
	return nil
}
