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

func (c Client) InstallPackages(ctx context.Context, jail string, packages []string) error {
	if jail == "" {
		return errors.New("jail is required")
	}
	if len(packages) == 0 {
		return errors.New("at least one package is required")
	}

	args := []string{"pkg", jail, "install", "-y"}
	args = append(args, packages...)

	_, err := c.Runner.Run(ctx, command.Spec{
		Path:    "bastille",
		Args:    args,
		Timeout: 20 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("install packages in jail %s: %w", jail, err)
	}
	return nil
}

func (c Client) SetRC(ctx context.Context, jail, setting string) error {
	if jail == "" {
		return errors.New("jail is required")
	}
	if setting == "" {
		return errors.New("setting is required")
	}

	_, err := c.Runner.Run(ctx, command.Spec{
		Path:    "bastille",
		Args:    []string{"sysrc", jail, setting},
		Timeout: 2 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("set sysrc in jail %s: %w", jail, err)
	}
	return nil
}

func (c Client) EnsureServiceStarted(ctx context.Context, jail, service string) error {
	if jail == "" {
		return errors.New("jail is required")
	}
	if service == "" {
		return errors.New("service is required")
	}

	_, statusErr := c.Runner.Run(ctx, command.Spec{
		Path:    "bastille",
		Args:    []string{"cmd", jail, "service", service, "onestatus"},
		Timeout: 30 * time.Second,
	})
	if statusErr == nil {
		return nil
	}

	_, startErr := c.Runner.Run(ctx, command.Spec{
		Path:    "bastille",
		Args:    []string{"service", jail, service, "start"},
		Timeout: 2 * time.Minute,
	})
	if startErr != nil {
		return fmt.Errorf("start service %s in jail %s: %w", service, jail, startErr)
	}

	return nil
}
