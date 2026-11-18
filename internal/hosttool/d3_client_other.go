//go:build !linux

package hosttool

import (
	"context"
	"errors"
)

type d3LeaseClient struct{}

func (e *Executor) ProtectExecution(ctx context.Context) (context.Context, error) {
	return ctx, errors.New("protected sandbox execution requires Linux")
}

func (e *Executor) protectDocker(_ context.Context, args []string) ([]string, func(error) error, error) {
	return args, nil, nil
}

func (e *Executor) runProtectedPromotion(context.Context, []string) error {
	return errors.New("protected sandbox execution requires Linux")
}

func (e *Executor) RecordRecoveryObserver(context.Context, ObserverProcess) error {
	return errors.New("protected sandbox execution requires Linux")
}

func (e *Executor) PrepareRecoveryObserver(context.Context) error {
	return errors.New("protected sandbox execution requires Linux")
}

func (e *Executor) CancelRecoveryObserver(context.Context) error {
	return errors.New("protected sandbox execution requires Linux")
}
