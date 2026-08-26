// Package cli owns Heliopause command parsing and presentation.
package cli

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

// New returns a new Heliopause root command.
func New(stdout, stderr io.Writer) (*cobra.Command, error) {
	if stdout == nil {
		return nil, errors.New("cli: stdout writer is nil")
	}
	if stderr == nil {
		return nil, errors.New("cli: stderr writer is nil")
	}

	command := &cobra.Command{
		Use:           "helox",
		Short:         "Inspect software artifacts before trusted promotion",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	command.RunE = func(command *cobra.Command, _ []string) error {
		if err := command.Context().Err(); err != nil {
			return err
		}

		return command.Help()
	}
	command.SetOut(stdout)
	command.SetErr(stderr)
	initializeCommandTree(command)

	return command, nil
}

type ReferenceParser func(string) (domain.ArtifactReference, error)

// Doctor reports bounded installation and Host readiness checks. A false
// Healthy result is never rendered as a successful diagnosis.
type Doctor interface {
	Diagnose(context.Context) DoctorReport
}

type DoctorReport struct {
	Healthy bool
	Checks  []DoctorCheck
}

type DoctorCheck struct {
	Name    string
	Healthy bool
	Detail  string
}

// AddDoctor connects the composition-root health boundary to the static root
// command. It performs no work while help is rendered.
func AddDoctor(root *cobra.Command, doctor Doctor) error {
	if root == nil || doctor == nil {
		return errors.New("doctor command requires health service")
	}
	command := findRootCommand(root, "doctor")
	if command == nil {
		return errors.New("doctor command is not registered")
	}
	command.RunE = func(command *cobra.Command, _ []string) error {
		report := doctor.Diagnose(contextOrBackground(command.Context()))
		for _, check := range report.Checks {
			status := "UNAVAILABLE"
			if check.Healthy {
				status = "OK"
			}
			if _, err := fmt.Fprintf(command.OutOrStdout(), "%s=%s (%s)\n", check.Name, status, check.Detail); err != nil {
				return err
			}
		}
		if !report.Healthy {
			return ExitError{Code: 1}
		}
		return nil
	}
	return nil
}

func AddGitHubReleaseInspect(root *cobra.Command, parser ReferenceParser, inspector Inspector) error {
	if root == nil || parser == nil || inspector == nil {
		return errors.New("github inspect command requires root and use case")
	}
	if command := findLeaf(root, "github", "inspect"); command != nil {
		command.RunE = func(command *cobra.Command, args []string) error {
			reference, err := parser(args[0])
			if err != nil {
				return err
			}
			request, err := application.NewInspectRequest(reference)
			if err != nil {
				return err
			}
			code, err := ExecuteInspect(contextOrBackground(command.Context()), inspector, request, true, command.OutOrStdout())
			if err != nil {
				return err
			}
			if code != 0 {
				return ExitError{Code: code}
			}
			return nil
		}
		return nil
	}
	return errors.New("github inspect command is not registered")
}

func AddGitHubReleaseInstall(root *cobra.Command, parser ReferenceParser, installer Installer) error {
	if root == nil || parser == nil || installer == nil {
		return errors.New("github install command requires root and use case")
	}
	if install := findLeaf(root, "github", "install"); install != nil {
		install.RunE = func(command *cobra.Command, args []string) error {
			reference, err := parser(args[0])
			if err != nil {
				return err
			}
			target, err := command.Flags().GetString("target")
			if err != nil {
				return err
			}
			targetValue, err := githubInstallTarget(target, reference)
			if err != nil {
				return err
			}
			installContext, err := domain.NewInstallContext(targetValue)
			if err != nil {
				return err
			}
			request, err := application.NewInstallRequest(reference, installContext)
			if err != nil {
				return err
			}
			code, err := ExecuteInstall(contextOrBackground(command.Context()), installer, request, true, command.OutOrStdout())
			if err != nil {
				return err
			}
			if code != 0 {
				return ExitError{Code: code}
			}
			return nil
		}
		return nil
	}
	return errors.New("github install command is not registered")
}

// AddPyPIInstall adds the injected PyPI Install/Promotion use case. The
// generic JSON/human result contract is intentionally shared with npm.
func AddPyPIInstall(root *cobra.Command, installer Installer) error {
	return AddPyPIInstallSources(root, map[string]Installer{"pypi": installer})
}

// AddPyPIInstallSources wires one bounded pip install command to the
// canonical PyPI and named PyTorch source profiles.
func AddPyPIInstallSources(root *cobra.Command, installers map[string]Installer) error {
	if root == nil || len(installers) == 0 {
		return errors.New("pypi install command requires root and use case")
	}
	if installCommand := findLeaf(root, "pip", "install"); installCommand != nil {
		installCommand.RunE = func(command *cobra.Command, args []string) error {
			source, err := command.Flags().GetString("source")
			if err != nil {
				return err
			}
			if source == "" {
				source = "pypi"
			}
			sourceID := artifactpypi.PublicPyPIProfile().Source()
			if source != "pypi" {
				profileName := strings.TrimPrefix(source, "pytorch:")
				if profileName == source {
					return errors.New("pip source must be pypi or a named pytorch profile")
				}
				profile, ok := artifactpypi.PyTorchProfile(profileName)
				if !ok {
					return errors.New("unknown PyTorch source profile")
				}
				sourceID = profile.Source()
			}
			installer, ok := installers[sourceID.String()]
			if !ok || installer == nil {
				return errors.New("selected Python source is unavailable on this Host")
			}
			reference, err := artifactpypi.ParseReferenceForSource(args[0], sourceID)
			if err != nil {
				return err
			}
			target, err := command.Flags().GetString("target")
			if err != nil {
				return err
			}
			installTarget, err := activePythonVenv(target)
			if err != nil {
				return err
			}
			installContext, err := domain.NewPythonVenvInstallContext(installTarget)
			if err != nil {
				return err
			}
			request, err := application.NewInstallRequest(reference, installContext)
			if err != nil {
				return err
			}
			exitCode, operationErr := ExecuteInstall(contextOrBackground(command.Context()), installer, request, true, command.OutOrStdout())
			if operationErr != nil {
				return operationErr
			}
			if exitCode != 0 {
				return ExitError{Code: exitCode}
			}
			return nil
		}
		return nil
	}
	return errors.New("pip install command is not registered")
}

func activePythonVenv(target string) (domain.InstallTarget, error) {
	if target == "" {
		target = os.Getenv("VIRTUAL_ENV")
	}
	if target == "" {
		return domain.InstallTarget{}, errors.New("pip install requires an active virtual environment")
	}
	value, err := domain.NewInstallTarget(target)
	if err != nil {
		return domain.InstallTarget{}, errors.New("active virtual environment is unsupported")
	}
	return value, nil
}

func npmInstallContext(target string) (domain.InstallContext, error) {
	if target != "" {
		value, err := domain.NewInstallTarget(target)
		if err != nil {
			return domain.InstallContext{}, err
		}
		return domain.NewInstallContext(value)
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return domain.InstallContext{}, errors.New("resolve npm project directory")
	}
	project, err := domain.NewInstallTarget(workingDirectory)
	if err != nil {
		return domain.InstallContext{}, errors.New("npm project directory is unsupported")
	}
	return domain.NewNPMProjectInstallContext(project)
}

func githubInstallTarget(target string, reference domain.ArtifactReference) (domain.InstallTarget, error) {
	if target != "" {
		return domain.NewInstallTarget(target)
	}
	owner, repo, tag, asset, err := splitGitHubLocator(reference)
	if err != nil {
		return domain.InstallTarget{}, errors.New("GitHub Release default target cannot be derived")
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return domain.InstallTarget{}, errors.New("resolve GitHub Release default target directory")
	}
	// Keep the default destination directly below the already-existing working
	// directory. The selector digest prevents collisions after safe segment
	// normalization while the directory remains recognizable to the user.
	selectorDigest := sha256.Sum256([]byte(reference.Locator()))
	name := fmt.Sprintf(".helox-github-%s-%s-%s-%s-%x",
		safeTargetSegment(owner),
		safeTargetSegment(repo),
		safeTargetSegment(tag),
		safeTargetSegment(asset),
		selectorDigest[:6],
	)
	return domain.NewInstallTarget(filepath.Join(workingDirectory, name))
}

func splitGitHubLocator(reference domain.ArtifactReference) (owner, repo, tag, asset string, err error) {
	if reference.Source().String() != "github-release" {
		return "", "", "", "", errors.New("unsupported GitHub Release source")
	}
	locator := reference.Locator()
	at := strings.IndexByte(locator, '@')
	hash := strings.LastIndexByte(locator, '#')
	if at <= 0 || hash <= at+1 || hash == len(locator)-1 || strings.Count(locator, "@") != 1 || strings.Count(locator, "#") != 1 {
		return "", "", "", "", errors.New("GitHub Release locator is invalid")
	}
	repository := locator[:at]
	if strings.Count(repository, "/") != 1 {
		return "", "", "", "", errors.New("GitHub Release repository is invalid")
	}
	parts := strings.SplitN(repository, "/", 2)
	return parts[0], parts[1], locator[at+1 : hash], locator[hash+1:], nil
}

func safeTargetSegment(value string) string {
	var builder strings.Builder
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '.' || character == '_' || character == '-' {
			builder.WriteRune(character)
			continue
		}
		builder.WriteByte('_')
	}
	if builder.Len() == 0 {
		return "unknown"
	}
	return builder.String()
}

