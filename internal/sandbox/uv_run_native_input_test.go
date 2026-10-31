package sandbox

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestUVRunFrozenSessionRejectsInvalidInputsBeforeProbe(t *testing.T) {
	frozen, _, _ := uvRunBindingFixture(t)
	for _, kind := range []string{"nil input", "multiple inputs", "tool install input", "canceled"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			resolver := &UVResolver{reader: &PyPIResolver{probe: func(context.Context) (PythonCapability, error) {
				calls++
				return PythonCapability{}, errors.New("must not probe")
			}}}
			input := []*pipxNativeInput{frozen}
			tool := false
			ctx := context.Background()
			switch kind {
			case "nil input":
				input = []*pipxNativeInput{nil}
			case "multiple inputs":
				input = append(input, frozen)
			case "tool install input":
				tool = true
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			result, err := resolver.withUVNativeSessionForToolInstall(ctx, func(context.Context, uvNativeSession) (uvNativeOutput, error) {
				t.Fatal("invalid frozen input reached operation")
				return uvNativeOutput{}, nil
			}, tool, input...)
			if err == nil || calls != 0 || len(result.lock) != 0 || len(result.controls) != 0 {
				t.Fatal("invalid offline input reached probe or returned DATA", err)
			}
		})
	}
}

func TestUVRunFrozenArgumentsKeepOnlyOfflineReadonlyOriginals(t *testing.T) {
	frozen, _, _ := uvRunBindingFixture(t)
	runtime := uvRuntimeVolume{name: "haa-uv-runtime-test", owner: "test", createdAt: "fixed", mountpoint: "/owned"}
	output := closureVolume{name: "haa-closure-output", transaction: "output", manifestID: frozen.id, createdAt: "fixed", mountpoint: "/output"}
	wheels := &pipxWheelVolume{volume: closureVolume{name: "haa-closure-wheels", transaction: "wheels", manifestID: frozen.id, capacity: pipxWheelInputCapacity, createdAt: "fixed", mountpoint: "/wheels", projectBuildDestination: pipxNativeWheels}, input: frozen, seeded: true, sealed: true}
	args, err := uvFrozenRunCreateArguments(runtime, output, wheels)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--network none") || !strings.Contains(joined, "target=/haa-wheels,volume-nocopy,readonly") || strings.Contains(joined, "--add-host") || strings.Contains(joined, "/haa-pipx") {
		t.Fatal("offline original DATA boundary differs", joined)
	}
	runtimeMount, _ := runtime.mountArgument()
	outputMount, _ := output.mountArgument(false)
	wheelMount, _ := wheels.readerMount()
	expected := pythonResolverCreateArguments("none", nil, []string{"--mount", runtimeMount}, []string{"--mount", outputMount, "--mount", wheelMount})
	if !reflect.DeepEqual(args, expected) {
		t.Fatal("existing fixed boundary changed")
	}
	if _, err := uvFrozenRunCreateArguments(runtime, output, nil); err == nil {
		t.Fatal("absent frozen mirror accepted")
	}
	wheels.seeded = false
	if _, err := uvFrozenRunCreateArguments(runtime, output, wheels); err == nil {
		t.Fatal("unseeded original mirror accepted")
	}
	if _, err := startUVFrozenRunTrace(context.Background(), &recordingObserver{}, strings.Repeat("a", 64)); err == nil {
		t.Fatal("observer without readonly topology accepted")
	}
}
