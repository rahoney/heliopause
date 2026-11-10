package sandbox

import (
	"context"
	"errors"
)

// This is only an accepted target image-load fact for one admission. It is
// independent of root exit, source approval, output completeness and cleanup.
type directExecTarget struct{ loaded, missed bool }

func parseDirectExecTarget(status string) (directExecTarget, error) {
	switch status {
	case "pending", "waiting":
		return directExecTarget{}, nil
	case "loaded":
		return directExecTarget{loaded: true}, nil
	case "missed":
		return directExecTarget{missed: true}, nil
	default:
		return directExecTarget{}, errors.New("observer target image fact is invalid")
	}
}

func directExecTargetLoaded(ctx context.Context, admission directExecAdmission) (directExecTarget, error) {
	ack, err := exchangeDirectExecAdmission(ctx, "target", admission, nil)
	if err != nil {
		return directExecTarget{}, err
	}
	return parseDirectExecTarget(ack["status"])
}
