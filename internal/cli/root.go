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
	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	artifactgomodule "github.com/rahoney/heliopause/internal/artifact/gomodule"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
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

		return runLauncherOrHelp(command)
	}
	command.SetOut(stdout)
	command.SetErr(stderr)
	initializeCommandTree(command)
	attachPresentation(command, stdout, stderr)

	return command, nil
}

type ReferenceParser func(string) (domain.ArtifactReference, error)

// GoModuleResolver is the M12 application boundary used by `helox go get`.
// The CLI only parses user input and renders the bounded resolution summary.
type GoModuleResolver interface {
	Get(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error)
}

// GoModuleProjectResolver resolves a complete current-project snapshot for
// commands that do not name one primary module.
type GoModuleProjectResolver interface {
	Resolve(context.Context, domain.InstallContext) (domain.ProjectDependencySnapshot, error)
}

type CargoResolver interface {
	Resolve(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error)
}

type TerraformResolver interface {
	Init(context.Context, domain.ArtifactReference, domain.InstallContext) (application.TerraformInitResult, error)
}

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
	presentationFor(root.OutOrStdout()).doctor = doctor
	command.RunE = func(command *cobra.Command, _ []string) error {
		report := doctor.Diagnose(contextOrBackground(command.Context()))
		if p := presentationFor(command.OutOrStdout()); p.human() {
			if err := p.renderDoctor(report); err != nil {
				return err
			}
			if !report.Healthy {
				return ExitError{Code: 1}
			}
			return nil
		}
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
			code, err := ExecuteInspect(contextOrBackground(command.Context()), inspector, request, machineOutput(command), command.OutOrStdout())
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
			if err := validateGitHubInstallArgs(command, args); err != nil {
				return err
			}
			if replace, _ := command.Flags().GetBool("replace"); replace {
				return errors.New("github replacement use case is not configured")
			}
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
			code, err := ExecuteInstall(contextOrBackground(command.Context()), installer, request, machineOutput(command), command.OutOrStdout())
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
	return AddPyPIInstallSourcesAndUpdates(root, installers, nil)
}

// AddPyPIInstallSourcesAndUpdates binds native upgrade intent to the guarded
// workflow, preserving the ordinary install path for all existing profiles.
func AddPyPIInstallSourcesAndUpdates(root *cobra.Command, installers map[string]Installer, updaters map[string]Installer) error {
	return addPackageClientInstall(root, PipPackageClient, installers, map[packageUpgradeMode]map[string]Installer{packageUpgradeRoot: updaters})
}

