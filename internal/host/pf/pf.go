package pf

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vnedyalk0v/mailjail/internal/config"
	"github.com/vnedyalk0v/mailjail/internal/host/command"
	"github.com/vnedyalk0v/mailjail/internal/modules"
	"github.com/vnedyalk0v/mailjail/internal/topology"
)

const anchorRootDir = "/etc/pf.anchors/mailjail"

type Client struct {
	Runner command.Runner
}

func New(runner command.Runner) Client {
	return Client{Runner: runner}
}

func AnchorName(cfg *config.Config) string {
	return "mailjail/" + cfg.Metadata.Name
}

func AnchorFilePath(cfg *config.Config) string {
	return filepath.Join(anchorRootDir, cfg.Metadata.Name+".conf")
}

func (c Client) EnsureAnchor(ctx context.Context, cfg *config.Config) error {
	content, err := Render(cfg)
	if err != nil {
		return err
	}

	anchorFile := AnchorFilePath(cfg)
	if err := os.MkdirAll(filepath.Dir(anchorFile), 0o755); err != nil {
		return fmt.Errorf("create PF anchor directory: %w", err)
	}
	if err := writeFileAtomically(anchorFile, []byte(content), 0o640); err != nil {
		return fmt.Errorf("write PF anchor file: %w", err)
	}

	anchorName := AnchorName(cfg)
	validateArgs := []string{"-n", "-a", anchorName, "-f", anchorFile}
	if _, err := c.Runner.Run(ctx, command.Spec{
		Path:    "pfctl",
		Args:    validateArgs,
		Timeout: 30 * time.Second,
	}); err != nil {
		return fmt.Errorf("validate PF anchor %s: %w", anchorName, err)
	}

	if _, err := c.Runner.Run(ctx, command.Spec{
		Path:    "pfctl",
		Args:    []string{"-a", anchorName, "-f", anchorFile},
		Timeout: 30 * time.Second,
	}); err != nil {
		return fmt.Errorf("load PF anchor %s: %w", anchorName, err)
	}

	return nil
}

func Render(cfg *config.Config) (string, error) {
	baseJailIP, err := topology.DeriveBaseJailIP(cfg)
	if err != nil {
		return "", fmt.Errorf("derive base jail ip: %w", err)
	}

	enabled, err := modules.Enabled(cfg)
	if err != nil {
		return "", err
	}

	moduleIPs := make([]string, 0, len(enabled))
	moduleIPByName := make(map[string]netip.Addr, len(enabled))
	for _, module := range enabled {
		moduleIPs = append(moduleIPs, module.IP.String())
		moduleIPByName[module.Definition.Name] = module.IP
	}
	sort.Strings(moduleIPs)

	var builder strings.Builder
	builder.WriteString("# Managed by MailJail. Do not edit manually.\n")
	_, _ = fmt.Fprintf(&builder, "# Stack: %s\n\n", cfg.Metadata.Name)
	_, _ = fmt.Fprintf(&builder, "table <mailjail_nodes> const { %s }\n", strings.Join(moduleIPs, ", "))
	_, _ = fmt.Fprintf(&builder, "table <mailjail_base> const { %s }\n\n", baseJailIP.String())

	_, _ = fmt.Fprintf(&builder, "pass quick on %s inet from %s to <mailjail_nodes> keep state\n", cfg.Network.Bridge, cfg.Network.Gateway4)
	_, _ = fmt.Fprintf(&builder, "pass quick on %s inet from <mailjail_nodes> to %s keep state\n", cfg.Network.Bridge, cfg.Network.Gateway4)
	_, _ = fmt.Fprintf(&builder, "pass quick on %s inet from %s to <mailjail_base> keep state\n", cfg.Network.Bridge, cfg.Network.Gateway4)
	_, _ = fmt.Fprintf(&builder, "pass quick on %s inet from <mailjail_base> to %s keep state\n", cfg.Network.Bridge, cfg.Network.Gateway4)
	_, _ = fmt.Fprintf(&builder, "pass quick on %s inet from <mailjail_base> to <mailjail_nodes> keep state\n", cfg.Network.Bridge)
	_, _ = fmt.Fprintf(&builder, "pass quick on %s inet from <mailjail_nodes> to <mailjail_base> keep state\n", cfg.Network.Bridge)

	connections := modules.DependencyConnections(cfg)
	if len(connections) > 0 {
		builder.WriteString("\n# Allowed inter-module flows\n")
		for _, connection := range connections {
			fromIP, ok := moduleIPByName[connection.From]
			if !ok {
				return "", fmt.Errorf("missing IP for source module %s", connection.From)
			}
			toIP, ok := moduleIPByName[connection.To]
			if !ok {
				return "", fmt.Errorf("missing IP for destination module %s", connection.To)
			}
			_, _ = fmt.Fprintf(&builder,
				"pass quick on %s inet proto tcp from %s to %s keep state # %s -> %s\n",
				cfg.Network.Bridge,
				fromIP.String(),
				toIP.String(),
				connection.From,
				connection.To,
			)
		}
	}

	builder.WriteString("\n")
	_, _ = fmt.Fprintf(&builder, "block drop quick on %s inet from <mailjail_nodes> to <mailjail_nodes>\n", cfg.Network.Bridge)
	_, _ = fmt.Fprintf(&builder, "block drop quick on %s inet from <mailjail_nodes> to <mailjail_base>\n", cfg.Network.Bridge)
	_, _ = fmt.Fprintf(&builder, "block drop quick on %s inet from <mailjail_base> to <mailjail_nodes>\n", cfg.Network.Bridge)

	return builder.String(), nil
}

func writeFileAtomically(path string, data []byte, mode os.FileMode) error {
	if path == "" {
		return errors.New("path is required")
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, path)
}
