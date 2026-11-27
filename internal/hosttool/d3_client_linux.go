//go:build linux

package hosttool

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"slices"
	"strings"
	"sync"
	"time"
)

type d3LeaseClient struct {
	request *ObservationResourceClient
	owner   string
}

// DockerRecoveryLabel identifies the additional label this Executor assigns to
// each resource it creates during a protected operation.
func (e *Executor) DockerRecoveryLabel() (key, value string) {
	if e == nil || e.recovery == nil {
		return "", ""
	}
	return d3OwnerLabel, e.recovery.owner
}

func (e *Executor) ProtectExecution(ctx context.Context) (context.Context, error) {
	if ctx == nil || e == nil || e.recovery != nil {
		return ctx, errD3Unconfirmed
	}
	client, err := NewSystemObservationResourceClient()
	if err != nil {
		return ctx, err
	}
	connection, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", client.endpoint)
	if err != nil {
		return ctx, errD3Unconfirmed
	}
	_ = connection.SetDeadline(time.Now().Add(5 * time.Second))
	decoder := json.NewDecoder(connection)
	decoder.DisallowUnknownFields()
	var response observationResourceResponse
	if json.NewEncoder(connection).Encode(observationResourceRequest{Operation: "hold"}) != nil ||
		decoder.Decode(&response) != nil || !response.OK || !validProcessOwner(response.Owner) ||
		verifyPolicySocket(client.endpoint, client.group) != nil {
		_ = connection.Close()
		return ctx, errD3Unconfirmed
	}
	_ = connection.SetDeadline(time.Time{})
	guarded, cancel := context.WithCancelCause(ctx)
	done := make(chan error, 1)
	e.recovery = &d3LeaseClient{request: client, owner: response.Owner}
	go func() {
		var response observationResourceResponse
		var result error
		if err := decoder.Decode(&response); err != nil || !response.OK {
			result = errD3Unconfirmed
		}
		if result != nil {
			cancel(result)
		}
		done <- result
	}()
	var once sync.Once
	var result error
	e.releaseRecovery = func() error {
		once.Do(func() {
			defer connection.Close()
			_ = connection.SetDeadline(time.Now().Add(20 * time.Second))
			if _, err := io.WriteString(connection, "{\"operation\":\"release\"}\n"); err != nil {
				result = errD3Unconfirmed
				return
			}
			result = <-done
		})
		return result
	}
	return guarded, nil
}

func (e *Executor) protectDocker(ctx context.Context, args []string) ([]string, func(error) error, error) {
	if e.recovery == nil || len(args) == 0 {
		return args, nil, nil
	}
	lease := e.recovery
	prefix := 0
	switch args[0] {
	case "create":
		prefix = 1
	case "network", "volume":
		if len(args) > 1 && args[1] == "create" {
			prefix = 2
		}
	case "start", "exec":
		id := dockerExecutionTarget(args)
		if !validDockerContainerID(id) {
			return nil, nil, errD3Unconfirmed
		}
		operation := "guard-start"
		if args[0] == "exec" {
			operation = "guard-exec"
		}
		_, err := lease.request.call(ctx, observationResourceRequest{Operation: operation, Owner: lease.owner, Container: id})
		return args, nil, err
	case "run", "restart", "unpause":
		return nil, nil, errD3Unconfirmed
	}
	if prefix == 0 {
		return args, nil, nil
	}
	for _, arg := range args[prefix:] {
		if strings.Contains(arg, d3OwnerLabel) {
			return nil, nil, errD3Unconfirmed
		}
	}
	if _, err := lease.request.call(ctx, observationResourceRequest{Operation: "guard-create", Owner: lease.owner}); err != nil {
		return nil, nil, err
	}
	key, value := e.DockerRecoveryLabel()
	owned := append(append([]string{}, args[:prefix]...), "--label", key+"="+value)
	owned = append(owned, args[prefix:]...)
	return owned, func(runErr error) error {
		// No acknowledgement may be sent for a possibly late Docker create.
		if runErr != nil {
			return runErr
		}
		finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_, err := lease.request.call(finish, observationResourceRequest{Operation: "guard-created", Owner: lease.owner})
		return errors.Join(runErr, err)
	}, nil
}

func dockerExecutionTarget(args []string) string {
	if args[0] == "start" {
		if len(args) == 2 {
			return args[1]
		}
		if len(args) == 4 && args[1] == "--attach" && args[2] == "--" {
			return args[3]
		}
		if len(args) == 5 && args[1] == "--attach" && args[2] == "--interactive" && args[3] == "--" {
			return args[4]
		}
		return ""
	}
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			return arg
		}
		switch arg {
		case "--":
			if i+1 < len(args) {
				return args[i+1]
			}
		case "-i", "-t", "-it", "--interactive", "--tty":
		case "-u", "--user", "-w", "--workdir", "-e", "--env":
			i++
		default:
			if !strings.HasPrefix(arg, "--user=") && !strings.HasPrefix(arg, "--workdir=") && !strings.HasPrefix(arg, "--env=") {
				return ""
			}
		}
	}
	return ""
}

// The existing trusted offline promotion previously used Docker run --rm.
// Separate create from start so the helper can retain the exact container ID
// before any code can run. The existing promotion arguments remain unchanged.
func (e *Executor) runProtectedPromotion(ctx context.Context, args []string) error {
	if len(args) < 3 || args[1] != "--rm" {
		return errD3Unconfirmed
	}
	create := append([]string{"create"}, slices.Clone(args[2:])...)
	out, err := e.output(ctx, "docker", create...)
	id := strings.TrimSpace(string(out))
	if err != nil || !validDockerContainerID(id) {
		return errD3Unconfirmed
	}
	runErr := e.RunDiscard(ctx, "docker", "start", "--attach", "--", id)
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 7*time.Second)
	defer cancel()
	return errors.Join(runErr, e.RunDiscard(cleanup, "docker", "rm", "--force", id))
}
