//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/releaseinstall"
)

const (
	policyUnitName     = "haa-network-policy-helper.service"
	policySocketPath   = "/run/heliopause-network-policy/network-policy.sock"
	resourceSocketPath = "/run/heliopause-network-policy/observation.sock"
	policyControlGroup = "/system.slice/" + policyUnitName
)

type nativePolicyConfig struct {
	SocketPath string `json:"socket_path"`
	ClientPath string `json:"client_path"`
	ClientUID  uint32 `json:"client_uid"`
	ClientGID  uint32 `json:"client_gid"`
}

func nativeVersionPath(version, name string) string {
	return filepath.Join(releaseinstall.SystemInstallRoot, "versions", version, name)
}

// nativePolicyUnit is the exact unit explicitly owned by native bootstrap.
// Package/foreign units and drop-ins are not silently adopted by an update.
func nativePolicyUnit(version string) ([]byte, error) {
	if !exactVersion(version) {
		return nil, ErrUntrusted
	}
	return []byte("[Unit]\nDescription=Heliopause typed network-policy helper\nAfter=docker.service\nConditionPathExists=" + policyConfigPath + "\n\n" +
		"[Service]\nType=simple\nExecStart=" + nativeVersionPath(version, "haa-network-policy-helper-linux-amd64") + "\nUser=root\nGroup=root\nUMask=0077\n" +
		"RuntimeDirectory=heliopause-network-policy\nRuntimeDirectoryMode=0755\nRuntimeDirectoryPreserve=yes\nNoNewPrivileges=yes\nRestart=no\nKillMode=control-group\nTimeoutStopSec=15s\n\n[Install]\nWantedBy=multi-user.target\n"), nil
}

func parseNativeConfigs(journal updateJournal) (nativePolicyConfig, hosttool.Config, error) {
	var policy nativePolicyConfig
	var host hosttool.Config
	if err := journal.validate(); err != nil {
		return policy, host, err
	}
	if err := canonicalConfig(journal.Configs[0].Body, &policy); err != nil {
		return policy, host, err
	}
	if err := canonicalConfig(journal.Configs[1].Body, &host); err != nil {
		return policy, host, err
	}
	if policy.SocketPath != policySocketPath || policy.ClientPath != nativeVersionPath(journal.Old.Version, "helox-linux-amd64") || policy.ClientUID != journal.Old.ClientUID || policy.ClientGID != journal.Old.ClientGID || host.ObserverHelperPath != nativeVersionPath(journal.Old.Version, "haa_gvisor_observer-linux-amd64") {
		return policy, host, ErrUntrusted
	}
	if host.DockerPath != "/usr/bin/docker" && host.DockerPath != "/usr/local/bin/docker" {
		return policy, host, ErrUntrusted
	}
	endpoint := strings.TrimPrefix(host.DockerEndpoint, "unix://")
	if host.DockerEndpoint != "unix://"+endpoint || !filepath.IsAbs(endpoint) || filepath.Clean(endpoint) != endpoint || strings.ContainsAny(endpoint, "\x00\r\n") {
		return policy, host, ErrUntrusted
	}
	unit, err := nativePolicyUnit(journal.Old.Version)
	if err != nil || !bytes.Equal(unit, journal.Configs[2].Body) {
		return policy, host, ErrUntrusted
	}
	return policy, host, nil
}

func canonicalConfig(body []byte, value any) error {
	if len(body) == 0 || len(body) > 16<<10 || json.Unmarshal(body, value) != nil {
		return ErrUntrusted
	}
	canonical, err := json.Marshal(value)
	if err != nil || !bytes.Equal(body, canonical) {
		return ErrUntrusted
	}
	return nil
}

func derivedRuntimeConfigs(journal updateJournal) ([3][]byte, error) {
	var bodies [3][]byte
	policy, host, err := parseNativeConfigs(journal)
	if err != nil {
		return bodies, err
	}
	policy.ClientPath = nativeVersionPath(journal.Version, "helox-linux-amd64")
	host.ObserverHelperPath = nativeVersionPath(journal.Version, "haa_gvisor_observer-linux-amd64")
	bodies[0], err = json.Marshal(policy)
	if err != nil {
		return bodies, ErrUntrusted
	}
	bodies[1], err = json.Marshal(host)
	if err != nil {
		return bodies, ErrUntrusted
	}
	bodies[2], err = nativePolicyUnit(journal.Version)
	return bodies, err
}

func switchRuntimeConfigs(ctx context.Context, journal updateJournal, restore bool) error {
	derived, err := derivedRuntimeConfigs(journal)
	if err != nil {
		return err
	}
	for index, path := range configPaths() {
		saved := journal.Configs[index]
		expected, replacement := saved.Body, derived[index]
		if restore {
			actual, err := readProtected(ctx, path, 16<<10)
			if err != nil {
				return err
			}
			if bytes.Equal(actual, saved.Body) {
				continue
			}
			if !bytes.Equal(actual, derived[index]) {
				return ErrUntrusted
			}
			expected, replacement = derived[index], saved.Body
		}
		if err := replaceProtected(ctx, path, expected, replacement, os.FileMode(saved.Mode), saved.GID); err != nil {
			return errors.Join(ErrTransition, err)
		}
	}
	return verifyRuntimeConfigs(ctx, journal, restore)
}

func verifyRuntimeConfigs(ctx context.Context, journal updateJournal, old bool) error {
	derived, err := derivedRuntimeConfigs(journal)
	if err != nil {
		return err
	}
	for index, path := range configPaths() {
		expected := derived[index]
		if old {
			expected = journal.Configs[index].Body
		}
		actual, err := snapshotConfig(ctx, path)
		if err != nil || !bytes.Equal(actual.Body, expected) || actual.Mode != journal.Configs[index].Mode || actual.GID != journal.Configs[index].GID {
			return errors.Join(ErrUntrusted, err)
		}
	}
	return nil
}
