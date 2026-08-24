package networkpolicy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/netip"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	maxMessageBytes = 8 * 1024
	requestTimeout  = 5 * time.Second
)

type operation string

const (
	createOperation operation = "create"
	verifyOperation operation = "verify"
	removeOperation operation = "remove"
)

type wireRequest struct {
	Operation string   `json:"operation"`
	Session   string   `json:"session"`
	NetworkID string   `json:"network_id"`
	Subnet    string   `json:"subnet"`
	Endpoints []string `json:"endpoints"`
}

type wireResponse struct {
	OK bool `json:"ok"`
}

// Client speaks the bounded typed protocol only. Production construction and
// socket identity validation belong to the Host tooling adapter.
type Client struct{ endpoint string }

func NewClient(endpoint string) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("network policy endpoint is required")
	}
	return &Client{endpoint: endpoint}, nil
}

func (c *Client) Create(ctx context.Context, policy ResolverPolicy) error {
	return c.call(ctx, createOperation, policy)
}

func (c *Client) Verify(ctx context.Context, policy ResolverPolicy) error {
	return c.call(ctx, verifyOperation, policy)
}

func (c *Client) Remove(ctx context.Context, policy ResolverPolicy) error {
	return c.call(ctx, removeOperation, policy)
}

func (c *Client) call(ctx context.Context, op operation, policy ResolverPolicy) error {
	if c == nil || c.endpoint == "" || ctx == nil {
		return errors.New("network policy service is unavailable")
	}
	request, err := encodeRequest(op, policy)
	if err != nil {
		return err
	}
	dialer := net.Dialer{Timeout: requestTimeout}
	connection, err := dialer.DialContext(ctx, "unix", c.endpoint)
	if err != nil {
		return errors.New("network policy service connection failed")
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(requestTimeout)); err != nil {
		return errors.New("network policy service deadline failed")
	}
	if err := json.NewEncoder(connection).Encode(request); err != nil {
		return errors.New("network policy request failed")
	}
	var response wireResponse
	decoder := json.NewDecoder(io.LimitReader(connection, maxMessageBytes+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil || decoder.Decode(&struct{}{}) != io.EOF || !response.OK {
		return errors.New("network policy service rejected request")
	}
	return nil
}

func encodeRequest(op operation, policy ResolverPolicy) (wireRequest, error) {
	if op != createOperation && op != verifyOperation && op != removeOperation {
		return wireRequest{}, errors.New("network policy operation is invalid")
	}
	canonical, err := NewResolverPolicy(policy.Session(), policy.NetworkID(), policy.Subnet(), policy.Endpoints())
	if err != nil {
		return wireRequest{}, err
	}
	endpoints := canonical.Endpoints()
	encoded := make([]string, len(endpoints))
	for index, endpoint := range endpoints {
		encoded[index] = endpoint.String()
	}
	return wireRequest{Operation: string(op), Session: canonical.Session().String(), NetworkID: canonical.NetworkID(), Subnet: canonical.Subnet().String(), Endpoints: encoded}, nil
}

func decodeRequest(body []byte) (operation, ResolverPolicy, error) {
	if len(body) == 0 || len(body) > maxMessageBytes {
		return "", ResolverPolicy{}, errors.New("network policy request is invalid")
	}
	var wire wireRequest
	decoder := json.NewDecoder(io.LimitReader(bytes.NewReader(body), maxMessageBytes+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return "", ResolverPolicy{}, errors.New("network policy request is invalid")
	}
	op := operation(wire.Operation)
	if op != createOperation && op != verifyOperation && op != removeOperation {
		return "", ResolverPolicy{}, errors.New("network policy operation is invalid")
	}
	session, err := domain.ParseSandboxSessionID(wire.Session)
	if err != nil {
		return "", ResolverPolicy{}, errors.New("network policy request is invalid")
	}
	subnet, err := netip.ParsePrefix(wire.Subnet)
	if err != nil {
		return "", ResolverPolicy{}, errors.New("network policy request is invalid")
	}
	endpoints := make([]netip.Addr, len(wire.Endpoints))
	for index, value := range wire.Endpoints {
		address, parseErr := netip.ParseAddr(value)
		if parseErr != nil || address.String() != value {
			return "", ResolverPolicy{}, errors.New("network policy request is invalid")
		}
		endpoints[index] = address
	}
	policy, err := NewResolverPolicy(session, wire.NetworkID, subnet, endpoints)
	if err != nil {
		return "", ResolverPolicy{}, errors.New("network policy request is invalid")
	}
	return op, policy, nil
}
