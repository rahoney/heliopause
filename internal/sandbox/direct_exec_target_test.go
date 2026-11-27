package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"testing"
)

func TestDirectExecTargetRejectsNonTargetStatuses(t *testing.T) {
	t.Parallel()
	for _, status := range []string{"", "consumed", "running", "exited:0", "loaded\n", "Loaded", " loaded"} {
		if fact, err := parseDirectExecTarget(status); err == nil || fact.loaded || fact.missed {
			t.Fatal("invalid target fact accepted", status)
		}
	}
}

func TestDirectExecTargetUsesExactAdmissionAuthority(t *testing.T) {
	for _, variant := range []string{"pending", "waiting", "loaded", "missed", "rejected", "exited:0", "foreign nonce", "foreign generation", "foreign container", "foreign operation"} {
		t.Run(variant, func(t *testing.T) {
			withAdmissionControl(t, func(conn *net.UnixConn, request admissionRequest) {
				var reply []byte
				switch request.Op {
				case "profile":
					reply = profileACK(request)
				case "target":
					status := variant
					if strings.HasPrefix(variant, "foreign ") {
						status = "loaded"
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
			result, err := directExecTargetLoaded(context.Background(), admission)
			valid := variant == "pending" || variant == "waiting" || variant == "loaded" || variant == "missed"
			if valid {
				want, _ := parseDirectExecTarget(variant)
				if err != nil || result != want {
					t.Fatal("valid terminal fact changed", result, err)
				}
			} else if err == nil || result.loaded || result.missed || !errors.Is(err, errObserverAuthorityAmbiguous) {
				t.Fatal("invalid/foreign outcome became authority", result, err)
			}
		})
	}
}
