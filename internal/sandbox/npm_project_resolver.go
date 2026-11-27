package sandbox

import (
	"bytes"
	"context"
	"errors"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type NPMNativeProjectSelection struct {
	controls  []domain.ProjectControlFile
	selection artifactnpm.NPMProjectLock
}

func (s NPMNativeProjectSelection) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), s.controls...)
}
func (s NPMNativeProjectSelection) Selection() artifactnpm.NPMProjectLock { return s.selection }

func NPMProjectRuntimeIdentity() string {
	return resolverRuntimeIdentity + ";runtime-lock-sha256:" + runtimeidentity.RuntimeLockSHA256
}

// SelectProject returns only unapproved native controls/placement. Both intents
// use the existing isolated npm lifecycle, without project code or configuration.
func (r *NPMResolver) SelectProject(ctx context.Context, input artifactnpm.NPMProjectInput) (NPMNativeProjectSelection, error) {
	if ctx == nil || r == nil || input.ClientVersion() != resolverNPMVersion {
		return NPMNativeProjectSelection{}, errors.New("npm project selector input is unavailable")
	}
	validated, err := artifactnpm.NewNPMProjectInput(input.Intent(), input.Controls(), resolverNPMVersion)
	if err != nil {
		return NPMNativeProjectSelection{}, err
	}
	streams := r.projectOutput
	if streams == nil {
		return NPMNativeProjectSelection{}, errors.New("npm project selection requires bounded control output")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	var result NPMNativeProjectSelection
	err = r.withNPMResolver(ctx, func(ctx context.Context, container string) error {
		transport, ok := r.runner.(inputCommandRunner)
		if !ok {
			return errors.New("npm project selection requires input transport")
		}
		for _, control := range validated.Controls() {
			if !control.Present() {
				continue
			}
			// Names are the two fixed control names validated above. Bytes pass
			// through stdin and never enter shell code or the Host filesystem.
			command := "umask 077; "
			if control.Name() == "package.json" {
				command += "mkdir -p " + resolverProjectDir + "; "
			}
			command += "cat > " + resolverProjectDir + "/" + control.Name()
			if err := transport.RunInput(ctx, bytes.NewReader(control.Body()), "docker", boundaryInputExecArguments(container, boundaryLaunchMode, "/bin/sh", "-ceu", command)...); err != nil {
				return errors.Join(errors.New("write frozen npm project control"), err)
			}
		}
		selection := "npm install --package-lock-only"
		if validated.Intent() == artifactnpm.NPMProjectCI {
			// Native ci validates its frozen ideal tree. Dry-run selects the
			// complete plan without installing uninspected archives or scripts.
			selection = "npm ci --dry-run"
		}
		command := "cd " + resolverProjectDir + "; HOME=/tmp npm_config_cache=/tmp/cache " + selection + " --ignore-scripts --no-audit --no-fund --registry=https://registry.npmjs.org/ --userconfig=/tmp/haa-user.npmrc --globalconfig=/tmp/haa-global.npmrc"
		if _, err := r.runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, "/bin/sh", "-ceu", command)...); err != nil {
			return errors.Join(errors.New("native npm project selection failed"), err)
		}
		controls := make([]domain.ProjectControlFile, 0, 2)
		for _, name := range []string{"package.json", "package-lock.json"} {
			output := nativeToolArchiveOutput{limit: artifactnpm.ProjectControlLimit}
			if err := streams.RunOutput(ctx, &output, "docker", boundaryExecArguments(container, boundaryLaunchMode, "cat", resolverProjectDir+"/"+name)...); err != nil || output.failed || ctx.Err() != nil {
				return errors.Join(errors.New("read bounded npm project control"), err, ctx.Err())
			}
			control, err := domain.NewProjectControlFile(name, output.body, true)
			if err != nil {
				return err
			}
			controls = append(controls, control)
		}
		controls, err = validated.ValidateOutput(controls[0].Body(), controls[1].Body())
		if err != nil {
			return err
		}
		lock, err := artifactnpm.ParseNPMProjectLock(controls[0].Body(), controls[1].Body(), resolverNPMVersion)
		if err != nil {
			return err
		}
		result = NPMNativeProjectSelection{controls, lock}
		return nil
	})
	if err != nil || ctx.Err() != nil {
		return NPMNativeProjectSelection{}, errors.Join(err, ctx.Err())
	}
	return result, nil
}
