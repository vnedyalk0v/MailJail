package bastille

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vnedyalk0v/mailjail/internal/host/command"
)

const defaultPrefix = "/usr/local/bastille"

type Client struct {
	Runner command.Runner
	Prefix string
}

func New(runner command.Runner) Client {
	return Client{
		Runner: runner,
		Prefix: defaultPrefix,
	}
}

func (c Client) EnsureRelease(ctx context.Context, release string) error {
	if release == "" {
		return errors.New("release is required")
	}

	if _, err := os.Stat(filepath.Join(c.Prefix, "releases", release)); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat bastille release: %w", err)
	}

	_, err := c.Runner.Run(ctx, command.Spec{
		Path:    "bastille",
		Args:    []string{"bootstrap", release},
		Timeout: 20 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("bootstrap Bastille release %s: %w", release, err)
	}
	return nil
}

func (c Client) EnsureJail(ctx context.Context, name, release, ipCIDR, iface string) error {
	if name == "" || release == "" || ipCIDR == "" {
		return errors.New("name, release, and ipCIDR are required")
	}

	if _, err := os.Stat(filepath.Join(c.Prefix, "jails", name)); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat bastille jail: %w", err)
	}

	args := []string{"create", name, release, ipCIDR}
	if iface != "" {
		args = append(args, iface)
	}

	_, err := c.Runner.Run(ctx, command.Spec{
		Path:    "bastille",
		Args:    args,
		Timeout: 20 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("create Bastille jail %s: %w", name, err)
	}

	return nil
}