// AddPyPIInspect adds the isolated PyPI primary-distribution inspect path.
func AddPyPIInspect(root *cobra.Command, inspector Inspector) error {
	if root == nil || inspector == nil {
		return errors.New("pypi inspect command requires root and use case")
	}
	if inspectCommand := findLeaf(root, "pypi", "inspect"); inspectCommand != nil {
		inspectCommand.RunE = func(command *cobra.Command, args []string) error {
			reference, err := artifactpypi.ParseReference(args[0])
			if err != nil {
				return err
			}
			request, err := application.NewInspectRequest(reference)
			if err != nil {
				return err
			}
			exitCode, operationErr := ExecuteInspect(contextOrBackground(command.Context()), inspector, request, true, command.OutOrStdout())
			if operationErr != nil {
				return operationErr
			}
			if exitCode != 0 {
				return ExitError{Code: exitCode}
			}
			return nil
		}
		return nil
	}
	return errors.New("pypi inspect command is not registered")
}

// AddNPMInspect adds the injected npm Inspect use case to a root command.
func AddNPMInspect(root *cobra.Command, inspector Inspector) error {
	if root == nil || inspector == nil {
		return errors.New("npm inspect command requires root and use case")
	}
	if inspectCommand := findLeaf(root, "npm", "inspect"); inspectCommand != nil {
		inspectCommand.RunE = func(command *cobra.Command, args []string) error {
			reference, err := artifactnpm.ParseReference(args[0])
			if err != nil {
				return err
			}
			request, err := application.NewInspectRequest(reference)
			if err != nil {
				return err
			}
			exitCode, operationErr := ExecuteInspect(contextOrBackground(command.Context()), inspector, request, true, command.OutOrStdout())
			if operationErr != nil {
				return operationErr
			}
			if exitCode != 0 {
				return ExitError{Code: exitCode}
			}
			return nil
		}
		return nil
	}
	return errors.New("npm inspect command is not registered")
}

