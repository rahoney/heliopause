package pypi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// ReadConsoleExecutionInventory measures an owned snapshot from controlled
// intake. It returns console declaration DATA, never current inspection/Policy
// approval or permission to import a module or execute its callable.
// This initial public-wheel lane uses the existing default wheel bounds.
func (i *StaticInspector) ReadConsoleExecutionInventory(ctx context.Context, artifact domain.AcquiredArtifact) (inventory artifactpypi.ConsoleExecutionInventory, err error) {
	if ctx == nil {
		return inventory, errors.New("console inventory requires a context")
	}
	defer func() {
		err = errors.Join(err, ctx.Err(), context.Cause(ctx))
		if err != nil {
			inventory = artifactpypi.ConsoleExecutionInventory{}
		}
	}()
	limits := artifactpypi.DefaultWheelLimits()
	if ctx.Err() != nil || i == nil || artifact.Identity().Source() != artifactpypi.PublicPyPIProfile().Source() || artifact.Identity().Variant() != "wheel" || artifact.SizeBytes() == 0 || artifact.SizeBytes() > uint64(limits.MaxCompressed) {
		return inventory, errors.New("console inventory requires a bounded acquired public wheel")
	}
	directory, filenameRecord, archiveName, err := i.artifactLocation(artifact)
	if err != nil {
		return inventory, err
	}
	before, err := os.Lstat(i.intakeRoot)
	if err != nil || !before.IsDir() {
		return inventory, errors.New("console intake root is not a real directory")
	}
	root, err := os.OpenRoot(i.intakeRoot)
	if err != nil {
		return inventory, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(before, opened) {
		return inventory, errors.New("console intake root changed while opening")
	}
	relative, err := filepath.Rel(i.intakeRoot, directory)
	if err != nil {
		return inventory, err
	}
	runInfo, err := root.Lstat(relative)
	if err != nil || !runInfo.IsDir() {
		return inventory, errors.New("console intake Run is not a real directory")
	}
	run, err := root.OpenRoot(relative)
	if err != nil {
		return inventory, err
	}
	defer func() { err = errors.Join(err, run.Close()) }()
	opened, err = run.Stat(".")
	if err != nil || !os.SameFile(runInfo, opened) {
		return inventory, errors.New("console intake Run changed while opening")
	}
	filenameBytes, err := readConsoleIntakeSnapshot(ctx, run, filenameRecord, 512, 0)
	if err != nil {
		return inventory, err
	}
	filename := string(filenameBytes)
	variant, ok := pypiVariant(filename)
	if !ok || variant != "wheel" || strings.ContainsAny(filename, `/\\`) {
		return inventory, errors.New("console intake filename is invalid")
	}
	body, err := readConsoleIntakeSnapshot(ctx, run, archiveName, limits.MaxCompressed, artifact.SizeBytes())
	if err != nil {
		return inventory, err
	}
	// Every canonical parser pass observes the same owned bytes, even if intake
	// is later modified. The parser measures the complete original archive.
	inventory, err = artifactpypi.InspectConsoleExecutionWheel(artifact, bytes.NewReader(body), filename, i.target, limits)
	if err != nil {
		return inventory, err
	}
	current, err := root.Lstat(relative)
	if err != nil || !os.SameFile(runInfo, current) || !current.IsDir() {
		return inventory, errors.New("console intake Run identity changed during measurement")
	}
	return inventory, nil
}

func readConsoleIntakeSnapshot(ctx context.Context, root *os.Root, name string, limit int64, expected uint64) (body []byte, err error) {
	before, err := root.Lstat(name)
	if err != nil || !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > limit || (expected != 0 && uint64(before.Size()) != expected) {
		return nil, errors.New("console intake member type or size is invalid")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, file.Close())
		if err != nil {
			body = nil
		}
	}()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) || !opened.Mode().IsRegular() || opened.Size() != before.Size() {
		return nil, errors.New("console intake member changed while opening")
	}
	body, err = io.ReadAll(io.LimitReader(consoleContextReader{ctx, file}, limit+1))
	if err != nil {
		return nil, err
	}
	after, err := root.Lstat(name)
	if err != nil || !os.SameFile(before, after) || !after.Mode().IsRegular() || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) || int64(len(body)) != before.Size() {
		return nil, errors.New("console intake member changed during snapshot")
	}
	return body, nil
}

type consoleContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r consoleContextReader) Read(body []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, errors.Join(err, context.Cause(r.ctx))
	}
	n, err := r.reader.Read(body)
	if r.ctx.Err() != nil {
		return n, errors.Join(err, r.ctx.Err(), context.Cause(r.ctx))
	}
	return n, err
}
