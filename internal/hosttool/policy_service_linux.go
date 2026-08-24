//go:build linux

package hosttool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"

	"github.com/rahoney/heliopause/internal/networkpolicy"
)

const policyServiceConfigPath = "/etc/heliopause/network-policy.json"

type policyServiceConfig struct {
	SocketPath string `json:"socket_path"`
	ClientPath string `json:"client_path"`
	ClientUID  uint32 `json:"client_uid"`
	ClientGID  uint32 `json:"client_gid"`
}

// NewSystemNetworkPolicyClient validates the installed local service socket
// before exposing the narrow typed client to resolver infrastructure.
func NewSystemNetworkPolicyClient() (networkpolicy.Service, error) {
	config, err := loadPolicyServiceConfig()
	if err != nil {
		return nil, err
	}
	if err := verifyPolicySocket(config.SocketPath, config.ClientGID); err != nil {
		return nil, err
	}
	client, err := networkpolicy.NewClient(config.SocketPath)
	if err != nil {
		return nil, err
	}
	return verifiedPolicyClient{config: config, client: client}, nil
}

// verifiedPolicyClient rechecks the protected socket around every request.
// A replacement, permission change or failed helper connection is always a
// resolver failure rather than a reason to continue with stale trust.
type verifiedPolicyClient struct {
	config policyServiceConfig
	client networkpolicy.Service
}

func (c verifiedPolicyClient) Create(ctx context.Context, policy networkpolicy.ResolverPolicy) error {
	return c.call(ctx, policy, c.client.Create)
}
func (c verifiedPolicyClient) Verify(ctx context.Context, policy networkpolicy.ResolverPolicy) error {
	return c.call(ctx, policy, c.client.Verify)
}
func (c verifiedPolicyClient) Remove(ctx context.Context, policy networkpolicy.ResolverPolicy) error {
	return c.call(ctx, policy, c.client.Remove)
}
func (c verifiedPolicyClient) call(ctx context.Context, policy networkpolicy.ResolverPolicy, operation func(context.Context, networkpolicy.ResolverPolicy) error) error {
	if ctx == nil || operation == nil || verifyPolicySocket(c.config.SocketPath, c.config.ClientGID) != nil {
		return errors.New("network policy service identity is unavailable")
	}
	if err := operation(ctx, policy); err != nil {
		return err
	}
	if err := verifyPolicySocket(c.config.SocketPath, c.config.ClientGID); err != nil {
		return errors.New("network policy service identity changed")
	}
	return nil
}

// ServeNetworkPolicy is the root-owned service entry point. It has no CLI
// operation mode other than serving the fixed protected local socket.
func ServeNetworkPolicy(ctx context.Context) error {
	if os.Geteuid() != 0 || ctx == nil {
		return errors.New("network policy helper requires root")
	}
	config, err := loadPolicyServiceConfig()
	if err != nil {
		return err
	}
	if err := verifyPolicySocketParent(config.SocketPath); err != nil {
		return err
	}
	if _, err := os.Lstat(config.SocketPath); !errors.Is(err, os.ErrNotExist) {
		return errors.New("network policy socket is already owned")
	}
	hostConfig, err := loadSystemConfig()
	if err != nil {
		return err
	}
	// Only the root-owned policy service requests firewall identities; the
	// ordinary product executor is always constructed without them.
	executor, err := newExecutor(ctx, hostConfig, true)
	if err != nil {
		return err
	}
	defer executor.Close()
	authorizer, err := NewPolicyPeerAuthorizer(config.ClientPath, config.ClientUID)
	if err != nil {
		return err
	}
	engine, err := NewPolicyEngine(executor)
	if err != nil {
		return err
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: config.SocketPath, Net: "unix"})
	if err != nil {
		return errors.New("bind network policy socket")
	}
	info, err := os.Lstat(config.SocketPath)
	if err != nil || info.Mode()&os.ModeSocket == 0 || os.Chmod(config.SocketPath, 0o660) != nil || os.Chown(config.SocketPath, 0, int(config.ClientGID)) != nil {
		_ = listener.Close()
		return errors.New("protect network policy socket")
	}
	server, err := networkpolicy.NewServer(listener, authorizer, engine)
	if err != nil {
		_ = listener.Close()
		return err
	}
	defer func(socketID os.FileInfo) {
		current, err := os.Lstat(config.SocketPath)
		if err == nil && os.SameFile(socketID, current) {
			_ = os.Remove(config.SocketPath)
		}
	}(info)
	return server.Serve(ctx)
}

func loadPolicyServiceConfig() (policyServiceConfig, error) {
	verified, err := verifyExecutable(policyServiceConfigPath, "")
	if err != nil {
		return policyServiceConfig{}, errors.New("network policy service configuration is unavailable")
	}
	file, err := os.Open(verified.path)
	if err != nil {
		return policyServiceConfig{}, errors.New("open network policy service configuration")
	}
	body, readErr := io.ReadAll(io.LimitReader(file, 8*1024+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(body) > 8*1024 {
		return policyServiceConfig{}, errors.New("read network policy service configuration")
	}
	var config policyServiceConfig
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&config) != nil || decoder.Decode(&struct{}{}) != io.EOF || !filepath.IsAbs(config.SocketPath) || filepath.Clean(config.SocketPath) != config.SocketPath || config.ClientPath == "" {
		return policyServiceConfig{}, errors.New("parse network policy service configuration")
	}
	return config, nil
}

func verifyPolicySocket(path string, group uint32) error {
	if err := verifyPolicySocketParent(path); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSocket == 0 || info.Mode()&0o007 != 0 {
		return errors.New("network policy socket is untrusted")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 || stat.Gid != group {
		return errors.New("network policy socket identity is untrusted")
	}
	return nil
}

func verifyPolicySocketParent(path string) error { return verifyTrustedParents(filepath.Dir(path)) }
