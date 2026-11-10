package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"
	"testing"
)

func TestDirectExecOutcomeDecodesOnlyTerminalKernelWaitStatus(t *testing.T) {
	t.Parallel()
	for _, status := range []string{"pending", "running"} {
		result, err := parseDirectExecOutcome(status)
		if err != nil || result.observed {
			t.Fatal("pending outcome became zero exit", status, err)
		}
	}
	for _, code := range []uint8{0, 37, 125, 126, 127, 255} {
		result, err := parseDirectExecOutcome("exited:" + strconv.Itoa(int(code)<<8))
		if err != nil || !result.observed || !result.exited || result.exitCode != code || result.signal != 0 || result.coreDump {
			t.Fatal("exit code changed", code, result, err)
		}
	}
	for _, signal := range []uint8{1, 9, 15, 64} {
		for _, core := range []bool{false, true} {
			status := int(signal)
			if core {
				status |= 128
			}
			result, err := parseDirectExecOutcome("exited:" + strconv.Itoa(status))
			if err != nil || !result.observed || result.exited || result.signal != signal || result.coreDump != core {
				t.Fatal("signal changed", signal, core, result, err)
			}
		}
	}
	for _, status := range []string{"", "consumed", "rejected", "exited:", "exited:-1", "exited:+1", "exited:00", "exited:01", "exited: 0", "exited:0\n", "exited:65536", "exited:65535", "exited:127", "exited:128", "exited:65", "exited:193", "exited:257", "exited:1.0", "exited:1e2"} {
		if result, err := parseDirectExecOutcome(status); err == nil || result.observed {
			t.Fatal("invalid or nonterminal status accepted", status)
		}
	}
}

func TestDirectExecOutcomeUsesExactAdmissionAuthority(t *testing.T) {
	for _, variant := range []string{"pending", "running", "exited:9472", "exited:9", "rejected", "exited:65535", "foreign nonce", "foreign generation", "foreign container", "foreign operation"} {
		t.Run(variant, func(t *testing.T) {
			withAdmissionControl(t, func(conn *net.UnixConn, request admissionRequest) {
				var reply []byte
				switch request.Op {
				case "profile":
					reply = profileACK(request)
				case "outcome":
					status := variant
					if strings.HasPrefix(variant, "foreign ") {
						status = "exited:0"
						switch variant {
						case "foreign nonce":
							request.Nonce = strings.Repeat("c", 64)
						case "foreign generation":
							request.Generation = strings.Repeat("c", 64)
						case "foreign container":
							request.ContainerID = "fedcba9876543210"
						case "foreign operation":
							request.Op = "status"
						}
					}
					reply = admissionACK(request, status)
				case "invalidate":
					reply, _ = json.Marshal(map[string]string{"op": "ack", "ack_op": "invalidate", "status": "invalidated", "container_id": request.ContainerID, "session_generation": request.Generation})
				}
				_, _ = conn.Write(reply)
			})
			session := registerAdmissionSession(t, "0123456789abcdef")
			nonce, err := newDirectExecAdmissionNonce()
			if err != nil {
				t.Fatal(err)
			}
			admission := directExecAdmission{containerID: session.containerID, generation: session.generation, mode: boundaryLaunchMode, nonce: nonce, session: session}
			result, err := directExecOutcome(context.Background(), admission)
			valid := variant == "pending" || variant == "running" || variant == "exited:9472" || variant == "exited:9"
			if valid {
				want, _ := parseDirectExecOutcome(variant)
				if err != nil || result != want {
					t.Fatal("valid terminal fact changed", result, err)
				}
			} else if err == nil || result.observed || !errors.Is(err, errObserverAuthorityAmbiguous) {
				t.Fatal("invalid/foreign outcome became authority", result, err)
			}
		})
	}
}