func addPackageClientInstall(root *cobra.Command, id PackageClientID, installers map[string]Installer, updaters map[packageUpgradeMode]map[string]Installer) error {
	if root == nil || len(installers) == 0 {
		return errors.New("pypi install command requires root and use case")
	}
	descriptor, ok := packageClientByID(id)
	if !ok {
		return errors.New("unsupported package client binding")
	}
	if installCommand, findErr := PackageClientInstallCommand(root, id); findErr == nil {
		installCommand.RunE = func(command *cobra.Command, args []string) error {
			if err := descriptor.validate(command, args); err != nil {
				return err
			}
			source := "pypi"
			var err error
			if command.Flags().Lookup("source") != nil {
				source, err = command.Flags().GetString("source")
				if err != nil {
					return err
				}
			}
			if source == "" {
				source = "pypi"
			}
			sourceID := artifactpypi.PublicPyPIProfile().Source()
			if source != "pypi" {
				if !descriptor.namedSources {
					return errors.New("package client source is unsupported")
				}
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
			upgradeMode, err := descriptor.upgrade(command, args)
			if err != nil {
				return err
			}
			target, err := command.Flags().GetString("target")
			if err != nil {
				return err
			}
			installTarget, err := activePythonVenvForCommand(target, strings.Join(descriptor.path, " "))
			if err != nil {
				return err
			}
			installContext, err := domain.NewPythonVenvInstallContext(installTarget)
			if err != nil {
				return err
			}
			var request application.InstallRequest
			if upgradeMode != packageNoUpgrade {
				installer = updaters[upgradeMode][sourceID.String()]
				if installer == nil {
					return errors.New("guarded upgrade is unavailable for the selected Python source")
				}
				adopt, _ := command.Flags().GetBool("adopt")
				request, err = application.NewInstallUpdateRequest(sourceID, args[0], installContext, adopt)
			} else {
				reference, parseErr := descriptor.parse(args[0], sourceID)
				if parseErr != nil {
					return parseErr
				}
				request, err = application.NewInstallRequest(reference, installContext)
			}
			if err != nil {
				return err
			}
			exitCode, operationErr := ExecuteInstall(contextOrBackground(command.Context()), installer, request, machineOutput(command), command.OutOrStdout())
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
	return errors.New(strings.Join(descriptor.path, " ") + " command is not registered")
}

func activePythonVenv(target string) (domain.InstallTarget, error) {
	return activePythonVenvForCommand(target, "pip install")
}

func activePythonVenvForCommand(target, command string) (domain.InstallTarget, error) {
	if target == "" {
		target = os.Getenv("VIRTUAL_ENV")
	}
	if target == "" {
		return domain.InstallTarget{}, errors.New(command + " requires an active virtual environment")
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
			exitCode, operationErr := ExecuteInspect(contextOrBackground(command.Context()), inspector, request, machineOutput(command), command.OutOrStdout())
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
			exitCode, operationErr := ExecuteInspect(contextOrBackground(command.Context()), inspector, request, machineOutput(command), command.OutOrStdout())
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
			exitCode, operationErr := ExecuteInstall(contextOrBackground(command.Context()), installer, request, machineOutput(command), command.OutOrStdout())
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

// AddGoModuleGet binds the guarded, inspected and approved frozen project
// update to the static command tree. A mere resolution does not update files.
func AddGoModuleGet(root *cobra.Command, resolver GoModuleResolver) error {
	if root == nil || resolver == nil {
		return errors.New("go get command requires a resolution use case")
	}
	command := findLeaf(root, "go", "get")
	if command == nil {
		return errors.New("go get command is not registered")
	}
	command.RunE = func(command *cobra.Command, args []string) error {
		reference, err := artifactgomodule.ParseReference(args[0])
		if err != nil {
			return err
		}
		workingDirectory, err := os.Getwd()
		if err != nil {
			return errors.New("resolve Go project directory")
		}
		target, err := domain.NewInstallTarget(workingDirectory)
		if err != nil {
			return errors.New("go project directory is unsupported")
		}
		installContext, err := domain.NewInstallContext(target)
		if err != nil {
			return err
		}
		progressContext, finishProgress := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
		defer func() { _ = finishProgress() }()
		resolution, err := resolver.Get(progressContext, reference, installContext)
		progressErr := finishProgress()
		err = errors.Join(err, progressErr)
		if err != nil {
			return err
		}
		if p := presentationFor(command.OutOrStdout()); p.human() {
			return p.renderProjectSummary(reference.Source().String(), "Modules", len(resolution.Graph().Nodes()), resolution.LockfileDigest().String())
		}
		_, err = fmt.Fprintf(command.OutOrStdout(), "Source: %s\nModules: %d\nLock digest: %s\n", reference.Source(), len(resolution.Graph().Nodes()), resolution.LockfileDigest())
		return err
	}
	return nil
}

// AddGoModuleDownload binds the complete project snapshot boundary to `go mod
// download`. The use case completes inspection and verified-cache promotion.
func AddGoModuleDownload(root *cobra.Command, resolver GoModuleProjectResolver) error {
	if root == nil || resolver == nil {
		return errors.New("go mod download command requires a project resolution use case")
	}
	goCommand := findRootCommand(root, "go")
	modCommand := findRootCommand(goCommand, "mod")
	command := findRootCommand(modCommand, "download")
	if command == nil {
		return errors.New("go mod download command is not registered")
	}
	command.RunE = func(command *cobra.Command, _ []string) error {
		installContext, err := currentGoProjectContext()
		if err != nil {
			return err
		}
		progressContext, finishProgress := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
		defer func() { _ = finishProgress() }()
		snapshot, err := resolver.Resolve(progressContext, installContext)
		progressErr := finishProgress()
		err = errors.Join(err, progressErr)
		if err != nil {
			return err
		}
		if p := presentationFor(command.OutOrStdout()); p.human() {
			return p.renderProjectSummary(snapshot.Source().String(), "Modules", len(snapshot.Dependencies()), snapshot.GraphDigest().String())
		}
		_, err = fmt.Fprintf(command.OutOrStdout(), "Source: %s\nModules: %d\nGraph digest: %s\n", snapshot.Source(), len(snapshot.Dependencies()), snapshot.GraphDigest())
		return err
	}
	return nil
}

func AddCargoAdd(root *cobra.Command, resolver CargoResolver) error {
	if root == nil || resolver == nil {
		return errors.New("cargo add command requires a resolution use case")
	}
	command := findLeaf(root, "cargo", "add")
	if command == nil {
		return errors.New("cargo add command is not registered")
	}
	command.RunE = func(command *cobra.Command, args []string) error {
		reference, err := artifactcargo.ParseReference(args[0])
		if err != nil {
			return err
		}
		workingDirectory, err := os.Getwd()
		if err != nil {
			return errors.New("resolve Cargo project directory")
		}
		target, err := domain.NewInstallTarget(workingDirectory)
		if err != nil {
			return errors.New("cargo project directory is unsupported")
		}
		installContext, err := domain.NewInstallContext(target)
		if err != nil {
			return err
		}
		progressContext, finishProgress := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
		defer func() { _ = finishProgress() }()
		resolution, err := resolver.Resolve(progressContext, reference, installContext)
		progressErr := finishProgress()
		err = errors.Join(err, progressErr)
		if err != nil {
			return err
		}
		if p := presentationFor(command.OutOrStdout()); p.human() {
			return p.renderProjectSummary(reference.Source().String(), "Crates", len(resolution.Graph().Nodes()), resolution.LockfileDigest().String())
		}
		_, err = fmt.Fprintf(command.OutOrStdout(), "Source: %s\nCrates: %d\nLock digest: %s\n", reference.Source(), len(resolution.Graph().Nodes()), resolution.LockfileDigest())
		return err
	}
	return nil
}

func AddTerraformInit(root *cobra.Command, resolver TerraformResolver) error {
	if root == nil || resolver == nil {
		return errors.New("terraform init command requires a guarded installation use case")
	}
	command := findLeaf(root, "terraform", "init")
	if command == nil {
		return errors.New("terraform init command is not registered")
	}
	command.RunE = func(command *cobra.Command, args []string) error {
		if err := validateTerraformInitArgs(command, args); err != nil {
			return err
		}
		if upgrade, _ := command.Flags().GetBool("upgrade"); upgrade {
			return errors.New("terraform upgrade service is unavailable")
		}
		reference, err := artifactterraform.ParseReference(args[0])
		if err != nil {
			return err
		}
		workingDirectory, err := os.Getwd()
		if err != nil {
			return errors.New("resolve Terraform project directory")
		}
		target, err := domain.NewInstallTarget(workingDirectory)
		if err != nil {
			return errors.New("terraform project directory is unsupported")
		}
		installContext, err := domain.NewInstallContext(target)
		if err != nil {
			return err
		}
		progressContext, finishProgress := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
		defer func() { _ = finishProgress() }()
		result, err := resolver.Init(progressContext, reference, installContext)
		progressErr := finishProgress()
		err = errors.Join(err, progressErr)
		if err != nil {
			return err
		}
		if p := presentationFor(command.OutOrStdout()); p.human() {
			return p.renderProjectSummary(reference.Source().String(), "Providers", result.ProviderCount, result.Resolution.LockfileDigest().String())
		}
		_, err = fmt.Fprintf(command.OutOrStdout(), "Source: %s\nProviders: %d\nLock digest: %s\n", reference.Source(), result.ProviderCount, result.Resolution.LockfileDigest())
		return err
	}
	return nil
}

func currentGoProjectContext() (domain.InstallContext, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return domain.InstallContext{}, errors.New("resolve Go project directory")
	}
	target, err := domain.NewInstallTarget(workingDirectory)
	if err != nil {
		return domain.InstallContext{}, errors.New("go project directory is unsupported")
	}
	installContext, err := domain.NewInstallContext(target)
	if err != nil {
		return domain.InstallContext{}, err
	}
	return installContext, nil
}

func initializeCommandTree(root *cobra.Command) {
	root.AddCommand(newSelfUpdateCommand())
	root.AddCommand(newBootstrapCommand())
	root.AddCommand(&cobra.Command{Use: "doctor", Short: "Check installed release and trusted Host runtime readiness", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error { return errors.New("doctor command is not configured") }})
	npm := ensureNPMCommand(root)
	npm.AddCommand(newStaticLeaf("inspect <package-reference>", "Inspect an npm package", false))
	npm.AddCommand(newStaticLeaf("install <package-reference>", "Install an npm package into the current project", true))
	npmUpdate := newStaticLeaf("update [package]", "Update an npm project after complete quarantine checks", true)
	npmUpdate.Args = validateNPMUpdateArgs
	npmUpdate.Long = npmUpdateDescriptionEnglish
	npmUpdate.Flags().Bool("adopt", false, "Explicitly replace supported native project dependencies after full inspection")
	npm.AddCommand(npmUpdate)
	pypi := ensurePyPICommand(root)
	pypi.AddCommand(newStaticLeaf("inspect <project>[@<version>]", "Inspect a PyPI distribution", false))
	initializePackageClientCommands(root)
	github := ensureGitHubCommand(root)
	github.AddCommand(newStaticLeaf("inspect <owner>/<repo>@<tag>#<asset>", "Inspect a GitHub Release asset", false))
	githubInstall := newStaticLeaf("install <owner>/<repo>@<tag>#<asset>", "Install a GitHub Release asset", true)
	githubInstall.Args = validateGitHubInstallArgs
	githubInstall.Long = githubReplacementDescriptionEnglish
	githubInstall.Flags().Bool("replace", false, "Replace only a proven HAA-managed asset after fresh inspection; requires --target")
	github.AddCommand(githubInstall)
	goCommand := ensureGoCommand(root)
	goGet := newStaticLeaf("get <module>[@<version>]", "Inspect exact or upgraded public Go Module dependencies", false)
	goGet.Args = validateGoGetArgs
	goGet.Long = goUpgradeDescriptionEnglish
	goGet.Flags().BoolP("upgrade", "u", false, "Inspect the whole selected graph before upgrading one existing module and its imported dependencies")
	goGet.Flags().Bool("adopt", false, "Explicitly enroll supported native or changed dependency controls with -u")
	goCommand.AddCommand(goGet)
	goCommand.AddCommand(newStaticLeaf("build <package>", "Build a Go project from the HAA-verified module cache", false))
	goMod := &cobra.Command{Use: "mod", Short: "Resolve public Go Modules with the canonical proxy and SumDB"}
	goMod.AddCommand(newStaticNoArgLeaf("download", "Resolve the current project's exact Go Module graph"))
	goCommand.AddCommand(goMod)
	cargo := ensureCargoCommand(root)
	cargo.AddCommand(newStaticLeaf("add <crate>@<version>", "Resolve and transactionally add a public crates.io crate", false))
	cargo.AddCommand(newStaticNoArgLeaf("build", "Build a Cargo project from the HAA-verified crate cache"))
	cargoUpdate := newStaticNoArgLeaf("update", "Inspect the complete selected crate graph before updating Cargo.lock")
	cargoUpdate.Args = validateCargoUpdateArgs
	cargoUpdate.Long = cargoUpdateDescriptionEnglish
	cargoUpdate.Flags().StringArrayP("package", "p", nil, "Select one existing public crate by name or name@locked-version")
	cargoUpdate.Flags().Bool("adopt", false, "Explicitly enroll supported native or changed project controls after full inspection")
	cargo.AddCommand(cargoUpdate)
	terraform := ensureTerraformCommand(root)
	terraformInit := newStaticLeaf("init [namespace/type@version]", "Resolve exact or upgraded public Terraform Providers", false)
	terraformInit.Args = validateTerraformInitArgs
	terraformInit.Long = terraformUpgradeDescriptionEnglish
	terraformInit.Flags().Bool("upgrade", false, "Inspect every required provider at its newest stable constrained version (native -upgrade supported)")
	terraformInit.Flags().Bool("adopt", false, "Explicitly enroll supported providerless native or changed managed controls with -upgrade")
	terraform.AddCommand(terraformInit)
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

func ensureCargoCommand(root *cobra.Command) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() == "cargo" {
			return command
		}
	}
	command := &cobra.Command{Use: "cargo", Short: "Inspect and build public crates.io packages"}
	root.AddCommand(command)
	return command
}

func ensureTerraformCommand(root *cobra.Command) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() == "terraform" {
			return command
		}
	}
	command := &cobra.Command{Use: "terraform", Short: "Install public Terraform Providers"}
	root.AddCommand(command)
	return command
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
