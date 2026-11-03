//go:build linux && realcorpus

package sandbox

import (
	"os"
	"testing"
)

// Literal-address, bounded DNS-wire and HTTP DATA only. The existing Python
// Artifact lane supplies isolation/accounting; no trusted responder role is proven.
func TestLoopbackResponderDATAIntegration(t *testing.T) {
	if os.Getenv("HELOX_RESPONDER_PRIMITIVE") != "1" {
		t.Skip("explicit pinned guest loopback DATA primitive")
	}
	const program = `import socket
ip = '127.0.0.1'
query = bytes.fromhex('123401000001000000000000') + b'\x08artifact\x05helox\x07invalid\x00\x00\x01\x00\x01'
answer = bytes.fromhex('123481800001000100000000') + query[12:] + bytes.fromhex('c00c000100010000000100047f000001')
with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as server, socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as client:
    server.settimeout(1); client.settimeout(1)
    server.bind((ip, 0)); client.bind((ip, 0))
    assert client.sendto(query, server.getsockname()) == len(query)
    data, peer = server.recvfrom(512)
    assert data == query
    assert server.sendto(answer, peer) == len(answer)
    response = client.recv(512)
    assert response == answer
    resolved = socket.inet_ntoa(response[-4:])
    assert resolved == ip
request = b'GET /probe HTTP/1.0\r\n\r\n'
reply = b'HTTP/1.0 200 OK\r\nContent-Length: 2\r\n\r\nok'
with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server, socket.socket(socket.AF_INET, socket.SOCK_STREAM) as client:
    server.settimeout(1); client.settimeout(1)
    server.bind((ip, 0)); server.listen(1)
    client.connect((resolved, server.getsockname()[1]))
    connection, _ = server.accept()
    with connection:
        connection.settimeout(1)
        client.sendall(request)
        received = b''
        while len(received) < len(request):
            part = connection.recv(len(request) - len(received))
            assert part
            received += part
        assert received == request
        connection.sendall(reply)
    received = b''
    while len(received) < len(reply):
        part = client.recv(len(reply) - len(received))
        assert part
        received += part
    assert received == reply
`
	result := runLinuxPyPIWheelIntegration(t, linuxDynamicWheelWithProgram(t, program))
	connections := uint64(0)
	for _, fact := range result.Observations() {
		if fact.Subject() == "network-attempt" {
			connections += fact.Count()
		}
	}
	states := result.EnvironmentStates()
	if connections < 2 || len(states) != 2 || !states[0].VFSOperationsCovered || !states[1].VFSOperationsCovered {
		t.Fatal("two isolated DATA exchanges were not observed")
	}
	t.Logf("literal-address Python fixture: loopback DNS-wire-to-HTTP in two network-none environments; observed connections=%d; no default resolver, trusted responder role or Policy claim", connections)
}
