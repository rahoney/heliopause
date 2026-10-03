package gomodule

import (
	"testing"
)

func TestGoResolutionDigestBindsContentAndControlsWithoutPrivatePaths(t *testing.T) {
	record := DownloadRecord{Path: "example.com/mod", Version: "v1.2.3", GoMod: "/one/private/mod.mod", Zip: "/one/private/mod.zip", Sum: testH1('a'), GoModSum: testH1('b')}
	mod := []byte("module local-app\nrequire example.com/mod v1.2.3\n")
	sum := []byte("fixture-sum\n")
	graph := []byte("local-app example.com/mod@v1.2.3\n")
	first, err := FreezeResolutionDigest([]DownloadRecord{record}, graph, mod, sum)
	if err != nil {
		t.Fatal(err)
	}
	record.GoMod, record.Zip, record.Info = "/two/private/mod.mod", "/two/private/mod.zip", "/two/private/mod.info"
	second, err := FreezeResolutionDigest([]DownloadRecord{record}, graph, mod, sum)
	if err != nil || second != first {
		t.Fatalf("private cache paths changed frozen identity: %v", err)
	}
	for _, member := range []string{"zip sum", "mod sum", "go.mod", "go.sum", "graph"} {
		t.Run(member, func(t *testing.T) {
			changed := record
			modBody, sumBody, graphBody := mod, sum, graph
			switch member {
			case "zip sum":
				changed.Sum = testH1('c')
			case "mod sum":
				changed.GoModSum = testH1('d')
			case "go.mod":
				modBody = append(append([]byte{}, mod...), []byte("// changed controls\n")...)
			case "go.sum":
				sumBody = []byte("changed-sum\n")
			case "graph":
				graphBody = []byte("local-app  example.com/mod@v1.2.3\n")
			}
			digest, err := FreezeResolutionDigest([]DownloadRecord{changed}, graphBody, modBody, sumBody)
			if err != nil || digest == first {
				t.Fatalf("changed %s was not bound: %v", member, err)
			}
		})
	}
}