// AddNPMInstall adds the injected npm Install/Promotion use case.
func AddNPMInstall(root *cobra.Command, installer Installer) error {
	if root == nil || installer == nil {
		return errors.New("npm install command requires root and use case")
	}
	if installCommand := findLeaf(root, "npm", "install"); installCommand != nil {
		installCommand.RunE = func(command *cobra.Command, args []string) error {
			reference, err := artifactnpm.ParseReference(args[0])
			if err != nil {
				return err
			}
			target, err := command.Flags().GetString("target")
			if err != nil {
				return err
			}
			installContext, err := npmInstallContext(target)
			if err != nil {
				return err
			}
			request, err := application.NewInstallRequest(reference, installContext)
			if err != nil {
				return err
			}
			exitCode, operationErr := ExecuteInstall(contextOrBackground(command.Context()), installer, request, true, command.OutOrStdout())
			if operationErr != nil {
				return operationErr
			}
			if exitCode != 0 {
				return ExitError{Code: exitCode}
			}
			return nil
		}
		return nil
	}
	return errors.New("npm install command is not registered")
}

func initializeCommandTree(root *cobra.Command) {
	root.AddCommand(&cobra.Command{Use: "doctor", Short: "Check installed release and trusted Host runtime readiness", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error { return errors.New("doctor command is not configured") }})
	npm := ensureNPMCommand(root)
	npm.AddCommand(newStaticLeaf("inspect <package-reference>", "Inspect an npm package", false))
	npm.AddCommand(newStaticLeaf("install <package-reference>", "Install an npm package into the current project", true))
	pypi := ensurePyPICommand(root)
	pypi.AddCommand(newStaticLeaf("inspect <project>[@<version>]", "Inspect a PyPI distribution", false))
	pip := ensurePipCommand(root)
	pipInstall := newStaticLeaf("install <project>[@<version>]", "Install a PyPI distribution into the active virtual environment", true)
	pipInstall.Flags().String("source", "pypi", pythonSourceHelp())
	pip.AddCommand(pipInstall)
	github := ensureGitHubCommand(root)
	github.AddCommand(newStaticLeaf("inspect <owner>/<repo>@<tag>#<asset>", "Inspect a GitHub Release asset", false))
	githubInstall := newStaticLeaf("install <owner>/<repo>@<tag>#<asset>", "Install a GitHub Release asset", true)
	github.AddCommand(githubInstall)
	goCommand := ensureGoCommand(root)
	goCommand.AddCommand(newStaticLeaf("get <module>@<version>", "Resolve and transactionally add a public Go Module", false))
	goCommand.AddCommand(newStaticLeaf("build <package>", "Build a Go project from the HAA-verified module cache", false))
	goMod := &cobra.Command{Use: "mod", Short: "Resolve public Go Modules with the canonical proxy and SumDB"}
	goMod.AddCommand(newStaticNoArgLeaf("download", "Resolve the current project's exact Go Module graph"))
	goCommand.AddCommand(goMod)
}

