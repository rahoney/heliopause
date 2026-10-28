package sandbox

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

// directExecTerminal is an externally observed root outcome, not coverage,
// output completeness, descendant cleanup, or Policy approval.
type directExecTerminal struct {
	observed bool
	exited   bool
	exitCode uint8
	signal   uint8
	coreDump bool
}

func parseDirectExecOutcome(status string) (directExecTerminal, error) {
	if status == "pending" || status == "running" {
		return directExecTerminal{}, nil
	}
	number, ok := strings.CutPrefix(status, "exited:")
	if !ok {
		return directExecTerminal{}, errors.New("observer terminal outcome is invalid")
	}
	wait, err := strconv.ParseUint(number, 10, 16)
	if err != nil || strconv.FormatUint(wait, 10) != number {
		return directExecTerminal{}, errors.New("observer wait status is invalid")
	}
	low := wait & 255
	if low == 0 {
		return directExecTerminal{observed: true, exited: true, exitCode: uint8(wait >> 8)}, nil
	}
	signal := low & 127
	if wait > 255 || signal == 0 || signal > 64 {
		return directExecTerminal{}, errors.New("observer wait status is not terminal")
	}
	return directExecTerminal{observed: true, signal: uint8(signal), coreDump: low&128 != 0}, nil
}

// This queries the existing protected, ordered admission peer. A caller must
// retain its exact slot until the kernel outcome has been reconciled; COMPLETE
// releases that fact. An absent/foreign/ambiguous slot is never a zero exit.
func directExecOutcome(ctx context.Context, admission directExecAdmission) (directExecTerminal, error) {
	ack, err := exchangeDirectExecAdmission(ctx, "outcome", admission, nil)
	if err != nil {
		return directExecTerminal{}, err
	}
	return parseDirectExecOutcome(ack["status"])
}