func pythonSourceHelp() string {
	names := make([]string, 0)
	for _, profile := range artifactpypi.AllSourceProfiles() {
		names = append(names, profile.Name())
	}
	return "named source profile: " + strings.Join(names, "/")
}

func findRootCommand(root *cobra.Command, name string) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() == name {
			return command
		}
	}
	return nil
}

func newStaticLeaf(use, short string, withTarget bool) *cobra.Command {
	command := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(*cobra.Command, []string) error {
			return errors.New("command is not configured")
		},
	}
	if withTarget {
		command.Flags().String("target", "", "advanced absolute destination (optional; existing paths are never overwritten)")
	}
	return command
}

func newStaticNoArgLeaf(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return errors.New("command is not configured")
		},
	}
}

func findLeaf(root *cobra.Command, parent, leaf string) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() != parent {
			continue
		}
		for _, child := range command.Commands() {
			if child.Name() == leaf {
				return child
			}
		}
	}
	return nil
}

func ensureNPMCommand(root *cobra.Command) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() == "npm" {
			return command
		}
	}
	command := &cobra.Command{Use: "npm", Short: "Inspect and install npm registry packages"}
	root.AddCommand(command)
	return command
}

func ensurePyPICommand(root *cobra.Command) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() == "pypi" {
			return command
		}
	}
	command := &cobra.Command{Use: "pypi", Short: "Inspect and install PyPI distributions"}
	root.AddCommand(command)
	return command
}

func ensurePipCommand(root *cobra.Command) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() == "pip" {
			return command
		}
	}
	command := &cobra.Command{Use: "pip", Short: "Inspect and install PyPI distributions"}
	root.AddCommand(command)
	return command
}

func ensureGitHubCommand(root *cobra.Command) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() == "github" {
			return command
		}
	}
	command := &cobra.Command{Use: "github", Short: "Inspect and install public GitHub Release assets"}
	root.AddCommand(command)
	return command
}

func ensureGoCommand(root *cobra.Command) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() == "go" {
			return command
		}
	}
	command := &cobra.Command{Use: "go", Short: "Inspect and build public Go Modules"}
	root.AddCommand(command)
	return command
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
